/*
 * This file is part of MinIO Direct CSI
 * Copyright (C) 2021, MinIO, Inc.
 *
 * This code is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License, version 3,
 * as published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License, version 3,
 * along with this program.  If not, see <http://www.gnu.org/licenses/>
 *
 */

package main

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	clientsetfake "github.com/minio/direct-csi/pkg/clientset/fake"
	directcsifake "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1beta3/fake"
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
		objM := utils.NewObjectMeta(
			drive,
			metav1.NamespaceNone,
			map[string]string{
				utils.NodeLabel:      utils.SanitizeLabelV(node),
				utils.DrivePathLabel: utils.SanitizeDrivePath(path),
			},
			map[string]string{},
			[]string{
				string(directcsi.DirectCSIDriveFinalizerDataProtection),
			},
			nil,
		)

		utils.SetAccessTierLabel(&objM, accessTier)
		return &directcsi.DirectCSIDrive{
			TypeMeta:   utils.DirectCSIDriveTypeMeta(),
			ObjectMeta: objM,
			Status: directcsi.DirectCSIDriveStatus{
				Path:              path,
				NodeName:          node,
				Filesystem:        string(sys.FSTypeXFS),
				DriveStatus:       driveStatus,
				FreeCapacity:      mb100,
				AllocatedCapacity: int64(0),
				TotalCapacity:     mb100,
				Topology:          getTopologySegmentsForNode(node),
				AccessTier:        accessTier,
			},
		}
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
	testClient := testClientSet.DirectV1beta3()
	utils.SetDirectCSIClient(testClient.(*directcsifake.FakeDirectV1beta3))

	resetDrives := func() error {
		driveList, err := utils.GetDriveList(ctx, testClient.DirectCSIDrives(), nil, nil, nil)
		if err != nil {
			return err
		}

		for _, drive := range driveList {
			drive.Spec.RequestedFormat = nil
			if _, err := testClient.DirectCSIDrives().Update(ctx, &drive, metav1.UpdateOptions{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
			}); err != nil {
				return err
			}
		}

		return nil
	}

	getFormattedDrives := func() ([]string, error) {
		driveList, err := utils.GetDriveList(ctx, testClient.DirectCSIDrives(), nil, nil, nil)
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
