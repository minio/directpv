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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AddFinalizer adds finalizer in object meta.
func AddFinalizer(objectMeta *metav1.ObjectMeta, finalizer string) []string {
	finalizers := objectMeta.GetFinalizers()
	for _, f := range finalizers {
		if f == finalizer {
			return finalizers
		}
	}
	finalizers = append(finalizers, finalizer)
	return finalizers
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

// IsCondition checks type/status/reason/message in conditions.
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

// ExcludeFinalizer returns finalizers excluding given finalizer.
func ExcludeFinalizer(finalizers []string, finalizer string) (result []string, found bool) {
	for _, f := range finalizers {
		if f != finalizer {
			result = append(result, f)
		} else {
			found = true
		}
	}
	return
}
