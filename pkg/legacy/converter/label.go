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

package converter

import (
	"fmt"
	"strings"

	directcsi "github.com/minio/directpv/pkg/legacy/apis/direct.csi.min.io/v1beta5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/klog/v2"
)

const (
	// DirectCSIControllerName is the name of the controller
	DirectCSIControllerName = "directcsi-controller"
	// DirectCSIDriverName is the driver name
	DirectCSIDriverName = "directcsi-driver"
)

func normalizeLabelValue(value string) string {
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

// LabelKey stores label keys
type LabelKey string

const (
	// PodNameLabelKey label key for pod name
	PodNameLabelKey LabelKey = directcsi.Group + "/pod.name"
	// PodNSLabelKey label key for pod namespace
	PodNSLabelKey LabelKey = directcsi.Group + "/pod.namespace"
	// NodeLabelKey label key for node
	NodeLabelKey LabelKey = directcsi.Group + "/node"
	// DriveLabelKey label key for drive
	DriveLabelKey LabelKey = directcsi.Group + "/drive"
	// PathLabelKey key for path
	PathLabelKey LabelKey = directcsi.Group + "/path"
	// AccessTierLabelKey label key for access-tier
	AccessTierLabelKey LabelKey = directcsi.Group + "/access-tier"
	// VersionLabelKey label key for version
	VersionLabelKey LabelKey = directcsi.Group + "/version"
	// CreatedByLabelKey label key for created by
	CreatedByLabelKey LabelKey = directcsi.Group + "/created-by"
	// DrivePathLabelKey label key for drive path
	DrivePathLabelKey LabelKey = directcsi.Group + "/drive-path"
	// DirectCSIVersionLabelKey label key for group and version
	DirectCSIVersionLabelKey LabelKey = directcsi.Group + "/" + directcsi.Version
	// TopologyDriverIdentity label key for identity
	TopologyDriverIdentity LabelKey = directcsi.Group + "/identity"
	// TopologyDriverNode label key for node
	TopologyDriverNode LabelKey = directcsi.Group + "/node"
	// TopologyDriverRack label key for rack
	TopologyDriverRack LabelKey = directcsi.Group + "/rack"
	// TopologyDriverZone label key for zone
	TopologyDriverZone LabelKey = directcsi.Group + "/zone"
	// TopologyDriverRegion label key for region
	TopologyDriverRegion LabelKey = directcsi.Group + "/region"
)

// LabelValue is a type definition for label value
type LabelValue string

// NewLabelValue validates and converts string value to label value
func NewLabelValue(value string) LabelValue {
	errs := validation.IsValidLabelValue(value)
	if len(errs) == 0 {
		return LabelValue(value)
	}

	result := LabelValue(normalizeLabelValue(value))
	klog.V(3).InfoS(
		fmt.Sprintf("label value converted due to invalid value; %v", strings.Join(errs, "; ")),
		"value", value, "converted value", result,
	)
	return result
}

// UpdateLabels updates labels in object.
func UpdateLabels(object metav1.Object, labels map[LabelKey]LabelValue) {
	values := object.GetLabels()
	if values == nil {
		values = make(map[string]string)
	}

	for key, value := range labels {
		values[string(key)] = string(value)
	}

	object.SetLabels(values)
}
