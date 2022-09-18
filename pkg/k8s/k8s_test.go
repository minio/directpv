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

package k8s

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVolumeStatusTransitions(t *testing.T) {
	statusList := []metav1.Condition{
		{
			Type:               "staged",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Type:               "published",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
		},
	}

	testCases := []struct {
		name       string
		condType   string
		condStatus metav1.ConditionStatus
	}{
		{
			name:       "NodeStageVolumeTransition",
			condType:   "staged",
			condStatus: metav1.ConditionTrue,
		},
		{
			name:       "NodePublishVolumeTransition",
			condType:   "published",
			condStatus: metav1.ConditionTrue,
		},
		{
			name:       "NodeUnpublishVolumeTransition",
			condType:   "published",
			condStatus: metav1.ConditionFalse,
		},
		{
			name:       "NodeUnstageVolumeTransition",
			condType:   "staged",
			condStatus: metav1.ConditionFalse,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			UpdateCondition(statusList, testCase.condType, testCase.condStatus, "", "")
			if !IsCondition(statusList, testCase.condType, testCase.condStatus, "", "") {
				t.Fatalf("case %v: Status transition failed (Type, Status) = (%s, %v) condition list: %v", testCase.name, testCase.condType, testCase.condStatus, statusList)
			}
		})
	}
}
