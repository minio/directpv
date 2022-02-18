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

package utils

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// support gcp, azure, and oidc client auth
	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func DrivesListerWatcher(nodeID string) cache.ListerWatcher {
	labelSelector := ""
	if nodeID != "" {
		labelSelector = fmt.Sprintf("%s=%s", NodeLabelKey, NewLabelValue(nodeID))
	}

	optionsModifier := func(options *metav1.ListOptions) {
		options.LabelSelector = labelSelector
	}

	return cache.NewFilteredListWatchFromClient(
		client.GetLatestDirectCSIRESTClient(),
		"DirectCSIDrives",
		"",
		optionsModifier,
	)
}

func VolumesListerWatcher(nodeID string) cache.ListerWatcher {
	labelSelector := ""
	if nodeID != "" {
		labelSelector = fmt.Sprintf("%s=%s", NodeLabelKey, NewLabelValue(nodeID))
	}

	optionsModifier := func(options *metav1.ListOptions) {
		options.LabelSelector = labelSelector
	}

	return cache.NewFilteredListWatchFromClient(
		client.GetLatestDirectCSIRESTClient(),
		"DirectCSIVolumes",
		"",
		optionsModifier,
	)
}

// IsCondition checks type/status/reason/message in conditions and this function used only for testing.
func IsCondition(statusConditions []metav1.Condition, condType string, condStatus metav1.ConditionStatus, reason, msg string) bool {
	for i := range statusConditions {
		if statusConditions[i].Type == condType &&
			statusConditions[i].Status == condStatus &&
			statusConditions[i].Reason == reason &&
			statusConditions[i].Message == msg {
			return true
		}
	}
	return false
}

// BoolToCondition converts boolean value to condition status.
func BoolToCondition(val bool) metav1.ConditionStatus {
	if val {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}

// RemoveFinalizer removes finalizer in object meta.
func RemoveFinalizer(objectMeta *metav1.ObjectMeta, finalizer string) []string {
	removeByIndex := func(s []string, index int) []string {
		return append(s[:index], s[index+1:]...)
	}
	finalizers := objectMeta.GetFinalizers()
	for index, f := range finalizers {
		if f == finalizer {
			finalizers = removeByIndex(finalizers, index)
			break
		}
	}
	return finalizers
}

// UpdateCondition updates conditions of type/status/reason/message.
func UpdateCondition(statusConditions []metav1.Condition, condType string, condStatus metav1.ConditionStatus, reason, msg string) {
	for i := range statusConditions {
		if statusConditions[i].Type == condType {
			statusConditions[i].Status = condStatus
			statusConditions[i].Reason = reason
			statusConditions[i].Message = msg
			statusConditions[i].LastTransitionTime = metav1.Now()
			break
		}
	}
}

// IsConditionStatus checks type/status in conditions.
func IsConditionStatus(statusConditions []metav1.Condition, condType string, condStatus metav1.ConditionStatus) bool {
	for i := range statusConditions {
		if statusConditions[i].Type == condType &&
			statusConditions[i].Status == condStatus {
			return true
		}
	}
	return false
}

func SetLabels(object metav1.Object, labels map[LabelKey]LabelValue) {
	values := make(map[string]string)
	for key, value := range labels {
		values[string(key)] = string(value)
	}
	object.SetLabels(values)
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

// SanitizeKubeResourceName - Sanitize given name to a valid kubernetes name format.
// RegEx for a kubernetes name is
//
//      ([a-z0-9][-a-z0-9]*)?[a-z0-9]
//
// with a max length of 253
//
// WARNING: This function will truncate to 253 bytes if the input is longer
func SanitizeKubeResourceName(name string) string {
	if len(name) > 253 {
		name = name[:253]
	}

	result := []rune(strings.ToLower(name))
	for i, r := range result {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
		default:
			if i == 0 {
				result[i] = '0'
			} else {
				result[i] = '-'
			}
		}
	}

	return string(result)
}
