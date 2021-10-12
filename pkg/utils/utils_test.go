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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/mitchellh/go-homedir"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWriteObject(t *testing.T) {
	var byteBuffer bytes.Buffer
	meta := metav1.ObjectMeta{
		Name:      SanitizeKubeResourceName("direct.csi.min.io"),
		Namespace: SanitizeKubeResourceName("direct.csi.min.io"),
		Annotations: map[string]string{
			CreatedByLabel: "kubectl/direct-csi",
		},
		Labels: map[string]string{
			"app":  "direct.csi.min.io",
			"type": "CSIDriver",
		},
	}
	testCases := []struct {
		input       interface{}
		errReturned bool
	}{
		{
			input: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: meta,
				Spec: corev1.NamespaceSpec{
					Finalizers: []corev1.FinalizerName{},
				},
				Status: corev1.NamespaceStatus{},
			},
			errReturned: false,
		},
		{[]string{"1"}, false},
		{make(chan int), true},
	}
	for i, test := range testCases {
		err := WriteObject(&byteBuffer, testCases[i].input)
		errReturned := err != nil
		if errReturned != test.errReturned {
			t.Fatalf("Test %d: expected %t got %t", i+1, test.errReturned, errReturned)
		}
	}
}

func TestNewSafeFile(t *testing.T) {
	tempFile, _ := os.CreateTemp("", "safefile.")
	dirname, _ := os.UserHomeDir()
	timeinNs := time.Now().UnixNano()
	homeDir, _ := homedir.Dir()
	filepath.Join(homeDir, ".direct-csi")
	defaultDirname := filepath.Join(homeDir, ".direct-csi")
	testCases := []struct {
		input       string
		output      *SafeFile
		errReturned bool
	}{
		{
			input: fmt.Sprintf("%v/%v-%v", dirname+"/.direct-csi", "audit/"+"install", timeinNs),
			output: &SafeFile{
				filename: fmt.Sprintf("%v/%v-%v", defaultDirname, "audit/"+"install", timeinNs),
				tempFile: tempFile,
			},
			errReturned: false,
		},
	}
	for i, test := range testCases {
		out, err := NewSafeFile(testCases[i].input)
		errReturned := err != nil
		if errReturned != test.errReturned {
			t.Fatalf("Test %d: expected %t got %t", i+1, test.errReturned, errReturned)
		}
		if !reflect.DeepEqual(out.filename, test.input) {
			t.Fatalf("Test %d: expected %v got %v", i+1, test.input, out.filename)
		}
	}

}

func TestVolumeStatusTransitions(t1 *testing.T) {

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

	for _, tt := range testCases {
		t1.Run(tt.name, func(t1 *testing.T) {
			UpdateCondition(statusList, tt.condType, tt.condStatus, "", "")
			if !IsCondition(statusList, tt.condType, tt.condStatus, "", "") {
				t1.Errorf("Test case name %s: Status transition failed (Type, Status) = (%s, %v) condition list: %v", tt.name, tt.condType, tt.condStatus, statusList)
			}
		})
	}

}

func TestValidateDriveStatus(t1 *testing.T) {
	testCases := []struct {
		statusStr            string
		directcsiDriveStatus directcsi.DriveStatus
		expectErr            bool
	}{
		{
			statusStr:            "inuse",
			directcsiDriveStatus: directcsi.DriveStatusInUse,
		},
		{
			statusStr:            "available",
			directcsiDriveStatus: directcsi.DriveStatusAvailable,
		},
		{
			statusStr:            "unavailable",
			directcsiDriveStatus: directcsi.DriveStatusUnavailable,
		},
		{
			statusStr:            "ready",
			directcsiDriveStatus: directcsi.DriveStatusReady,
		},
		{
			statusStr:            "terminating",
			directcsiDriveStatus: directcsi.DriveStatusTerminating,
		},
		{
			statusStr:            "released",
			directcsiDriveStatus: directcsi.DriveStatusReleased,
		},
		{
			statusStr:            "abc",
			directcsiDriveStatus: directcsi.DriveStatus("unknown"),
			expectErr:            true,
		},
	}

	for i, testCase := range testCases {
		driveStatus, err := ValidateDriveStatus(testCase.statusStr)
		if err != nil && !testCase.expectErr {
			t1.Fatalf("case %v: did not expect error but got: %v", i+1, err)
		}
		if testCase.directcsiDriveStatus != driveStatus {
			t1.Errorf("case %v: Expected drive status = %v, got %v", i+1, testCase.directcsiDriveStatus, driveStatus)
		}
	}
}
