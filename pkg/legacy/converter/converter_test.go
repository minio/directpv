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

	directv1beta1 "github.com/minio/directpv/pkg/legacy/apis/direct.csi.min.io/v1beta1"
	directv1beta2 "github.com/minio/directpv/pkg/legacy/apis/direct.csi.min.io/v1beta2"
	directv1beta3 "github.com/minio/directpv/pkg/legacy/apis/direct.csi.min.io/v1beta3"
	directv1beta4 "github.com/minio/directpv/pkg/legacy/apis/direct.csi.min.io/v1beta4"
	directv1beta5 "github.com/minio/directpv/pkg/legacy/apis/direct.csi.min.io/v1beta5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	json1 "k8s.io/apimachinery/pkg/util/json"
)

func TestMigrate(t *testing.T) {
	testCases := []struct {
		srcObject    runtime.Object
		destObject   runtime.Object
		groupVersion schema.GroupVersion
	}{
		// upgrade drive v1beta1 => v1beta5
		{
			srcObject: &directv1beta1.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta1", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta1.DirectCSIDriveFinalizerDataProtection),
						directv1beta1.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta1.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta1.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta1.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			destObject: &directv1beta5.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta5", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta5.DirectCSIDriveFinalizerDataProtection),
						directv1beta5.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta5.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta5.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta5.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta5",
			},
		},
		// upgrade drive v1beta1 => v1beta4
		{
			srcObject: &directv1beta1.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta1", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta1.DirectCSIDriveFinalizerDataProtection),
						directv1beta1.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta1.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta1.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta1.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			destObject: &directv1beta4.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta4", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta4.DirectCSIDriveFinalizerDataProtection),
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta4.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta4.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
					PCIPath:           "pci-0000:2e:00.0-nvme-1",
					SerialNumberLong:  "KXG6AZNV512G TOSHIBA_31IF73XDFDM3",
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta4",
			},
		},
		// upgrade drive v1beta1 => v1beta3
		{
			srcObject: &directv1beta1.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta1", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta1.DirectCSIDriveFinalizerDataProtection),
						directv1beta1.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta1.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta1.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta1.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			destObject: &directv1beta3.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta3.DirectCSIDriveFinalizerDataProtection),
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta3.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta3.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta3",
			},
		},
		// upgrade drive v1beta2 => v1beta5
		{
			srcObject: &directv1beta2.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta2", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta2.DirectCSIDriveFinalizerDataProtection),
						directv1beta2.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta2.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta2.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta2.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			destObject: &directv1beta5.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta5", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta5.DirectCSIDriveFinalizerDataProtection),
						directv1beta5.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta5.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta5.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta5.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta5",
			},
		},
		// upgrade drive v1beta2 => v1beta4
		{
			srcObject: &directv1beta2.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta2", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta2.DirectCSIDriveFinalizerDataProtection),
						directv1beta2.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta2.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta2.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta2.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			destObject: &directv1beta4.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta4", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta4.DirectCSIDriveFinalizerDataProtection),
						directv1beta4.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta4.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta4.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta4.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
					PCIPath:           "pci-0000:2e:00.0-nvme-1",
					SerialNumberLong:  "KXG6AZNV512G TOSHIBA_31IF73XDFDM3",
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta4",
			},
		},
		// upgrade drive v1beta2 => v1beta3
		{
			srcObject: &directv1beta2.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta2", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta2.DirectCSIDriveFinalizerDataProtection),
						directv1beta2.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta2.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta2.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta2.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			destObject: &directv1beta3.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta3.DirectCSIDriveFinalizerDataProtection),
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta3.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta3.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta3",
			},
		},
		// downgrade drive v1beta5 => v1beta1
		{
			srcObject: &directv1beta4.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta4", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta4.DirectCSIDriveFinalizerDataProtection),
						directv1beta4.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta4.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta4.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta4.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
					PCIPath:           "pci-0000:2e:00.0-nvme-1",
					SerialNumberLong:  "KXG6AZNV512G TOSHIBA_31IF73XDFDM3",
				},
			},
			destObject: &directv1beta1.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta1", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta1.DirectCSIDriveFinalizerDataProtection),
						directv1beta1.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta1.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta1.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta1.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta1",
			},
		},
		// downgrade drive v1beta4 => v1beta1
		{
			srcObject: &directv1beta4.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta4", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta4.DirectCSIDriveFinalizerDataProtection),
						directv1beta4.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta4.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta4.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta4.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
					PCIPath:           "pci-0000:2e:00.0-nvme-1",
					SerialNumberLong:  "KXG6AZNV512G TOSHIBA_31IF73XDFDM3",
				},
			},
			destObject: &directv1beta1.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta1", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta1.DirectCSIDriveFinalizerDataProtection),
						directv1beta1.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta1.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta1.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta1.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta1",
			},
		},
		// downgrade drive v1beta5 => v1beta2
		{
			srcObject: &directv1beta5.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta5", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta5.DirectCSIDriveFinalizerDataProtection),
						directv1beta5.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta5.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta5.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta5.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			destObject: &directv1beta2.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta2", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta2.DirectCSIDriveFinalizerDataProtection),
						directv1beta2.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta2.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta2.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta2.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta2",
			},
		},
		// downgrade drive v1beta4 => v1beta2
		{
			srcObject: &directv1beta4.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta4", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta4.DirectCSIDriveFinalizerDataProtection),
						directv1beta4.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta4.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta4.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta4.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
					PCIPath:           "pci-0000:2e:00.0-nvme-1",
					SerialNumberLong:  "KXG6AZNV512G TOSHIBA_31IF73XDFDM3",
				},
			},
			destObject: &directv1beta2.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta2", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta2.DirectCSIDriveFinalizerDataProtection),
						directv1beta2.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta2.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta2.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta2.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta2",
			},
		},
		// downgrade drive v1beta5 => v1beta3
		{
			srcObject: &directv1beta5.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta5", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta5.DirectCSIDriveFinalizerDataProtection),
						directv1beta5.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta5.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta5.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta5.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			destObject: &directv1beta3.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta3.DirectCSIDriveFinalizerDataProtection),
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta3.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta3.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta3",
			},
		},
		// downgrade drive v1beta4 => v1beta3
		{
			srcObject: &directv1beta4.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta4", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta4.DirectCSIDriveFinalizerDataProtection),
						directv1beta4.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta4.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta4.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta4.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
					PCIPath:           "pci-0000:2e:00.0-nvme-1",
					SerialNumberLong:  "KXG6AZNV512G TOSHIBA_31IF73XDFDM3",
				},
			},
			destObject: &directv1beta3.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta3.DirectCSIDriveFinalizerDataProtection),
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta3.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta3.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta3",
			},
		},

		// downgrade drive v1beta3 => v1beta1
		{
			srcObject: &directv1beta3.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta3.DirectCSIDriveFinalizerDataProtection),
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta3.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta3.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			destObject: &directv1beta1.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta1", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta1.DirectCSIDriveFinalizerDataProtection),
						directv1beta1.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta1.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta1.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta1.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta1",
			},
		},
		// downgrade drive v1beta3 => v1beta2
		{
			srcObject: &directv1beta3.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta3.DirectCSIDriveFinalizerDataProtection),
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta3.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta3.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			destObject: &directv1beta2.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta2", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta2.DirectCSIDriveFinalizerDataProtection),
						directv1beta2.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta2.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta2.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta2.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta2",
			},
		},
		// upgrage volume v1beta1 => v1beta5
		{
			srcObject: &directv1beta1.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta1", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta1.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta1.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta5.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta5", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta5.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta5.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta5",
			},
		},
		// upgrage volume v1beta1 => v1beta4
		{
			srcObject: &directv1beta1.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta1", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta1.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta1.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta4.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta4", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta4.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta4.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta4",
			},
		},
		// upgrage volume v1beta2 => v1beta5
		{
			srcObject: &directv1beta2.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta2", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta2.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta2.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta5.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta5", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta5.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta5.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta5",
			},
		},
		// upgrage volume v1beta2 => v1beta4
		{
			srcObject: &directv1beta2.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta2", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta2.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta2.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta4.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta4", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta4.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta4.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta4",
			},
		},
		// upgrage volume v1beta1 => v1beta3
		{
			srcObject: &directv1beta1.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta1", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta1.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta1.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta3.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta3.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta3.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta3",
			},
		},
		// upgrage volume v1beta2 => v1beta3
		{
			srcObject: &directv1beta2.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta2", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta2.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta2.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta3.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta3.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta3.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta3",
			},
		},
		// downgrade volume v1beta5 => v1beta1
		{
			srcObject: &directv1beta5.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta5", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta5.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta5.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta1.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta1", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta1.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta1.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta1",
			},
		},
		// downgrade volume v1beta4 => v1beta1
		{
			srcObject: &directv1beta4.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta4", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta4.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta4.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta1.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta1", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta1.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta1.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta1",
			},
		},
		// downgrage volume v1beta5 => v1beta2
		{
			srcObject: &directv1beta5.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta5", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta5.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta5.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta2.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta2", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta2.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta2.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta2",
			},
		},
		// downgrage volume v1beta4 => v1beta2
		{
			srcObject: &directv1beta4.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta4", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta4.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta4.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta2.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta2", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta2.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta2.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta2",
			},
		},
		// downgrage volume v1beta5 => v1beta3
		{
			srcObject: &directv1beta5.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta5", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta5.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta5.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta3.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta3.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta3.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta3",
			},
		},
		// downgrage volume v1beta4 => v1beta3
		{
			srcObject: &directv1beta4.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta4", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta4.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta4.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta3.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta3.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta3.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta3",
			},
		},

		// downgrade volume v1beta3 => v1beta1
		{
			srcObject: &directv1beta3.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta3.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta3.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta1.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta1", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta1.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta1.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta1",
			},
		},
		// downgrage volume v1beta3 => v1beta2
		{
			srcObject: &directv1beta3.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta3.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta3.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta2.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta2", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta2.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta2.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta2",
			},
		},
	}

	for i, test := range testCases {
		objBytes, err := json1.Marshal(test.srcObject)
		if err != nil {
			t.Fatalf("failed to marshaling source object: %v", test.srcObject)
		}
		cr := unstructured.Unstructured{}
		if err := cr.UnmarshalJSON(objBytes); err != nil {
			t.Fatalf("failed to umarshaling source object: %v", test.srcObject)
		}
		result := &unstructured.Unstructured{}

		err = Migrate(&cr, result, test.groupVersion)
		if err != nil {
			t.Fatalf("failed to convert runtime object: %v", err)
		}
		gv := result.GetObjectKind().GroupVersionKind().GroupVersion().String()
		if gv != test.destObject.GetObjectKind().GroupVersionKind().GroupVersion().String() {
			t.Fatalf("Test  %d failed wrong group version: %s, expected: %s", i+1, gv, test.groupVersion.Version)
		}
	}
}
