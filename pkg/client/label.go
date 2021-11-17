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

package client

import (
	"fmt"
	"strings"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/klog/v2"
)

const (
	DirectCSIControllerName = "directcsi-controller"
	DirectCSIDriverName     = "directcsi-driver"
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

type LabelKey string

const (
	PodNameLabelKey          LabelKey = directcsi.Group + "/pod.name"
	PodNSLabelKey            LabelKey = directcsi.Group + "/pod.namespace"
	NodeLabelKey             LabelKey = directcsi.Group + "/node"
	DriveLabelKey            LabelKey = directcsi.Group + "/drive"
	PathLabelKey             LabelKey = directcsi.Group + "/path"
	AccessTierLabelKey       LabelKey = directcsi.Group + "/access-tier"
	VersionLabelKey          LabelKey = directcsi.Group + "/version"
	CreatedByLabelKey        LabelKey = directcsi.Group + "/created-by"
	DrivePathLabelKey        LabelKey = directcsi.Group + "/drive-path"
	DirectCSIVersionLabelKey LabelKey = directcsi.Group + "/" + directcsi.Version

	TopologyDriverIdentity LabelKey = directcsi.Group + "/identity"
	TopologyDriverNode     LabelKey = directcsi.Group + "/node"
	TopologyDriverRack     LabelKey = directcsi.Group + "/rack"
	TopologyDriverZone     LabelKey = directcsi.Group + "/zone"
	TopologyDriverRegion   LabelKey = directcsi.Group + "/region"
)

type LabelValue string

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

func SetLabels(object metav1.Object, labels map[LabelKey]LabelValue) {
	values := make(map[string]string)
	for key, value := range labels {
		values[string(key)] = string(value)
	}
	object.SetLabels(values)
}
