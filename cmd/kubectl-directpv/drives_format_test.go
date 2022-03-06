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
	"os"
	"reflect"
	"testing"

	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/utils"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

const (
	KB    = 1 << 10
	MB    = KB << 10
	mb100 = 100 * MB
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestFormatDrivesByAttributes(t1 *testing.T) {
	getTopologySegmentsForNode := func(node string) map[string]string {
		switch node {
		case "N1":
			return map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R1"}
		case "N2":
			return map[string]string{"node": "N2", "rack": "RK2", "zone": "Z2", "region": "R2"}
		default:
			return map[string]string{}
		}
	}

	createTestDrive := func(node, drive, path string, driveStatus directcsi.DriveStatus, accessTier directcsi.AccessTier) *directcsi.DirectCSIDrive {
		csiDrive := &directcsi.DirectCSIDrive{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name:      drive,
				Namespace: metav1.NamespaceNone,
				Labels: map[string]string{
					string(utils.NodeLabelKey): string(utils.NewLabelValue(node)),
					string(utils.PathLabelKey): string(utils.NewLabelValue(utils.SanitizeDrivePath(path))),
				},
				Finalizers: []string{string(directcsi.DirectCSIDriveFinalizerDataProtection)},
			},
			Status: directcsi.DirectCSIDriveStatus{
				Path:              path,
				NodeName:          node,
				Filesystem:        "xfs",
				DriveStatus:       driveStatus,
				FreeCapacity:      mb100,
				AllocatedCapacity: int64(0),
				TotalCapacity:     mb100,
				Topology:          getTopologySegmentsForNode(node),
				AccessTier:        accessTier,
			},
		}
		setDriveAccessTier(csiDrive, accessTier)
		return csiDrive
	}

	testDriveObjects := []runtime.Object{
		// Drives from Node n1
		createTestDrive("n1", "d1", "/var/lib/direct-csi/devices/xvdb", directcsi.DriveStatusAvailable, directcsi.AccessTierUnknown),
		createTestDrive("n1", "d2", "/var/lib/direct-csi/devices/xvdc", directcsi.DriveStatusAvailable, directcsi.AccessTierCold),
		createTestDrive("n1", "d3", "/var/lib/direct-csi/devices/xvda", directcsi.DriveStatusUnavailable, directcsi.AccessTierUnknown),

		// Drives from Node n2
		createTestDrive("n2", "d4", "/var/lib/direct-csi/devices/xvdb", directcsi.DriveStatusAvailable, directcsi.AccessTierUnknown),
		createTestDrive("n2", "d5", "/var/lib/direct-csi/devices/xvdc", directcsi.DriveStatusAvailable, directcsi.AccessTierCold),
		createTestDrive("n2", "d6", "/var/lib/direct-csi/devices/xvda", directcsi.DriveStatusUnavailable, directcsi.AccessTierUnknown),
		createTestDrive("n2", "d7", "/var/lib/direct-csi/devices/xvdh", directcsi.DriveStatusAvailable, directcsi.AccessTierUnknown),
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	testClientSet := clientsetfake.NewSimpleClientset(testDriveObjects...)
	driveInterface := testClientSet.().DirectCSIDrives()
	client.SetLatestDirectCSIDriveInterface(driveInterface)

	resetDrives := func() error {
		driveList, err := client.GetDriveList(ctx, nil, nil, nil)
		if err != nil {
			return err
		}

		for _, drive := range driveList {
			drive.Spec.RequestedFormat = nil
			if _, err := driveInterface.Update(ctx, &drive, metav1.UpdateOptions{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
			}); err != nil {
				return err
			}
		}

		return nil
	}

	getFormattedDrives := func() ([]string, error) {
		driveList, err := client.GetDriveList(ctx, nil, nil, nil)
		if err != nil {
			return []string{}, err
		}

		formattedDriveNames := []string{}
		for _, drive := range driveList {
			if drive.Spec.RequestedFormat != nil {
				formattedDriveNames = append(formattedDriveNames, drive.Name)
			}
		}

		return formattedDriveNames, nil
	}

	testCases := []struct {
		name           string
		drives         []string
		nodes          []string
		accessTiers    []string
		all            bool
		force          bool
		expectedDrives []string
	}{
		{
			name:           "test-format-by-drives",
			drives:         []string{"/dev/xvdc"},
			nodes:          []string{},
			accessTiers:    []string{},
			all:            false,
			force:          true,
			expectedDrives: []string{"d2", "d5"},
		},
		{
			name:           "test-format-by-nodes",
			drives:         []string{},
			nodes:          []string{"n1"},
			accessTiers:    []string{},
			all:            false,
			force:          true,
			expectedDrives: []string{"d1", "d2"},
		},
		{
			name:           "test-format-by-accesstiers",
			drives:         []string{},
			nodes:          []string{},
			accessTiers:    []string{string(directcsi.AccessTierCold)},
			all:            false,
			force:          true,
			expectedDrives: []string{"d2", "d5"},
		},
		{
			name:           "test-format-by-multiple-params",
			drives:         []string{"/dev/xvdc"},
			nodes:          []string{"n2"},
			accessTiers:    []string{string(directcsi.AccessTierCold)},
			all:            false,
			force:          true,
			expectedDrives: []string{"d5"},
		},
		{
			name:           "test-format-all-drives",
			drives:         []string{},
			nodes:          []string{},
			accessTiers:    []string{},
			all:            true,
			force:          true,
			expectedDrives: []string{"d1", "d2", "d4", "d5", "d7"},
		},
		{
			name:           "test-format-drives-by-ellipses-selectors",
			drives:         []string{"/dev/xvd{b...c}"},
			nodes:          []string{"n1"},
			accessTiers:    []string{},
			force:          true,
			expectedDrives: []string{"d1", "d2"},
		},
	}

	for _, tt := range testCases {
		t1.Run(tt.name, func(t1 *testing.T) {
			drives = tt.drives
			nodes = tt.nodes
			accessTiers = tt.accessTiers
			all = tt.all
			force = tt.force

			if err := validateDriveSelectors(); err != nil {
				t1.Fatalf("Test case name %s: validateDriveSelectors failed with %v", tt.name, err)
			}

			if err := formatDrives(ctx, []string{}); err != nil {
				t1.Errorf("Test case name %s: Failed with %v", tt.name, err)
			}

			driveList, err := getFormattedDrives()
			if err != nil {
				t1.Errorf("Test case name %s: Failed while fetching the drives %v", tt.name, err)
			}

			if !reflect.DeepEqual(driveList, tt.expectedDrives) {
				t1.Errorf("Test case name %s: Expected formatted drive list: %v But, got: %v", tt.name, tt.expectedDrives, driveList)
			}

			if err := resetDrives(); err != nil {
				t1.Errorf("Test case name %s: Error while resetting the drives %v", tt.name, err)
			}
		})
	}
}
