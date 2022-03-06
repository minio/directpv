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
	"fmt"
	"os"
	"strings"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/utils"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
			return "", fmt.Errorf("required volume context key not found: %v", key)
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
		klog.Errorf("unable to parse volume context %v for volume %v; %v", req.GetVolumeContext(), req.GetVolumeId(), err)
		return
	}

	if pod, err := client.GetKubeClient().CoreV1().Pods(podNS).Get(ctx, podName, metav1.GetOptions{}); err != nil {
		klog.Errorf("unable to get pod information; name=%v, namespace=%v; %v", podName, podNS, err)
	} else {
		podLabels = pod.GetLabels()
	}

	return
}

// NodePublishVolume is node publish volume request handler.
func (n *NodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	klog.V(3).InfoS("NodePublishVolumeRequest",
		"volumeID", req.GetVolumeId(),
		"stagingTargetPath", req.GetStagingTargetPath(),
		"containerPath", req.GetTargetPath())

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

	volumeInterface := n.directcsiClient.DirectV1beta4().DirectCSIVolumes()

	vol, err := volumeInterface.Get(ctx, req.GetVolumeId(), metav1.GetOptions{TypeMeta: utils.DirectCSIVolumeTypeMeta()})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if vol.Status.StagingPath != req.GetStagingTargetPath() {
		return nil, status.Errorf(codes.FailedPrecondition, "volume %v is not yet staged, but requested with %v", vol.Name, req.GetStagingTargetPath())
	}

	volumeLabels := vol.GetLabels()
	if volumeLabels == nil {
		volumeLabels = make(map[string]string)
	}
	volumeLabels[string(utils.PodNameLabelKey)] = podName
	volumeLabels[string(utils.PodNSLabelKey)] = podNS
	for key, value := range podLabels {
		if strings.HasPrefix(key, directcsi.Group+"/") {
			volumeLabels[key] = value
		}
	}
	vol.Labels = volumeLabels

	if err := checkStagingTargetPath(req.GetStagingTargetPath(), n.probeMounts); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := os.MkdirAll(req.GetTargetPath(), 0755); err != nil {
		return nil, err
	}

	if err := n.safeBindMount(req.GetStagingTargetPath(), req.GetTargetPath(), false, req.GetReadonly()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed volume publish: %v", err)
	}

	conditions := vol.Status.Conditions
	for i, c := range conditions {
		if c.Type == string(directcsi.DirectCSIVolumeConditionPublished) {
			conditions[i].Status = utils.BoolToCondition(true)
			conditions[i].Reason = string(directcsi.DirectCSIVolumeReasonInUse)
		}
	}
	vol.Status.ContainerPath = req.GetTargetPath()

	_, err = volumeInterface.Update(ctx, vol, metav1.UpdateOptions{
		TypeMeta: utils.DirectCSIVolumeTypeMeta(),
	})
	if err != nil {
		return nil, err
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnpublishVolume is node unpublish volume handler.
func (n *NodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	klog.V(3).InfoS("NodeUnPublishVolumeRequest",
		"volumeID", req.GetVolumeId(),
		"ContainerPath", req.GetTargetPath())
	vID := req.GetVolumeId()
	if vID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}
	containerPath := req.GetTargetPath()
	if containerPath == "" {
		return nil, status.Error(codes.InvalidArgument, "containerPath missing in request")
	}

	directCSIClient := n.directcsiClient.DirectV1beta4()
	vclient := directCSIClient.DirectCSIVolumes()
	vol, err := vclient.Get(ctx, vID, metav1.GetOptions{
		TypeMeta: utils.DirectCSIVolumeTypeMeta(),
	})
	if err != nil {
		if errors.IsNotFound(err) {
			return &csi.NodeUnpublishVolumeResponse{}, nil
		}
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if err := n.safeUnmount(containerPath, true, true, false); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	conditions := vol.Status.Conditions
	for i, c := range conditions {
		switch c.Type {
		case string(directcsi.DirectCSIVolumeConditionPublished):
			conditions[i].Status = utils.BoolToCondition(false)
			conditions[i].Reason = string(directcsi.DirectCSIVolumeReasonNotInUse)
		case string(directcsi.DirectCSIVolumeConditionStaged):
		case string(directcsi.DirectCSIVolumeConditionReady):
		}
	}
	vol.Status.ContainerPath = ""

	if _, err := vclient.Update(ctx, vol, metav1.UpdateOptions{
		TypeMeta: utils.DirectCSIVolumeTypeMeta(),
	}); err != nil {
		return nil, err
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}
