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
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"

	"github.com/minio/direct-csi/pkg/client"
	clientset "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/ellipsis"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	dot                     = "•"
	directCSIPartitionInfix = "-part-"
)

var (
	bold  = color.New(color.Bold).SprintFunc()
	red   = color.New(color.FgRed).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()

	globRegexp                  = regexp.MustCompile(`(^|[^\\])[\*\?\[]`)
	errMixedSelectorUsage       = errors.New("either glob or ellipsis pattern is supported")
	errMixedStatusSelectorUsage = fmt.Errorf("either glob or [%s] is supported", strings.Join(directcsi.SupportedStatusSelectorValues(), ", "))
)

type Command string

const (
	SetAcessTier   Command = "setAccessTier"
	UnSetAcessTier Command = "unSetAccessTier"
	Format         Command = "format"
	DriveRelease   Command = "driveRelease"
)

func printableString(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func printableBytes(value int64) string {
	if value == 0 {
		return "-"
	}

	return humanize.IBytes(uint64(value))
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

func processFilteredDrives(
	ctx context.Context,
	driveInterface clientset.DirectCSIDriveInterface,
	idArgs []string,
	matchFunc func(*directcsi.DirectCSIDrive) bool,
	applyFunc func(*directcsi.DirectCSIDrive) error,
	processFunc func(context.Context, *directcsi.DirectCSIDrive) error, command Command) error {
	var resultCh <-chan client.ListDriveResult
	var err error
	if len(idArgs) > 0 {
		resultCh = getDrivesByIds(ctx, idArgs)
	} else {
		ctx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()

		resultCh, err = client.ListDrives(ctx,
			driveInterface,
			nodeSelectorValues,
			driveSelectorValues,
			accessTierSelectorValues,
			client.MaxThreadCount)
		if err != nil {
			return err
		}
	}

	defaultAuditDir, err := utils.GetDefaultAuditDir()
	if err != nil {
		return fmt.Errorf("unable to get default audit directory; %w", err)
	}
	if err := os.MkdirAll(defaultAuditDir, 0700); err != nil {
		return err
	}

	file, err := utils.NewSafeFile(fmt.Sprintf("%v/%v-%v", defaultAuditDir, string(command), time.Now().UnixNano()))
	if err != nil {
		return fmt.Errorf("unable to get default audit directory ; %w", err)
	}

	defer func() {
		if cerr := file.Close(); err != nil {
			klog.Errorf("unable to close file; %w", cerr)
		} else {
			err = cerr
		}
	}()

	return client.ProcessDrives(
		ctx,
		resultCh,
		func(drive *directcsi.DirectCSIDrive) bool {
			return drive.MatchGlob(
				nodeGlobs,
				driveGlobs,
				statusGlobs) && matchFunc(drive)
		},
		applyFunc,
		processFunc,
		file,
		dryRun,
	)
}

func getFilteredDriveList(ctx context.Context, driveInterface clientset.DirectCSIDriveInterface, filterFunc func(directcsi.DirectCSIDrive) bool) ([]directcsi.DirectCSIDrive, error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh, err := client.ListDrives(ctx,
		driveInterface,
		nodeSelectorValues,
		driveSelectorValues,
		accessTierSelectorValues,
		client.MaxThreadCount)
	if err != nil {
		return nil, err
	}

	filteredDrives := []directcsi.DirectCSIDrive{}
	for result := range resultCh {
		if result.Err != nil {
			return nil, result.Err
		}
		if result.Drive.MatchGlob(
			nodeGlobs,
			driveGlobs,
			statusGlobs) && filterFunc(result.Drive) {
			filteredDrives = append(filteredDrives, result.Drive)
		}
	}

	return filteredDrives, nil
}

func getFilteredVolumeList(ctx context.Context, volumeInterface clientset.DirectCSIVolumeInterface, filterFunc func(directcsi.DirectCSIVolume) bool) ([]directcsi.DirectCSIVolume, error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh, err := client.ListVolumes(ctx,
		volumeInterface,
		nodeSelectorValues,
		driveSelectorValues,
		podNameSelectorValues,
		podNsSelectorValues,
		client.MaxThreadCount)
	if err != nil {
		return nil, err
	}

	filteredVolumes := []directcsi.DirectCSIVolume{}
	for result := range resultCh {
		if result.Err != nil {
			return nil, result.Err
		}
		if result.Volume.MatchNodeDrives(nodeGlobs, driveGlobs) &&
			result.Volume.MatchPodName(podNameGlobs) &&
			result.Volume.MatchPodNamespace(podNsGlobs) &&
			result.Volume.MatchStatus(volumeStatusList) &&
			filterFunc(result.Volume) {
			filteredVolumes = append(filteredVolumes, result.Volume)
		}
	}

	return filteredVolumes, nil
}

func defaultDriveUpdateFunc(directCSIClient clientset.DirectV1beta3Interface) func(context.Context, *directcsi.DirectCSIDrive) error {
	return func(ctx context.Context, drive *directcsi.DirectCSIDrive) error {
		_, err := directCSIClient.DirectCSIDrives().Update(ctx, drive, metav1.UpdateOptions{})
		return err
	}
}

func getValidSelectors(selectors []string) (globs []string, values []utils.LabelValue, err error) {
	for _, selector := range selectors {
		if globRegexp.MatchString(selector) {
			globs = append(globs, selector)
		} else {
			result, err := ellipsis.Expand(selector)
			if err != nil {
				return nil, nil, err
			}

			for _, value := range result {
				values = append(values, utils.NewLabelValue(value))
			}
		}
	}

	if len(globs) > 0 && len(values) > 0 {
		return nil, nil, errMixedSelectorUsage
	}

	return
}

func getValidDriveSelectors(drives []string) ([]string, []utils.LabelValue, error) {
	for i := range drives {
		drives[i] = utils.SanitizeDrivePath(drives[i])
	}
	return getValidSelectors(drives)
}

func getValidNodeSelectors(nodes []string) ([]string, []utils.LabelValue, error) {
	return getValidSelectors(nodes)
}

func getValidAccessTierSelectors(accessTiers []string) ([]utils.LabelValue, error) {
	accessTierSet, err := directcsi.StringsToAccessTiers(accessTiers)
	if err != nil {
		return nil, err
	}

	var labelValues []utils.LabelValue
	for _, accessTier := range accessTierSet {
		labelValues = append(labelValues, utils.NewLabelValue(string(accessTier)))
	}

	return labelValues, nil
}

func getValidDriveStatusSelectors(selectors []string) (globs []string, statusList []directcsi.DriveStatus, err error) {
	for _, selector := range selectors {
		if globRegexp.MatchString(selector) {
			globs = append(globs, selector)
		} else {
			driveStatus, err := directcsi.ToDriveStatus(selector)
			if err != nil {
				return nil, nil, err
			}
			statusList = append(statusList, driveStatus)
		}
	}

	if len(globs) > 0 && len(statusList) > 0 {
		return nil, nil, errMixedStatusSelectorUsage
	}

	return
}

func getValidPodNameSelectors(podNames []string) ([]string, []utils.LabelValue, error) {
	return getValidSelectors(podNames)
}

func getValidPodNameSpaceSelectors(podNamespaces []string) ([]string, []utils.LabelValue, error) {
	return getValidSelectors(podNamespaces)
}

func getValidVolumeStatusSelectors(statusList []string) ([]string, error) {
	for _, status := range statusList {
		switch directcsi.DirectCSIVolumeCondition(strings.Title(status)) {
		case directcsi.DirectCSIVolumeConditionPublished, directcsi.DirectCSIVolumeConditionStaged, directcsi.DirectCSIVolumeConditionReady:
		default:
			return nil, fmt.Errorf("invalid status '%s'. supported values are ['published', 'staged']", status)
		}
	}
	return statusList, nil
}

func setDriveAccessTier(drive *directcsi.DirectCSIDrive, accessTier directcsi.AccessTier) {
	drive.Status.AccessTier = accessTier
	labels := drive.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[string(utils.AccessTierLabelKey)] = string(utils.NewLabelValue(string(accessTier)))
	drive.SetLabels(labels)
}

func getLabelValue(obj metav1.Object, key string) string {
	if labels := obj.GetLabels(); labels != nil {
		return labels[key]
	}
	return ""
}

func getDirectCSIPath(driveName string) string {
	if strings.Contains(driveName, sys.DirectCSIDevRoot) {
		return driveName
	}
	if strings.HasPrefix(driveName, sys.HostDevRoot) {
		return getDirectCSIPath(filepath.Base(driveName))
	}
	return filepath.Join(sys.DirectCSIDevRoot, driveName)
}
