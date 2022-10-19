// This file is part of MinIO DirectPV
// Copyright (c) 2021, 2022 MinIO, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package node

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/container-storage-interface/spec/lib/go/csi"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/drive"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

const (
	podNameKey      = "csi.storage.k8s.io/pod.name"
	podNamespaceKey = "csi.storage.k8s.io/pod.namespace"
)

func parseVolumeContext(volumeContext map[string]string) (name, ns string, err error) {
	parseValue := func(key string) (string, error) {
		value, ok := volumeContext[key]
		if !ok {
			return "", fmt.Errorf("required volume context key %v not found", key)
		}
		return value, nil
	}

	name, err = parseValue(podNameKey)
	if err != nil {
		return "", "", err
	}

	ns, err = parseValue(podNamespaceKey)
	if err != nil {
		return "", "", err
	}

	return
}

func getPodInfo(ctx context.Context, req *csi.NodePublishVolumeRequest) (podName, podNS string, podLabels map[string]string) {
	var err error
	if podName, podNS, err = parseVolumeContext(req.GetVolumeContext()); err != nil {
		klog.ErrorS(err, "unable to parse volume context", "context", req.GetVolumeContext(), "volume", req.GetVolumeId())
		return
	}

	if pod, err := k8s.KubeClient().CoreV1().Pods(podNS).Get(ctx, podName, metav1.GetOptions{}); err != nil {
		klog.ErrorS(err, "unable to get pod information", "name", podName, "namespace", podNS)
	} else {
		podLabels = pod.GetLabels()
	}

	return
}

// NodePublishVolume is node publish volume request handler.
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#nodepublishvolume
func (server *Server) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	klog.V(3).InfoS("Publish volume requested",
		"volumeID", req.GetVolumeId(),
		"stagingTargetPath", req.GetStagingTargetPath(),
		"targetPath", req.GetTargetPath())

	if req.GetVolumeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID must not be empty")
	}

	if req.GetStagingTargetPath() == "" {
		return nil, status.Error(codes.InvalidArgument, "Staging target path must not be empty")
	}

	if req.GetTargetPath() == "" {
		return nil, status.Error(codes.InvalidArgument, "target path must not be empty")
	}

	podName, podNS, podLabels := getPodInfo(ctx, req)

	volume, err := client.VolumeClient().Get(ctx, req.GetVolumeId(), metav1.GetOptions{TypeMeta: types.NewVolumeTypeMeta()})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if volume.Status.StagingTargetPath != req.GetStagingTargetPath() {
		return nil, status.Errorf(codes.FailedPrecondition, "volume %v is not yet staged, but requested with %v", volume.Name, req.GetStagingTargetPath())
	}

	volume.SetPodName(podName)
	volume.SetPodNS(podNS)
	for key, value := range podLabels {
		if strings.HasPrefix(key, consts.GroupName+"/") {
			volume.SetLabel(directpvtypes.LabelKey(key), directpvtypes.LabelValue(value))
		}
	}

	mountPointMap, err := server.getMounts()
	if err != nil {
		klog.ErrorS(err, "unable to get mounts")
		return nil, status.Error(codes.Internal, err.Error())
	}
	if _, found := mountPointMap[req.GetStagingTargetPath()]; !found {
		klog.Errorf("stagingPath %v is not mounted", req.GetStagingTargetPath())
		return nil, status.Error(codes.Internal, fmt.Sprintf("stagingPath %v is not mounted", req.GetStagingTargetPath()))
	}

	if err := server.mkdir(req.GetTargetPath()); err != nil && !errors.Is(err, os.ErrExist) {
		if errors.Unwrap(err) == syscall.EIO {
			if err := drive.SetIOError(ctx, volume.GetDriveID()); err != nil {
				return nil, status.Errorf(codes.Internal, "unable to set drive error; %v", err)
			}
		}
		klog.ErrorS(err, "unable to create target path", "TargetPath", req.GetTargetPath())
		return nil, status.Errorf(codes.Internal, "unable to create target path: %v", err)
	}

	if err := server.bindMount(req.GetStagingTargetPath(), req.GetTargetPath(), req.GetReadonly()); err != nil {
		klog.ErrorS(err, "unable to bind mount staging target path to target path", "StagingTargetPath", req.GetStagingTargetPath(), "TargetPath", req.GetTargetPath())
		return nil, status.Errorf(codes.Internal, "unable to bind mount staging target path to target path; %v", err)
	}

	volume.Status.TargetPath = req.GetTargetPath()
	_, err = client.VolumeClient().Update(ctx, volume, metav1.UpdateOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to update volume: %v", err)
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnpublishVolume is node unpublish volume handler.
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#nodeunpublishvolume
func (server *Server) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	klog.V(3).InfoS("Unpublish volume requested",
		"volumeID", req.GetVolumeId(),
		"targetPath", req.GetTargetPath())
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}
	targetPath := req.GetTargetPath()
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "targetPath missing in request")
	}

	volume, err := client.VolumeClient().Get(ctx, volumeID, metav1.GetOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return &csi.NodeUnpublishVolumeResponse{}, nil
		}
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if !volume.IsPublished() {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("unpublish is called without publish for volume %v", volume.Name))
	}

	if volume.Status.TargetPath != targetPath {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("target path %v doesn't match with requested target path %v", volume.Status.TargetPath, targetPath))
	}

	if err := server.unmount(targetPath); err != nil {
		klog.ErrorS(err, "unable to unmount target path", "TargetPath", targetPath)
		return nil, status.Error(codes.Internal, err.Error())
	}

	volume.Status.TargetPath = ""
	if _, err := client.VolumeClient().Update(ctx, volume, metav1.UpdateOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	}); err != nil {
		return nil, err
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}
