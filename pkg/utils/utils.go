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
	"encoding/json"
	"strings"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/yaml"

	"k8s.io/klog"

	"fmt"
)

func JSONifyAndLog(val interface{}) {
	jsonBytes, err := json.MarshalIndent(val, "", " ")
	if err != nil {
		return
	}
	klog.V(3).Infof(string(jsonBytes))
}

func BoolToCondition(val bool) metav1.ConditionStatus {
	if val {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}

func LogYAML(obj interface{}) error {
	y, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}
	PrintYaml(y)
	return nil
}

func PrintYaml(data []byte) {
	fmt.Print(string(data))
	fmt.Println()
	fmt.Println("---")
	fmt.Println()
}

func ValidateAccessTier(at string) (directcsi.AccessTier, error) {
	switch directcsi.AccessTier(strings.Title(at)) {
	case directcsi.AccessTierWarm:
		return directcsi.AccessTierWarm, nil
	case directcsi.AccessTierHot:
		return directcsi.AccessTierHot, nil
	case directcsi.AccessTierCold:
		return directcsi.AccessTierCold, nil
	case directcsi.AccessTierUnknown:
		return directcsi.AccessTierUnknown, fmt.Errorf("Please set any one among ['hot','warm', 'cold']")
	default:
		return directcsi.AccessTierUnknown, fmt.Errorf("Invalid 'access-tier' value, Please set any one among ['hot','warm','cold']")
	}
}

func GetVersionFromObjectMeta(objectMeta metav1.ObjectMeta) string {
	labels := objectMeta.GetLabels()
	if labels == nil {
		return ""
	}
	val, found := labels[directcsi.Group+"/version"]
	if found {
		return val
	}
	return ""
}

func GetDriveNameForLabel(driveObj *directcsi.DirectCSIDrive) string {
	name := driveObj.Name
	version := GetVersionFromObjectMeta(driveObj.ObjectMeta)
	if version == "v1alpha1" || version == "v1beta1" {
		// The drive name will exceed the threshold limit for older versions
		name = name[0:63]
	}
	return name
}
