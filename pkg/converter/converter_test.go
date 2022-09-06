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

package converter

import (
	"testing"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utiljson "k8s.io/apimachinery/pkg/util/json"
)

func TestMigrate(t *testing.T) {
	testCases := []struct {
		srcObject    runtime.Object
		destObject   runtime.Object
		groupVersion schema.GroupVersion
	}{
		// upgrade/downgrade drive LatestAPIVersion => LatestAPIVersion i.e. no-op
		{
			srcObject: &types.Drive{
				TypeMeta: types.NewDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(consts.DriveFinalizerDataProtection),
						consts.DriveFinalizerPrefix + "volume-1",
						consts.DriveFinalizerPrefix + "volume-2",
					},
				},
				Status: types.DriveStatus{
					NodeName:          "node-name",
					Status:            directpvtypes.DriveStatusOK,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			destObject: &types.Drive{
				TypeMeta: types.NewDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(consts.DriveFinalizerDataProtection),
						consts.DriveFinalizerPrefix + "volume-1",
						consts.DriveFinalizerPrefix + "volume-2",
					},
				},
				Status: types.DriveStatus{
					NodeName:          "node-name",
					Status:            directpvtypes.DriveStatusOK,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   consts.GroupName,
				Version: consts.LatestAPIVersion,
			},
		},
	}

	for i, test := range testCases {
		objBytes, err := utiljson.Marshal(test.srcObject)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		cr := unstructured.Unstructured{}
		if err := cr.UnmarshalJSON(objBytes); err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		result := &unstructured.Unstructured{}

		err = Migrate(&cr, result, test.groupVersion)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		gv := result.GetObjectKind().GroupVersionKind().GroupVersion().String()
		if gv != test.destObject.GetObjectKind().GroupVersionKind().GroupVersion().String() {
			t.Fatalf("case %v: groupVersion: expected: %v, got: %v", i+1, test.groupVersion.Version, gv)
		}
	}
}
