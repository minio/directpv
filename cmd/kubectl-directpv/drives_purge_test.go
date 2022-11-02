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

package main

import (
	"context"
	"reflect"
	"testing"

	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/utils"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta5"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func TestPurgeDrives(t1 *testing.T) {
	createTestDrive := func(name, path string, driveStatus directcsi.DriveStatus, conditions []metav1.Condition) *directcsi.DirectCSIDrive {
		csiDrive := &directcsi.DirectCSIDrive{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name:       name,
				Namespace:  metav1.NamespaceNone,
				Finalizers: []string{string(directcsi.DirectCSIDriveFinalizerDataProtection)},
			},
			Status: directcsi.DirectCSIDriveStatus{
				Path:        path,
				DriveStatus: driveStatus,
				Conditions:  conditions,
			},
		}
		return csiDrive
	}

	testDriveObjects := []runtime.Object{
		createTestDrive("d1", "/dev/sda1", directcsi.DriveStatusAvailable, []metav1.Condition{
			{
				Type:    string(directcsi.DirectCSIVolumeConditionReady),
				Status:  metav1.ConditionTrue,
				Message: "",
				Reason:  string(directcsi.DirectCSIVolumeReasonReady),
			},
		}),
		createTestDrive("d2", "/dev/sda2", directcsi.DriveStatusInUse, []metav1.Condition{
			{
				Type:    string(directcsi.DirectCSIVolumeConditionReady),
				Status:  metav1.ConditionTrue,
				Message: "",
				Reason:  string(directcsi.DirectCSIVolumeReasonReady),
			},
		}),
		createTestDrive("d3", "/dev/sda3", directcsi.DriveStatusInUse, []metav1.Condition{
			{
				Type:    string(directcsi.DirectCSIVolumeConditionReady),
				Status:  metav1.ConditionFalse,
				Message: string(directcsi.DirectCSIDriveMessageLost),
				Reason:  string(directcsi.DirectCSIDriveReasonLost),
			},
		}),
		createTestDrive("d4", "/dev/sda4", directcsi.DriveStatusReady, []metav1.Condition{
			{
				Type:    string(directcsi.DirectCSIVolumeConditionReady),
				Status:  metav1.ConditionFalse,
				Message: string(directcsi.DirectCSIDriveMessageLost),
				Reason:  string(directcsi.DirectCSIDriveReasonLost),
			},
		}),
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	testClientSet := clientsetfake.NewSimpleClientset(testDriveObjects...)
	driveInterface := testClientSet.DirectV1beta5().DirectCSIDrives()
	client.SetLatestDirectCSIDriveInterface(driveInterface)

	if err := validateDriveSelectors(); err != nil {
		t1.Fatalf("validateDriveSelectors failed with %v", err)
	}

	if err := purgeDrives(ctx, nil); err != nil {
		t1.Fatalf("purgeDrives failed with %v", err)
	}

	drives, err := client.GetDriveList(ctx, nil, nil, nil)
	if err != nil {
		t1.Fatalf("couldn't get the volume list due to %v", err)
	}
	// only ready and lost drive should be removed
	if len(drives) != 3 {
		t1.Errorf("expected drives count: 3 but got %d", len(drives))
	}

	var intactDriveNames []string
	for _, drive := range drives {
		intactDriveNames = append(intactDriveNames, drive.Name)
	}
	if !reflect.DeepEqual(intactDriveNames, []string{"d1", "d2", "d3"}) {
		t1.Errorf("unexpected result list after purging: %v", intactDriveNames)
	}
}
