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

package client

import (
	"context"
	"fmt"
	"testing"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func TestGetDriveList(t *testing.T) {
	SetLatestDirectCSIDriveInterface(clientsetfake.NewSimpleClientset().DirectV1beta4().DirectCSIDrives())
	drives, err := GetDriveList(context.TODO(), nil, nil, nil)
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
	SetLatestDirectCSIDriveInterface(clientsetfake.NewSimpleClientset(objects...).DirectV1beta4().DirectCSIDrives())
	drives, err = GetDriveList(context.TODO(), nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(drives) != 2000 {
		t.Fatalf("expected: 2000, got: %v", len(drives))
	}
}

func TestGetVolumeList(t *testing.T) {
	SetLatestDirectCSIVolumeInterface(clientsetfake.NewSimpleClientset().DirectV1beta4().DirectCSIVolumes())
	volumes, err := GetVolumeList(context.TODO(), nil, nil, nil, nil)
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

	SetLatestDirectCSIVolumeInterface(clientsetfake.NewSimpleClientset(objects...).DirectV1beta4().DirectCSIVolumes())
	volumes, err = GetVolumeList(context.TODO(), nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(volumes) != 2000 {
		t.Fatalf("expected: 2000, got: %v", len(volumes))
	}
}
