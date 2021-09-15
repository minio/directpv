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
	"context"
	"fmt"
	"testing"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	clientsetfake "github.com/minio/direct-csi/pkg/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func TestGetDriveList(t *testing.T) {
	drives, err := GetDriveList(
		context.TODO(),
		clientsetfake.NewSimpleClientset().DirectV1beta3().DirectCSIDrives(),
		nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(drives) != 0 {
		t.Fatalf("expected: 0, got: %v", len(drives))
	}

	objects := []runtime.Object{}
	for i := 0; i < 2000; i++ {
		objects = append(
			objects, &directcsi.DirectCSIDrive{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("drive-%v", i)}},
		)
	}
	drives, err = GetDriveList(
		context.TODO(),
		clientsetfake.NewSimpleClientset(objects...).DirectV1beta3().DirectCSIDrives(),
		nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(drives) != 2000 {
		t.Fatalf("expected: 2000, got: %v", len(drives))
	}
}

func TestGetVolumeList(t *testing.T) {
	volumes, err := GetVolumeList(
		context.TODO(),
		clientsetfake.NewSimpleClientset().DirectV1beta3().DirectCSIVolumes(),
		nil, nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(volumes) != 0 {
		t.Fatalf("expected: 0, got: %v", len(volumes))
	}

	objects := []runtime.Object{}
	for i := 0; i < 2000; i++ {
		objects = append(
			objects, &directcsi.DirectCSIVolume{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("volume-%v", i)}},
		)
	}
	volumes, err = GetVolumeList(
		context.TODO(),
		clientsetfake.NewSimpleClientset(objects...).DirectV1beta3().DirectCSIVolumes(),
		nil, nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(volumes) != 2000 {
		t.Fatalf("expected: 2000, got: %v", len(volumes))
	}
}
