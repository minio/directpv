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

package node

import (
	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
	"time"
)

func TestGetLatestStatus(t1 *testing.T) {
	testCases := []struct {
		name           string
		reqStatusList  []metav1.Condition
		selectedStatus metav1.Condition
	}{
		{
			name: "test1",
			reqStatusList: []metav1.Condition{
				{
					Type:               "staged",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Type:               "published",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					Type:               "published",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Date(2000, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					Type:               "staged",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Date(2000, 1, 4, 0, 0, 0, 0, time.UTC),
				},
			},
			selectedStatus: metav1.Condition{
				Type:               "staged",
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Date(2000, 1, 4, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "test2",
			reqStatusList: []metav1.Condition{
				{
					Type:               "staged",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Type:               "published",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					Type:               "published",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Date(2000, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					Type:               "published",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Date(2000, 1, 4, 0, 0, 0, 0, time.UTC),
				},
			},
			selectedStatus: metav1.Condition{
				Type:               "published",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Date(2000, 1, 4, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "test3",
			reqStatusList: []metav1.Condition{
				{
					Type:               "staged",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Type:               "published",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},
			selectedStatus: metav1.Condition{
				Type:               "published",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "test2",
			reqStatusList: []metav1.Condition{
				{
					Type:               "staged",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Type:               "published",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Date(2000, 1, 4, 0, 0, 0, 0, time.UTC),
				},
				{
					Type:               "published",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Date(2000, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					Type:               "published",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},
			selectedStatus: metav1.Condition{
				Type:               "published",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Date(2000, 1, 4, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tt := range testCases {
		t1.Run(tt.name, func(t1 *testing.T) {
			resStatus := GetLatestStatus(tt.reqStatusList)
			if !reflect.DeepEqual(resStatus, tt.selectedStatus) {
				t1.Errorf("Test case name %s: Expected status option = %v, got %v", tt.name, tt.selectedStatus, resStatus)
			}
		})
	}

}

func TestCheckStatusEquality(t2 *testing.T) {
	testCases := []struct {
		name           string
		conditionList1 []metav1.Condition
		conditionList2 []metav1.Condition
		expectedResult bool
	}{
		{
			name: "Test1",
			conditionList1: []metav1.Condition{
				{
					Type:               string(directcsi.DirectCSIDriveConditionOwned),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionMounted),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionFormatted),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionInitialized),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
			},
			conditionList2: []metav1.Condition{
				{
					Type:               string(directcsi.DirectCSIDriveConditionOwned),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionMounted),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionFormatted),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionInitialized),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
			},
			expectedResult: true,
		},
		{
			name: "Test2",
			conditionList1: []metav1.Condition{
				{
					Type:               string(directcsi.DirectCSIDriveConditionOwned),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionMounted),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionFormatted),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionInitialized),
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
				},
			},
			conditionList2: []metav1.Condition{
				{
					Type:               string(directcsi.DirectCSIDriveConditionOwned),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionMounted),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionFormatted),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionInitialized),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
			},
			expectedResult: false,
		},
		{
			name: "Test3",
			conditionList1: []metav1.Condition{
				{
					Type:               string(directcsi.DirectCSIDriveConditionOwned),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionMounted),
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionFormatted),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionInitialized),
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
				},
			},
			conditionList2: []metav1.Condition{
				{
					Type:               string(directcsi.DirectCSIDriveConditionOwned),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionMounted),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionFormatted),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionInitialized),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
			},
			expectedResult: false,
		},
		{
			name: "Test4",
			conditionList1: []metav1.Condition{
				{
					Type:               string(directcsi.DirectCSIDriveConditionOwned),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionMounted),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionFormatted),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionInitialized),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
			},
			conditionList2: []metav1.Condition{
				{
					Type:               string(directcsi.DirectCSIDriveConditionOwned),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionMounted),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionFormatted),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionInitialized),
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
				},
			},
			expectedResult: false,
		},
	}

	for _, tt := range testCases {
		t2.Run(tt.name, func(t2 *testing.T) {
			result := CheckStatusEquality(tt.conditionList1, tt.conditionList2)
			if result != tt.expectedResult {
				t2.Errorf("Test case name %s: Expected result = %v, got %v", tt.name, tt.expectedResult, result)
			}
		})
	}

}
