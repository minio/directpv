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
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	clientset "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/sys"

	"github.com/minio/direct-csi/pkg/utils"
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

var ( // Default direct csi directory where direct csi audit logs are stored.
	defaultDirectCSIDir = ".direct-csi"

	// Directory contains below files for audit logs
	auditDir = "audit"
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
			return nil, result.Err
		}

		if filterFunc(result.Drive) {
			filteredDrives = append(filteredDrives, result.Drive)
		}
	}

	return filteredDrives, nil
}

func defaultDriveUpdateFunc(directCSIClient clientset.DirectV1beta3Interface) func(context.Context, *directcsi.DirectCSIDrive) error {
	return func(ctx context.Context, drive *directcsi.DirectCSIDrive) error {
		_, err := directCSIClient.DirectCSIDrives().Update(ctx, drive, metav1.UpdateOptions{})
		return err
	}
}

type objectResult struct {
	object runtime.Object
	err    error
}

func processObjects(
	ctx context.Context,
	resultCh <-chan objectResult,
	matchFunc func(runtime.Object) bool,
	applyFunc func(runtime.Object) error,
	processFunc func(context.Context, runtime.Object) error,
) error {
	stopCh := make(chan struct{})
	var stopChMu int32
	closeStopCh := func() {
		if atomic.AddInt32(&stopChMu, 1) == 1 {
			close(stopCh)
		}
	}
	defer closeStopCh()

	objectCh := make(chan runtime.Object)
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
				case object, ok := <-objectCh:
					if !ok {
						return
					}
					if err := processFunc(ctx, object); err != nil {
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
		if result.err != nil {
			err = result.err
			break
		}

		if !matchFunc(result.object) {
			continue
		}

		if err = applyFunc(result.object); err != nil {
			break
		}

		if dryRun {
			if err := utils.LogYAML(result.object); err != nil {
				klog.Errorf("Unable to convert to YAML. %v", err)
			}
			continue
		}

		breakLoop := false
		select {
		case <-ctx.Done():
			breakLoop = true
		case <-stopCh:
			breakLoop = true
		case objectCh <- result.object:
		}

		if breakLoop {
			break
		}
	}

	close(objectCh)
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

func processVolumes(
	ctx context.Context,
	resultCh <-chan utils.ListVolumeResult,
	matchFunc func(*directcsi.DirectCSIVolume) bool,
	applyFunc func(*directcsi.DirectCSIVolume) error,
	processFunc func(context.Context, *directcsi.DirectCSIVolume) error,
) error {
	objectCh := make(chan objectResult)
	go func() {
		defer close(objectCh)
		for result := range resultCh {
			var oresult objectResult
			if result.Err != nil {
				oresult.err = result.Err
			} else {
				volume := result.Volume
				oresult.object = &volume
			}

			select {
			case <-ctx.Done():
				return
			case objectCh <- oresult:
			}
		}
	}()

	return processObjects(
		ctx,
		objectCh,
		func(object runtime.Object) bool {
			return matchFunc(object.(*directcsi.DirectCSIVolume))
		},
		func(object runtime.Object) error {
			return applyFunc(object.(*directcsi.DirectCSIVolume))
		},
		func(ctx context.Context, object runtime.Object) error {
			return processFunc(ctx, object.(*directcsi.DirectCSIVolume))
		},
	)
}

func processDrives(
	ctx context.Context,
	resultCh <-chan utils.ListDriveResult,
	matchFunc func(*directcsi.DirectCSIDrive) bool,
	applyFunc func(*directcsi.DirectCSIDrive) error,
	processFunc func(context.Context, *directcsi.DirectCSIDrive) error,
) error {
	objectCh := make(chan objectResult)
	go func() {
		defer close(objectCh)
		for result := range resultCh {
			var oresult objectResult
			if result.Err != nil {
				oresult.err = result.Err
			} else {
				drive := result.Drive
				oresult.object = &drive
			}

			select {
			case <-ctx.Done():
				return
			case objectCh <- oresult:
			}
		}
	}()

	return processObjects(
		ctx,
		objectCh,
		func(object runtime.Object) bool {
			return matchFunc(object.(*directcsi.DirectCSIDrive))
		},
		func(object runtime.Object) error {
			return applyFunc(object.(*directcsi.DirectCSIDrive))
		},
		func(ctx context.Context, object runtime.Object) error {
			return processFunc(ctx, object.(*directcsi.DirectCSIDrive))
		},
	)
}

func getDirectCSIHomeDir() (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, defaultDirectCSIDir), nil
}

func GetDefaultAuditDir() (string, error) {
	defaultDir, err := getDirectCSIHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(defaultDir, auditDir), nil
}
