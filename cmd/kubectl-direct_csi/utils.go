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

package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/fatih/color"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	clientset "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1beta2"
	"github.com/minio/direct-csi/pkg/sys"
	"k8s.io/client-go/util/retry"

	"github.com/minio/direct-csi/pkg/converter"
	"github.com/minio/direct-csi/pkg/utils"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	runtime "k8s.io/apimachinery/pkg/runtime"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/klog/v2"
)

const (
	dot                     = "•"
	directCSIPartitionInfix = "-part-"
)

type migrateFunc func(ctx context.Context, fromVersion string) error

var (
	bold   = color.New(color.Bold).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
)

// ListVolumesInDrive returns a slice of all the DirectCSIVolumes created on a given DirectCSIDrive
func ListVolumesInDrive(drive directcsi.DirectCSIDrive, volumes *directcsi.DirectCSIVolumeList, vols []directcsi.DirectCSIVolume) []directcsi.DirectCSIVolume {
	for _, volume := range volumes.Items {
		if volume.Status.Drive == drive.ObjectMeta.Name {
			vols = append(vols, volume)
		}
	}
	return vols
}

func getAccessTierSet(accessTiers []string) ([]directcsi.AccessTier, error) {
	var atSet []directcsi.AccessTier
	for i := range accessTiers {
		if accessTiers[i] == "*" {
			return []directcsi.AccessTier{
				directcsi.AccessTierHot,
				directcsi.AccessTierWarm,
				directcsi.AccessTierCold,
			}, nil
		}
		at, err := utils.ValidateAccessTier(strings.TrimSpace(accessTiers[i]))
		if err != nil {
			return atSet, err
		}
		atSet = append(atSet, at)
	}
	return atSet, nil
}

func printableString(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func printYAML(obj interface{}) error {
	y, err := utils.ToYAML(obj)
	if err != nil {
		return err
	}
	fmt.Println(y)
	return nil
}

func printJSON(obj interface{}) error {
	j, err := utils.ToJSON(obj)
	if err != nil {
		return err
	}
	fmt.Println(j)
	return nil
}

func canonicalNameFromPath(val string) string {
	dr := strings.ReplaceAll(val, sys.DirectCSIDevRoot+"/", "")
	dr = strings.ReplaceAll(dr, sys.HostDevRoot+"/", "")
	return strings.ReplaceAll(dr, directCSIPartitionInfix, "")
}

func syncCRDObjects(ctx context.Context) error {
	crdClient := utils.GetCRDClient()

	supportedCRDs := []string{
		driveCRDName,
		volumeCRDName,
	}
	for _, crdName := range supportedCRDs {
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			crd, err := crdClient.Get(ctx, crdName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			v := utils.GetLabelV(crd, utils.VersionLabel)
			if v == string(directcsi.Version) {
				// already upgraded to latest
				return nil
			}

			if err := syncObjects(ctx, crd); err != nil {
				return err
			}

			utils.SetLabelKV(crd, utils.VersionLabel, directcsi.Version)
			if _, err := crdClient.Update(ctx, crd, metav1.UpdateOptions{}); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func syncObjects(ctx context.Context, crd *apiextensions.CustomResourceDefinition) error {
	storedVersions := crd.Status.StoredVersions
	if len(storedVersions) == 0 {
		// No objects stored
		return nil
	}

	if len(storedVersions) == 1 && storedVersions[0] == string(directcsi.Version) {
		// already latest
		return nil
	}

	info, err := utils.GetGroupKindVersions(directcsi.Group, crd.Spec.Names.Kind, "v1beta1", "v1alpha1")
	if err != nil {
		return err
	}
	fromVersion := info.Version

	migrateFn := func() migrateFunc {
		switch crd.Name {
		case driveCRDName:
			return migrateDriveObjects
		case volumeCRDName:
			return migrateVolumeObjects
		default:
			fn := func(_ context.Context, _ string) error {
				return fmt.Errorf("Unsupported crd: %v", crd.Name)
			}
			return fn
		}
	}()
	if err := migrateFn(ctx, fromVersion); err != nil {
		return err
	}

	klog.Infof("'%s' objects successfully synced", utils.Bold(crd.Name))
	return nil
}

func toUnstructured(obj interface{}) (unstructured.Unstructured, error) {
	unstructured := unstructured.Unstructured{}
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return unstructured, err
	}
	unstructured.Object = unstructuredObj
	return unstructured, nil
}

func migrateDriveObjects(ctx context.Context, fromVersion string) error {
	directCSIClient := utils.GetDirectCSIClient()
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh, err := utils.ListDrives(ctx, directCSIClient.DirectCSIDrives(), nil, nil, nil, utils.MaxThreadCount)
	if err != nil {
		return err
	}

	return processDrives(
		ctx,
		resultCh,
		func(drive *directcsi.DirectCSIDrive) bool {
			return true
		},
		func(drive *directcsi.DirectCSIDrive) error {
			return nil
		},
		func(ctx context.Context, drive *directcsi.DirectCSIDrive) error {
			unstructured, err := toUnstructured(drive)
			if err != nil {
				klog.V(4).Infof("Error while syncing directcsidrive [%v]: %v", unstructured.GetName(), err)
				return nil
			}
			unstructured.SetAPIVersion(strings.Join([]string{directcsi.Group, fromVersion}, "/"))
			if err := converter.Migrate(&unstructured, strings.Join([]string{directcsi.Group, directcsi.Version}, "/")); err != nil {
				klog.V(4).Infof("Error while syncing directcsidrive [%v]: %v", unstructured.GetName(), err)
				return nil
			}
			var updatedDrive directcsi.DirectCSIDrive
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.Object, &updatedDrive); err != nil {
				klog.V(4).Infof("Error while syncing directcsidrive [%v]: %v", unstructured.GetName(), err)
				return nil
			}

			_, err = directCSIClient.DirectCSIDrives().Update(ctx, &updatedDrive, metav1.UpdateOptions{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
			})
			if err != nil {
				klog.V(4).Infof("Error while syncing directcsidrive [%v]: %v", unstructured.GetName(), err)
			}
			return nil
		},
	)
}

func migrateVolumeObjects(ctx context.Context, fromVersion string) error {
	volumeClient := utils.GetDirectCSIClient().DirectCSIVolumes()
	resultCh, err := utils.ListVolumes(ctx, volumeClient, nil, nil, nil, nil, utils.MaxThreadCount)
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}

	for result := range resultCh {
		if result.Err != nil {
			return result.Err
		}

		v := result.Volume

		threadiness <- struct{}{}
		wg.Add(1)
		go func(v directcsi.DirectCSIVolume) {
			defer func() {
				wg.Done()
				<-threadiness
			}()

			unstructured, err := toUnstructured(&v)
			if err != nil {
				klog.V(4).Infof("Error while syncing directcsivolume [%v]: %v", unstructured.GetName(), err)
				return
			}
			unstructured.SetAPIVersion(strings.Join([]string{directcsi.Group, fromVersion}, "/"))
			if err := converter.Migrate(&unstructured, strings.Join([]string{directcsi.Group, directcsi.Version}, "/")); err != nil {
				klog.V(4).Infof("Error while syncing directcsivolume [%v]: %v", unstructured.GetName(), err)
				return
			}
			var directCSIVolume directcsi.DirectCSIVolume
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.Object, &directCSIVolume); err != nil {
				klog.V(4).Infof("Error while syncing directcsivolume [%v]: %v", unstructured.GetName(), err)
				return
			}
			updateOpts := metav1.UpdateOptions{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
			}
			if _, err := volumeClient.Update(ctx, &directCSIVolume, updateOpts); err != nil {
				klog.V(4).Infof("Error while syncing directcsivolume [%v]: %v", unstructured.GetName(), err)
				return
			}
		}(v)
	}

	wg.Wait()

	return nil
}

func getFilteredDriveList(ctx context.Context, driveInterface clientset.DirectCSIDriveInterface, filterFunc func(directcsi.DirectCSIDrive) bool) ([]directcsi.DirectCSIDrive, error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh, err := utils.ListDrives(ctx, driveInterface, nil, nil, nil, utils.MaxThreadCount)
	if err != nil {
		return nil, err
	}

	filteredDrives := []directcsi.DirectCSIDrive{}
	for result := range resultCh {
		if result.Err != nil {
			return nil, err
		}

		if filterFunc(result.Drive) {
			filteredDrives = append(filteredDrives, result.Drive)
		}
	}

	return filteredDrives, nil
}

func defaultDriveUpdateFunc(directCSIClient clientset.DirectV1beta2Interface) func(context.Context, *directcsi.DirectCSIDrive) error {
	return func(ctx context.Context, drive *directcsi.DirectCSIDrive) error {
		_, err := directCSIClient.DirectCSIDrives().Update(ctx, drive, metav1.UpdateOptions{})
		return err
	}
}

func processDrives(
	ctx context.Context,
	resultCh <-chan utils.ListDriveResult,
	matchFunc func(*directcsi.DirectCSIDrive) bool,
	applyFunc func(*directcsi.DirectCSIDrive) error,
	processFunc func(context.Context, *directcsi.DirectCSIDrive) error,
) error {
	stopCh := make(chan struct{})
	var stopChMu int32
	closeStopCh := func() {
		if atomic.AddInt32(&stopChMu, 1) == 1 {
			close(stopCh)
		}
	}
	defer closeStopCh()

	driveCh := make(chan *directcsi.DirectCSIDrive)
	var wg sync.WaitGroup

	// Start utils.MaxThreadCount workers.
	var errs []error
	for i := 0; i < utils.MaxThreadCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-stopCh:
					return
				case drive, ok := <-driveCh:
					if !ok {
						return
					}
					if err := processFunc(ctx, drive); err != nil {
						errs = append(errs, err)
						defer closeStopCh()
						return
					}
				}
			}
		}()
	}

	var err error
	for result := range resultCh {
		if result.Err != nil {
			err = result.Err
			break
		}

		drive := result.Drive

		if !matchFunc(&drive) {
			continue
		}

		if err = applyFunc(&drive); err != nil {
			break
		}

		if dryRun {
			if err := utils.LogYAML(drive); err != nil {
				klog.Errorf("Unable to convert DirectCSIDrive to YAML. %v", err)
			}
			continue
		}

		breakLoop := false
		select {
		case <-ctx.Done():
			breakLoop = true
		case <-stopCh:
			breakLoop = true
		case driveCh <- &drive:
		}

		if breakLoop {
			break
		}
	}

	close(driveCh)
	wg.Wait()

	if err != nil {
		return err
	}

	msgs := []string{}
	for _, err := range errs {
		msgs = append(msgs, err.Error())
	}
	if msg := strings.Join(msgs, "; "); msg != "" {
		return errors.New(msg)
	}

	return nil
}
