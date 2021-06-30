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
	"reflect"
	"strings"
	"testing"

	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	fakedirect "github.com/minio/direct-csi/pkg/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

const (
	KB    = 1 << 10
	MB    = KB << 10
	mb100 = 100 * MB
)

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
		return &directcsi.DirectCSIDrive{
			TypeMeta: utils.DirectCSIDriveTypeMeta(strings.Join([]string{directcsi.Group, directcsi.Version}, "/")),
			ObjectMeta: metav1.ObjectMeta{
				Name: drive,
				Finalizers: []string{
					string(directcsi.DirectCSIDriveFinalizerDataProtection),
				},
			},
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
		// Drives from Node N1
		createTestDrive("N1", "D1", "/var/lib/direct-csi/devices/xvdb", directcsi.DriveStatusAvailable, directcsi.AccessTierUnknown),
		createTestDrive("N1", "D2", "/var/lib/direct-csi/devices/xvdc", directcsi.DriveStatusAvailable, directcsi.AccessTierCold),
		createTestDrive("N1", "D3", "/var/lib/direct-csi/devices/xvda", directcsi.DriveStatusUnavailable, directcsi.AccessTierUnknown),
		// Drives from Node N2
		createTestDrive("N2", "D4", "/var/lib/direct-csi/devices/xvdb", directcsi.DriveStatusAvailable, directcsi.AccessTierUnknown),
		createTestDrive("N2", "D5", "/var/lib/direct-csi/devices/xvdc", directcsi.DriveStatusAvailable, directcsi.AccessTierCold),
		createTestDrive("N2", "D6", "/var/lib/direct-csi/devices/xvda", directcsi.DriveStatusUnavailable, directcsi.AccessTierUnknown),
		createTestDrive("N2", "D7", "/var/lib/direct-csi/devices/xvdh", directcsi.DriveStatusReleased, directcsi.AccessTierUnknown),
	}

	utils.SetFake()
	ctx := context.TODO()
	testClientSet := fakedirect.NewSimpleClientset(testDriveObjects...)
	testClient := testClientSet.DirectV1beta2()
	utils.SetFakeDirectCSIClient(testClient)

	resetDrives := func() error {
		driveList, err := testClient.DirectCSIDrives().List(ctx, metav1.ListOptions{
			TypeMeta: utils.DirectCSIDriveTypeMeta(strings.Join([]string{directcsi.Group, directcsi.Version}, "/")),
		})
		if err != nil {
			return err
		}

		for _, drive := range driveList.Items {
			drive.Spec.RequestedFormat = nil
			if _, err := testClient.DirectCSIDrives().Update(ctx, &drive, metav1.UpdateOptions{
				TypeMeta: utils.DirectCSIDriveTypeMeta(strings.Join([]string{directcsi.Group, directcsi.Version}, "/")),
			}); err != nil {
				return err
			}
		}

		return nil
	}

	getDrivesWithRequestedFormat := func() ([]string, error) {
		driveList, err := testClient.DirectCSIDrives().List(ctx, metav1.ListOptions{
			TypeMeta: utils.DirectCSIDriveTypeMeta(strings.Join([]string{directcsi.Group, directcsi.Version}, "/")),
		})
		if err != nil {
			return []string{}, err
		}

		formattedDriveNames := []string{}
		for _, drive := range driveList.Items {
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
		unrelease      bool
		expectedDrives []string
	}{
		{
			name:           "test-format-by-drives",
			drives:         []string{"/dev/xvdc"},
			nodes:          []string{},
			accessTiers:    []string{},
			all:            false,
			force:          true,
			unrelease:      false,
			expectedDrives: []string{"D2", "D5"},
		},
		{
			name:           "test-format-by-nodes",
			drives:         []string{},
			nodes:          []string{"N1"},
			accessTiers:    []string{},
			all:            false,
			force:          true,
			unrelease:      false,
			expectedDrives: []string{"D1", "D2"},
		},
		{
			name:           "test-format-by-accesstiers",
			drives:         []string{},
			nodes:          []string{},
			accessTiers:    []string{"cold"},
			all:            false,
			force:          true,
			unrelease:      false,
			expectedDrives: []string{"D2", "D5"},
		},
		{
			name:           "test-format-by-multiple-params",
			drives:         []string{"/dev/xvdc"},
			nodes:          []string{"N2"},
			accessTiers:    []string{"cold"},
			all:            false,
			force:          true,
			unrelease:      false,
			expectedDrives: []string{"D5"},
		},
		{
			name:           "test-format-all-drives",
			drives:         []string{},
			nodes:          []string{},
			accessTiers:    []string{},
			all:            true,
			force:          true,
			unrelease:      true,
			expectedDrives: []string{"D1", "D2", "D4", "D5", "D7"},
		},
	}

	for _, tt := range testCases {
		t1.Run(tt.name, func(t1 *testing.T) {
			drives = tt.drives
			nodes = tt.nodes
			accessTiers = tt.accessTiers
			all = tt.all
			force = tt.force
			unrelease = tt.unrelease

			if err := formatDrives(ctx, []string{}); err != nil {
				t1.Errorf("Test case name %s: Failed with %v", tt.name, err)
			}

			driveList, err := getDrivesWithRequestedFormat()
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
