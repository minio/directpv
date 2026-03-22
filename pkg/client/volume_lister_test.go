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
	"fmt"
	"testing"

	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func TestGetVolumeList(t *testing.T) {
	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset())
	SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
	SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	volumes, err := client.NewVolumeLister().Get(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(volumes) != 0 {
		t.Fatalf("expected: 0, got: %v", len(volumes))
	}

	objects := []runtime.Object{}
	for i := range 2000 {
		objects = append(
			objects, &types.Volume{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("volume-%v", i)}},
		)
	}

	clientset = types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(objects...))
	SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
	SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	volumes, err = client.NewVolumeLister().Get(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(volumes) != 2000 {
		t.Fatalf("expected: 2000, got: %v", len(volumes))
	}
}

func TestGetSortedVolumeList(t *testing.T) {
	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset())
	SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
	SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	volumes, err := client.NewVolumeLister().Get(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(volumes) != 0 {
		t.Fatalf("expected: 0, got: %v", len(volumes))
	}

	objects := []runtime.Object{}
	for i := 1; i <= 4; i++ {
		objects = append(
			objects, &types.Volume{ObjectMeta: metav1.ObjectMeta{Namespace: "CCC", Name: fmt.Sprintf("volume-%v", i)}},
		)
	}
	for i := 5; i <= 8; i++ {
		objects = append(
			objects, &types.Volume{ObjectMeta: metav1.ObjectMeta{Namespace: "BBB", Name: fmt.Sprintf("volume-%v", i)}},
		)
	}
	for i := 9; i <= 12; i++ {
		objects = append(
			objects, &types.Volume{ObjectMeta: metav1.ObjectMeta{Namespace: "AAA", Name: fmt.Sprintf("volume-%v", i)}},
		)
	}

	clientset = types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(objects...))
	SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
	SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	volumes, err = client.NewVolumeLister().Get(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if volumes[0].Namespace != "AAA" {
		t.Fatalf("expected volume to be in Namespace : AAA, got: %v", volumes[0].Namespace)
	}
	if volumes[4].Namespace != "BBB" {
		t.Fatalf("expected volume to be in Namespace : BBB, got: %v", volumes[3].Namespace)
	}
	if volumes[8].Namespace != "CCC" {
		t.Fatalf("expected volume to be in Namespace : CCC, got: %v", volumes[7].Namespace)
	}
}
