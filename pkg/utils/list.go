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

package utils

import (
	"context"
	"fmt"
	"strings"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	clientset "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1beta3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/klog/v2"
)

type LabelValue string

func AccessTiersToLabelValues(accessTiers []directcsi.AccessTier) (labelValues []LabelValue) {
	for _, accessTier := range accessTiers {
		labelValues = append(labelValues, LabelValue(accessTier))
	}
	return
}

func labelValuesToStrings(values []LabelValue) (strValues []string) {
	for _, value := range values {
		strValues = append(strValues, string(value))
	}
	return
}

func NewLabelValue(value string) (LabelValue, error) {
	value = SanitizeLabelV(value)
	if errList := validation.IsValidLabelValue(value); len(errList) > 0 {
		return LabelValue(""), fmt.Errorf("invalid label value %v: %v", value, strings.Join(errList, " ,"))
	}
	return LabelValue(value), nil
}

func ToLabelValues(vs []string) (labelValues []LabelValue, err error) {
	for _, v := range vs {
		labelV, err := NewLabelValue(v)
		if err != nil {
			return nil, err
		}
		labelValues = append(labelValues, labelV)
	}
	return
}

func toLabelSelector(labelMap map[string][]string) string {
	selectors := []string{}
	for label, values := range labelMap {
		if len(values) > 0 {
			selectors = append(selectors, fmt.Sprintf("%s in (%s)", label, strings.Join(values, ",")))
		}
	}
	return strings.Join(selectors, ",")
}

// ListDriveResult denotes list of drive result.
type ListDriveResult struct {
	Drive directcsi.DirectCSIDrive
	Err   error
}

// ListDrives lists direct-csi drives.
func ListDrives(ctx context.Context, nodes, drives, accessTiers []LabelValue, maxObjects int64) (<-chan ListDriveResult, error) {
	labelMap := map[string][]string{
		DrivePathLabel:  labelValuesToStrings(drives),
		NodeLabel:       labelValuesToStrings(nodes),
		AccessTierLabel: labelValuesToStrings(accessTiers),
	}
	labelSelector := toLabelSelector(labelMap)

	resultCh := make(chan ListDriveResult)
	go func() {
		defer close(resultCh)
		klog.V(5).InfoS("Listing DirectCSIDrives", "limit", maxObjects, "selectors", labelSelector)

		send := func(result ListDriveResult) bool {
			select {
			case <-ctx.Done():
				return false
			case resultCh <- result:
				return true
			}
		}

		options := metav1.ListOptions{
			Limit:         maxObjects,
			LabelSelector: labelSelector,
		}
		for {
			result, err := GetDirectCSIClient().DirectCSIDrives().List(ctx, options)
			if err != nil {
				send(ListDriveResult{Err: err})
				return
			}

			for _, item := range result.Items {
				if !send(ListDriveResult{Drive: item}) {
					return
				}
			}

			if result.Continue == "" {
				return
			}

			options.Continue = result.Continue
		}
	}()

	return resultCh, nil
}

// GetDriveList gets list of drives.
func GetDriveList(ctx context.Context, nodes, drives, accessTiers []LabelValue) ([]directcsi.DirectCSIDrive, error) {
	resultCh, err := ListDrives(ctx, nodes, drives, accessTiers, MaxThreadCount)
	if err != nil {
		return nil, err
	}

	driveList := []directcsi.DirectCSIDrive{}
	for result := range resultCh {
		if result.Err != nil {
			return driveList, result.Err
		}
		driveList = append(driveList, result.Drive)
	}

	return driveList, nil
}

// ListVolumeResult denotes list of volume result.
type ListVolumeResult struct {
	Volume directcsi.DirectCSIVolume
	Err    error
}

// ListVolumes lists direct-csi volumes.
func ListVolumes(ctx context.Context, volumeInterface clientset.DirectCSIVolumeInterface, nodes, drives, podNames, podNss []LabelValue, maxObjects int64) (<-chan ListVolumeResult, error) {
	labelMap := map[string][]string{
		ReservedDrivePathLabel: labelValuesToStrings(drives),
		NodeLabel:              labelValuesToStrings(nodes),
		PodNameLabel:           labelValuesToStrings(podNames),
		PodNamespaceLabel:      labelValuesToStrings(podNss),
	}
	labelSelector := toLabelSelector(labelMap)

	resultCh := make(chan ListVolumeResult)
	go func() {
		defer close(resultCh)
		klog.V(5).InfoS("Listing DirectCSIVolumes", "limit", maxObjects, "selectors", labelSelector)

		send := func(result ListVolumeResult) bool {
			select {
			case <-ctx.Done():
				return false
			case resultCh <- result:
				return true
			}
		}

		options := metav1.ListOptions{
			Limit:         maxObjects,
			LabelSelector: labelSelector,
		}

		for {
			result, err := volumeInterface.List(ctx, options)
			if err != nil {
				send(ListVolumeResult{Err: err})
				return
			}

			for _, item := range result.Items {
				if !send(ListVolumeResult{Volume: item}) {
					return
				}
			}

			if result.Continue == "" {
				return
			}

			options.Continue = result.Continue
		}
	}()

	return resultCh, nil
}

// GetVolumeList gets list of volumes.
func GetVolumeList(ctx context.Context, volumeInterface clientset.DirectCSIVolumeInterface, nodes, drives, podNames, podNss []LabelValue) ([]directcsi.DirectCSIVolume, error) {
	resultCh, err := ListVolumes(ctx, volumeInterface, nodes, drives, podNames, podNss, MaxThreadCount)
	if err != nil {
		return nil, err
	}

	volumeList := []directcsi.DirectCSIVolume{}
	for result := range resultCh {
		if result.Err != nil {
			return volumeList, result.Err
		}
		volumeList = append(volumeList, result.Volume)
	}

	return volumeList, nil
}
