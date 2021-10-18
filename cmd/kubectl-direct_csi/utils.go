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
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	clientset "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/ellipsis"
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

	globRegexp            = regexp.MustCompile(`(^|[^\\])[\*\?\[]`)
	errMixedSelectorUsage = errors.New("either glob or ellipsis pattern is supported")
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

func expandSelectors(selectors []string) (result []string, err error) {
	var values []string
	for _, selector := range selectors {
		values, err = ellipsis.Expand(selector)
		if err != nil {
			return nil, err
		}
		result = append(result, values...)
	}

	return result, nil
}

func splitSelectors(selectors []string) (globs, ellipses []string) {
	for _, selector := range selectors {
		if globRegexp.MatchString(selector) {
			globs = append(globs, selector)
			continue
		}
		ellipses = append(ellipses, selector)
	}
	return globs, ellipses
}

// func hasGlobSelectors(selectors []string) (bool, error) {
// 	globCount := 0
// 	for _, selector := range selectors {
// 		if globRegexp.MatchString(selector) {
// 			globCount++
// 		}
// 	}
// 	if globCount > 0 && globCount != len(selectors) {
// 		return false, errMixedSelectorUsage
// 	}
// 	return globCount > 0, nil
// }
func processFilteredDrives(
	ctx context.Context,
	driveInterface clientset.DirectCSIDriveInterface,
	idArgs []string,
	matchFunc func(*directcsi.DirectCSIDrive) bool,
	applyFunc func(*directcsi.DirectCSIDrive) error,
	processFunc func(context.Context, *directcsi.DirectCSIDrive) error) error {
	var resultCh <-chan utils.ListDriveResult
	var globNodeSelectors, ellipsesNodeSelectors, globDriveSelectors, ellipsesDriveSelectors []string
	if len(idArgs) > 0 {
		resultCh = getDrivesByIds(ctx, idArgs)
	} else {
		globNodeSelectors, ellipsesNodeSelectors = splitSelectors(nodes)
		if len(globNodeSelectors) > 0 && len(ellipsesNodeSelectors) > 0 {
			return errMixedSelectorUsage
		}

		globDriveSelectors, ellipsesDriveSelectors = splitSelectors(drives)
		if len(globDriveSelectors) > 0 && len(ellipsesDriveSelectors) > 0 {
			return errMixedSelectorUsage
		}

		expandedNodeList, err := expandSelectors(ellipsesNodeSelectors)
		if err != nil {
			return err
		}

		expandedDriveList, err := expandSelectors(ellipsesDriveSelectors)
		if err != nil {
			return err
		}

		accessTierSet, err := directcsi.StringsToAccessTiers(accessTiers)
		if err != nil {
			return err
		}
		accessTierSelector := directcsi.AccessTiersToStrings(accessTierSet)

		directCSIClient := utils.GetDirectCSIClient()
		ctx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()

		resultCh, err = utils.ListDrives(ctx,
			directCSIClient.DirectCSIDrives(),
			expandedNodeList,
			expandedDriveList,
			accessTierSelector,
			utils.MaxThreadCount)
		if err != nil {
			return err
		}
	}

	if len(globNodeSelectors) > 0 || len(globDriveSelectors) > 0 {
		if !dryRun {
			klog.Warning("Glob matches will be deprecated soon. Please use ellipses instead")
		}
	}

	return processDrives(
		ctx,
		resultCh,
		func(drive *directcsi.DirectCSIDrive) bool {
			return drive.MatchGlob(
				globNodeSelectors,
				globDriveSelectors,
				status) && matchFunc(drive)
		},
		applyFunc,
		processFunc,
	)
}

func getFilteredDriveList(ctx context.Context, driveInterface clientset.DirectCSIDriveInterface, filterFunc func(directcsi.DirectCSIDrive) bool) ([]directcsi.DirectCSIDrive, error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	globNodeSelectors, ellipsesNodeSelectors := splitSelectors(nodes)
	if len(globNodeSelectors) > 0 && len(ellipsesNodeSelectors) > 0 {
		return nil, errMixedSelectorUsage
	}

	globDriveSelectors, ellipsesDriveSelectors := splitSelectors(drives)
	if len(globDriveSelectors) > 0 && len(ellipsesDriveSelectors) > 0 {
		return nil, errMixedSelectorUsage
	}

	expandedNodeList, err := expandSelectors(ellipsesNodeSelectors)
	if err != nil {
		return nil, err
	}

	expandedDriveList, err := expandSelectors(ellipsesDriveSelectors)
	if err != nil {
		return nil, err
	}

	accessTierSet, err := directcsi.StringsToAccessTiers(accessTiers)
	if err != nil {
		return nil, err
	}
	accessTierSelector := directcsi.AccessTiersToStrings(accessTierSet)

	resultCh, err := utils.ListDrives(ctx,
		driveInterface,
		expandedNodeList,
		expandedDriveList,
		accessTierSelector,
		utils.MaxThreadCount)
	if err != nil {
		return nil, err
	}

	if len(globNodeSelectors) > 0 || len(globDriveSelectors) > 0 {
		if !dryRun {
			klog.Warning("Glob matches will be deprecated soon. Please use ellipses instead")
		}
	}

	filteredDrives := []directcsi.DirectCSIDrive{}
	for result := range resultCh {
		if result.Err != nil {
			return nil, result.Err
		}
		if result.Drive.MatchGlob(
			globNodeSelectors,
			globDriveSelectors,
			status) && filterFunc(result.Drive) {
			filteredDrives = append(filteredDrives, result.Drive)
		}
	}

	return filteredDrives, nil
}

func getFilteredVolumeList(ctx context.Context, volumeInterface clientset.DirectCSIVolumeInterface, filterFunc func(directcsi.DirectCSIVolume) bool) ([]directcsi.DirectCSIVolume, error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	globNodeSelectors, ellipsesNodeSelectors := splitSelectors(nodes)
	if len(globNodeSelectors) > 0 && len(ellipsesNodeSelectors) > 0 {
		return nil, errMixedSelectorUsage
	}

	globDriveSelectors, ellipsesDriveSelectors := splitSelectors(drives)
	if len(globDriveSelectors) > 0 && len(ellipsesDriveSelectors) > 0 {
		return nil, errMixedSelectorUsage
	}

	globPodNameSelectors, ellipsesPodNameSelectors := splitSelectors(podNames)
	if len(globPodNameSelectors) > 0 && len(ellipsesPodNameSelectors) > 0 {
		return nil, errMixedSelectorUsage
	}

	globPodNssSelectors, ellipsesPodNssSelectors := splitSelectors(podNss)
	if len(globPodNssSelectors) > 0 && len(ellipsesPodNssSelectors) > 0 {
		return nil, errMixedSelectorUsage
	}

	expandedNodeList, err := expandSelectors(ellipsesNodeSelectors)
	if err != nil {
		return nil, err
	}

	expandedDriveList, err := expandSelectors(ellipsesDriveSelectors)
	if err != nil {
		return nil, err
	}

	expandedPodNameList, err := expandSelectors(ellipsesPodNameSelectors)
	if err != nil {
		return nil, err
	}

	expandedPodNssList, err := expandSelectors(ellipsesPodNssSelectors)
	if err != nil {
		return nil, err
	}

	resultCh, err := utils.ListVolumes(ctx,
		volumeInterface,
		expandedNodeList,
		expandedDriveList,
		expandedPodNameList,
		expandedPodNssList,
		utils.MaxThreadCount)
	if err != nil {
		return nil, err
	}

	if len(globNodeSelectors) > 0 || len(globDriveSelectors) > 0 || len(globPodNameSelectors) > 0 || len(globPodNssSelectors) > 0 {
		if !dryRun {
			klog.Warning("Glob matches will be deprecated soon. Please use ellipses instead")
		}
	}

	filteredVolumes := []directcsi.DirectCSIVolume{}
	for result := range resultCh {
		if result.Err != nil {
			return nil, result.Err
		}
		if result.Volume.MatchNodeDrives(globNodeSelectors, globDriveSelectors) &&
			result.Volume.MatchPodName(globPodNameSelectors) &&
			result.Volume.MatchPodNamespace(globPodNssSelectors) &&
			result.Volume.MatchStatus(volumeStatus) &&
			filterFunc(result.Volume) {
			filteredVolumes = append(filteredVolumes, result.Volume)
		}
	}

	return filteredVolumes, nil
}

func defaultDriveUpdateFunc(ctx context.Context, drive *directcsi.DirectCSIDrive) error {
	_, err := utils.GetDirectCSIClient().DirectCSIDrives().Update(ctx, drive, metav1.UpdateOptions{})
	return err
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
