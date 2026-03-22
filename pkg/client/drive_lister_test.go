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
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGetDriveList(t *testing.T) {
	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset())
	SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
	drives, err := client.NewDriveLister().Get(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(drives) != 0 {
		t.Fatalf("expected: 0, got: %v", len(drives))
	}

	objects := []runtime.Object{}
	for i := range 2000 {
		objects = append(
			objects, &types.Drive{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("drive-%v", i)}},
		)
	}

	clientset = types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(objects...))
	SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
	drives, err = client.NewDriveLister().Get(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(drives) != 2000 {
		t.Fatalf("expected: 2000, got: %v", len(drives))
	}
}
