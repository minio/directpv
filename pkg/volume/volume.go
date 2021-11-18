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

package volume

import (
	"context"
	"fmt"
	"os"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/client"
	"github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/listener"
	"github.com/minio/direct-csi/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"

	"k8s.io/klog/v2"
)

type volumeEventHandler struct {
	kubeClient      kubernetes.Interface
	directCSIClient clientset.Interface
	nodeID          string
}

func newVolumeEventHandler(nodeID string) *volumeEventHandler {
	return &volumeEventHandler{
		directCSIClient: client.GetDirectClientset(),
		kubeClient:      client.GetKubeClient(),
		nodeID:          nodeID,
	}
}

func (handler *volumeEventHandler) ListerWatcher() cache.ListerWatcher {
	labelSelector := ""
	if handler.nodeID != "" {
		labelSelector = fmt.Sprintf("%s=%s", utils.NodeLabelKey, utils.NewLabelValue(handler.nodeID))
	}

	optionsModifier := func(options *metav1.ListOptions) {
		options.LabelSelector = labelSelector
	}

	return cache.NewFilteredListWatchFromClient(
		handler.directCSIClient.DirectV1beta3().RESTClient(),
		"DirectCSIVolumes",
		"",
		optionsModifier,
	)
}

func (handler *volumeEventHandler) KubeClient() kubernetes.Interface {
	return handler.kubeClient
}

func (handler *volumeEventHandler) Name() string {
	return "volume"
}

func (handler *volumeEventHandler) ObjectType() runtime.Object {
	return &directcsi.DirectCSIVolume{}
}

func (handler *volumeEventHandler) releaseVolume(ctx context.Context, driveName, volumeName string, capacity int64) error {
	drive, err := handler.directCSIClient.DirectV1beta3().DirectCSIDrives().Get(
		ctx, driveName, metav1.GetOptions{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
		},
	)
	if err != nil {
		return err
	}

	finalizers, found := utils.ExcludeFinalizer(
		drive.GetFinalizers(), directcsi.DirectCSIDriveFinalizerPrefix+volumeName,
	)

	if found {
		if len(finalizers) == 1 {
			if finalizers[0] == directcsi.DirectCSIDriveFinalizerDataProtection {
				drive.Status.DriveStatus = directcsi.DriveStatusReady
			}
		}

		drive.SetFinalizers(finalizers)
		drive.Status.FreeCapacity += capacity
		drive.Status.AllocatedCapacity = drive.Status.TotalCapacity - drive.Status.FreeCapacity

		_, err = handler.directCSIClient.DirectV1beta3().DirectCSIDrives().Update(
			ctx, drive, metav1.UpdateOptions{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
			},
		)
	}

	return err
}

func (handler *volumeEventHandler) delete(ctx context.Context, volume *directcsi.DirectCSIVolume) error {
	finalizers, _ := utils.ExcludeFinalizer(
		volume.GetFinalizers(), string(directcsi.DirectCSIVolumeFinalizerPurgeProtection),
	)
	if len(finalizers) > 0 {
		return fmt.Errorf("waiting for the volume to be released before cleaning up")
	}

	// Remove associated directory of the volume.
	if err := os.RemoveAll(volume.Status.HostPath); err != nil {
		return err
	}

	// Release volume from associated drive.
	if err := handler.releaseVolume(ctx, volume.Status.Drive, volume.Name, volume.Status.TotalCapacity); err != nil {
		return err
	}

	volume.SetFinalizers(finalizers)
	_, err := handler.directCSIClient.DirectV1beta3().DirectCSIVolumes().Update(
		ctx, volume, metav1.UpdateOptions{
			TypeMeta: utils.DirectCSIVolumeTypeMeta(),
		},
	)

	return err
}

func (handler *volumeEventHandler) Handle(ctx context.Context, args listener.EventArgs) error {
	if args.Event == listener.DeleteEvent {
		return handler.delete(ctx, args.Object.(*directcsi.DirectCSIVolume))
	}

	return nil
}

// StartController starts volume controller.
func StartController(ctx context.Context, nodeID string) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	listener := listener.NewListener(newVolumeEventHandler(nodeID), "volume-controller", hostname, 40)
	return listener.Run(ctx)
}

func getLabels(ctx context.Context, volume *directcsi.DirectCSIVolume) map[string]string {
	drive, err := client.GetDirectCSIClient().DirectCSIDrives().Get(
		ctx, volume.Status.Drive, metav1.GetOptions{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
		},
	)

	var driveName, drivePath string
	if err == nil {
		finalizer := directcsi.DirectCSIDriveFinalizerPrefix + volume.Name
		for _, f := range drive.GetFinalizers() {
			if f == finalizer {
				driveName, drivePath = drive.Name, drive.Status.Path
				break
			}
		}
	}

	labels := volume.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	labels[string(utils.NodeLabelKey)] = string(utils.NewLabelValue(volume.Status.NodeName))
	labels[string(utils.DrivePathLabelKey)] = string(utils.NewLabelValue(utils.SanitizeDrivePath(drivePath)))
	labels[string(utils.DriveLabelKey)] = string(utils.NewLabelValue(driveName))
	labels[string(utils.CreatedByLabelKey)] = utils.DirectCSIControllerName

	return labels
}

// SyncVolumes syncs direct-csi volume CRD.
func SyncVolumes(ctx context.Context, nodeID string) {
	volumeClient := client.GetDirectCSIClient().DirectCSIVolumes()

	updateLabels := func(volume *directcsi.DirectCSIVolume) func() error {
		return func() error {
			volume, err := volumeClient.Get(
				ctx, volume.Name, metav1.GetOptions{
					TypeMeta: utils.DirectCSIVolumeTypeMeta(),
				},
			)
			if err != nil {
				return err
			}

			// update labels
			volume.SetLabels(getLabels(ctx, volume))
			_, err = volumeClient.Update(
				ctx, volume, metav1.UpdateOptions{
					TypeMeta: utils.DirectCSIVolumeTypeMeta(),
				},
			)
			return err
		}
	}

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh, err := client.ListVolumes(ctx,
		volumeClient,
		[]utils.LabelValue{utils.NewLabelValue(nodeID)},
		nil,
		nil,
		nil,
		client.MaxThreadCount)
	if err != nil {
		klog.V(3).Infof("Error while syncing CRD versions in directcsivolume: %v", err)
		return
	}

	for result := range resultCh {
		if result.Err != nil {
			klog.V(3).Infof("Error while syncing CRD versions in directcsivolume: %v", err)
			return
		}

		if err := retry.RetryOnConflict(retry.DefaultRetry, updateLabels(&result.Volume)); err != nil {
			klog.V(3).Infof("Error while syncing CRD versions in directcsivolume: %v", err)
		}
	}
}
