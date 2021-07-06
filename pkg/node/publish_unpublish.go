// This file is part of MinIO Direct CSI
// Copyright (c) 2021 MinIO, Inc.
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

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	"github.com/minio/direct-csi/pkg/utils"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
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

func (n *NodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	klog.V(3).Infof("NodePublishVolumeRequest: %v", req)
	vID := req.GetVolumeId()
	if vID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}
	stagingTargetPath := req.GetStagingTargetPath()
	if stagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "stagingTargetPath missing in request")
	}

	containerPath := req.GetTargetPath()
	if containerPath == "" {
		return nil, status.Error(codes.InvalidArgument, "containerPath missing in request")
	}

	readOnly := req.GetReadonly()
	directCSIClient := n.directcsiClient.DirectV1beta2()
	vclient := directCSIClient.DirectCSIVolumes()

	vol, err := vclient.Get(ctx, vID, metav1.GetOptions{
		TypeMeta: utils.DirectCSIVolumeTypeMeta(),
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	// If not staged
	if vol.Status.StagingPath != stagingTargetPath {
		return nil, status.Error(codes.Internal, "cannot publish volume that hasn't been staged")
	}

	extractPodLabels := func() map[string]string {
		volumeContext, volumeLabels := req.GetVolumeContext(), vol.ObjectMeta.GetLabels()
		if volumeLabels == nil {
			volumeLabels = make(map[string]string)
		}

		podName, podNs, parseErr := parseVolumeContext(volumeContext)
		if parseErr != nil {
			klog.V(5).Infof("Failed to parse the volume context: %v", parseErr)
			return nil
		}

		pod, err := utils.GetKubeClient().CoreV1().Pods(podNs).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			klog.V(5).Infof("Failed to extract pod labels: %v", err)
			return nil
		}

		podLabels := pod.ObjectMeta.GetLabels()
		for k, v := range podLabels {
			if strings.HasPrefix(k, directcsi.Group+"/") {
				volumeLabels[k] = v
			}
		}

		volumeLabels[directcsi.Group+"/pod.name"] = podName
		volumeLabels[directcsi.Group+"/pod.namespace"] = podNs
		return volumeLabels
	}

	volLabels := extractPodLabels()
	if volLabels != nil {
		vol.ObjectMeta.Labels = volLabels
	}

	if err := os.MkdirAll(containerPath, 0755); err != nil {
		return nil, err
	}

	if err := n.mounter.MountVolume(ctx, stagingTargetPath, containerPath, vID, 0, readOnly); err != nil {
		return nil, status.Errorf(codes.Internal, "failed volume publish: %v", err)
	}

	conditions := vol.Status.Conditions
	for i, c := range conditions {
		switch c.Type {
		case string(directcsi.DirectCSIVolumeConditionPublished):
			conditions[i].Status = utils.BoolToCondition(true)
			conditions[i].Reason = string(directcsi.DirectCSIVolumeReasonInUse)
		case string(directcsi.DirectCSIVolumeConditionStaged):
		case string(directcsi.DirectCSIVolumeConditionReady):
		}
	}
	vol.Status.ContainerPath = containerPath

	if _, err := vclient.Update(ctx, vol, metav1.UpdateOptions{
		TypeMeta: utils.DirectCSIVolumeTypeMeta(),
	}); err != nil {
		return nil, err
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (n *NodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	klog.V(3).Infof("NodeUnPublishVolumeRequest: %v", req)
	vID := req.GetVolumeId()
	if vID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}
	containerPath := req.GetTargetPath()
	if containerPath == "" {
		return nil, status.Error(codes.InvalidArgument, "containerPath missing in request")
	}

	directCSIClient := n.directcsiClient.DirectV1beta2()
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

	if err := n.mounter.UnmountVolume(containerPath); err != nil {
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
