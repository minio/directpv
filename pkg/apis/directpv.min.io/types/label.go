// This file is part of MinIO
// Copyright (c) 2022 MinIO, Inc.
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

package types

import (
	"fmt"
	"strings"

	"github.com/minio/directpv/pkg/consts"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/klog/v2"
)

// LabelKey stores label keys
type LabelKey string

const (
	// NodeLabelKey label key for node
	NodeLabelKey LabelKey = consts.GroupName + "/node"

	// DriveNameLabelKey key for drive name
	DriveNameLabelKey LabelKey = consts.GroupName + "/drive-name"

	// AccessTierLabelKey label key for access-tier
	AccessTierLabelKey LabelKey = consts.GroupName + "/access-tier"

	// DriveLabelKey label key for drive
	DriveLabelKey LabelKey = consts.GroupName + "/drive"

	// VersionLabelKey label key for version
	VersionLabelKey LabelKey = consts.GroupName + "/version"

	// CreatedByLabelKey label key for created by
	CreatedByLabelKey LabelKey = consts.GroupName + "/created-by"

	// PodNameLabelKey label key for pod name
	PodNameLabelKey LabelKey = consts.GroupName + "/pod.name"

	// PodNSLabelKey label key for pod namespace
	PodNSLabelKey LabelKey = consts.GroupName + "/pod.namespace"

	// LatestVersionLabelKey label key for group and version
	LatestVersionLabelKey LabelKey = consts.GroupName + "/" + consts.LatestAPIVersion

	// TopologyDriverIdentity label key for identity
	TopologyDriverIdentity LabelKey = consts.GroupName + "/identity"

	// TopologyDriverNode label key for node
	TopologyDriverNode LabelKey = NodeLabelKey

	// TopologyDriverRack label key for rack
	TopologyDriverRack LabelKey = consts.GroupName + "/rack"

	// TopologyDriverZone label key for zone
	TopologyDriverZone LabelKey = consts.GroupName + "/zone"

	// TopologyDriverRegion label key for region
	TopologyDriverRegion LabelKey = consts.GroupName + "/region"

	// MigratedLabelKey denotes drive/volume migrated from legacy DirectCSI
	MigratedLabelKey LabelKey = consts.GroupName + "/migrated"

	// RequestIDLabelKey label key for request ID
	RequestIDLabelKey LabelKey = consts.GroupName + "/request-id"

	// SuspendLabelKey denotes if the volume is suspended.
	SuspendLabelKey LabelKey = consts.GroupName + "/suspend"

	// VolumeClaimIDLabelKey label key to denote the unique allocation of drives for volumes
	VolumeClaimIDLabelKey LabelKey = consts.GroupName + "/volume-claim-id"

	// VolumeClaimIDLabelKeyPrefix label key prefix for volume claim id to be set on the drive
	VolumeClaimIDLabelKeyPrefix = consts.GroupName + "/volume-claim-id-"

	// ClaimIDLabelKey label key to denote the claim id of the volumes
	ClaimIDLabelKey LabelKey = consts.GroupName + "/claim-id"
)

// LabelValue is a type definition for label value
type LabelValue string

// ToLabelValue validates and converts string value to label value
func ToLabelValue(value string) LabelValue {
	errs := validation.IsValidLabelValue(value)
	if len(errs) == 0 {
		return LabelValue(value)
	}

	normalizeLabelValue := func(value string) string {
		if len(value) > 63 {
			value = value[:63]
		}

		result := []rune(value)
		for i, r := range result {
			switch {
			case (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			default:
				if i != 0 && r != '.' && r != '_' && r != '-' {
					result[i] = '-'
				} else {
					result[i] = 'x'
				}
			}
		}

		return string(result)
	}

	result := LabelValue(normalizeLabelValue(value))
	klog.V(3).InfoS(
		fmt.Sprintf("label value converted due to invalid value; %v", strings.Join(errs, "; ")),
		"value", value, "converted value", result,
	)
	return result
}

// ToLabelSelector converts a map of label key and label value to selector string
func ToLabelSelector(labels map[LabelKey][]LabelValue) string {
	selectors := []string{}
	for key, values := range labels {
		if len(values) != 0 {
			result := []string{}
			for _, value := range values {
				result = append(result, string(value))
			}
			selectors = append(selectors, fmt.Sprintf("%s in (%s)", key, strings.Join(result, ",")))
		}
	}
	return strings.Join(selectors, ",")
}

// NewLabelKey creates a valid label key
func NewLabelKey(labelK string) (labelKey LabelKey, err error) {
	if !strings.HasPrefix(labelK, consts.GroupName) {
		labelK = consts.GroupName + "/" + labelK
	}
	if errs := validation.IsQualifiedName(labelK); len(errs) > 0 {
		err = fmt.Errorf("invalid label key; %s", strings.Join(errs, ", "))
		return
	}
	return LabelKey(labelK), err
}

// NewLabelValue creates a valid label value
func NewLabelValue(v string) (labelValue LabelValue, err error) {
	if errs := validation.IsValidLabelValue(v); len(errs) > 0 {
		err = fmt.Errorf("invalid label value; %s", strings.Join(errs, ", "))
		return
	}
	return LabelValue(v), err
}
