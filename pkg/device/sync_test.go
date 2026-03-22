// This file is part of MinIO DirectPV
// Copyright (c) 2023 MinIO, Inc.
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

package device

import (
	"testing"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/types"
)

func newTestDevice(name string, totalCapacity int64, dmname string) device {
	return device{
		TotalCapacity: totalCapacity,
		Device: Device{
			Name:   name,
			DMName: dmname,
		},
	}
}

func TestSyncDrive(t *testing.T) {
	newDrive := func(totalCapacity int64, make, volume string) *types.Drive {
		drive := types.NewDrive(
			directpvtypes.DriveID("sda-id"),
			types.DriveStatus{
				TotalCapacity: totalCapacity,
				Make:          make,
			},
			directpvtypes.NodeID("nodeId"),
			directpvtypes.DriveName("sda"),
			directpvtypes.AccessTierDefault,
		)
		drive.AddVolumeFinalizer(volume)
		return drive
	}

	testCases := []struct {
		drive                 *types.Drive
		device                device
		updated               bool
		expectedDriveName     directpvtypes.DriveName
		expectedDriveCapacity int64
		expectedMake          string
	}{
		{
			drive:                 newDrive(100, "dmname", "volume-1"),
			device:                newTestDevice("sda", 100, "dmname"),
			updated:               false,
			expectedDriveName:     "sda",
			expectedDriveCapacity: 100,
			expectedMake:          "dmname",
		},
		{
			drive:                 newDrive(100, "dmname", "volume-1"),
			device:                newTestDevice("sda", 200, "dmname"),
			updated:               true,
			expectedDriveName:     "sda",
			expectedDriveCapacity: 200,
			expectedMake:          "dmname",
		},
		{
			drive:                 newDrive(100, "dmname", "volume-1"),
			device:                newTestDevice("sda", 100, "dmname-new"),
			updated:               true,
			expectedDriveName:     "sda",
			expectedDriveCapacity: 100,
			expectedMake:          "dmname-new",
		},
		{
			drive:                 newDrive(100, "dmname", "volume-1"),
			device:                newTestDevice("sdb", 100, "dmname"),
			updated:               true,
			expectedDriveName:     "sdb",
			expectedDriveCapacity: 100,
			expectedMake:          "dmname",
		},
		{
			drive:                 newDrive(100, "dmname", "volume-1"),
			device:                newTestDevice("sda", 100, "dmname"),
			updated:               false,
			expectedDriveName:     "sda",
			expectedDriveCapacity: 100,
			expectedMake:          "dmname",
		},
	}

	for _, testCase := range testCases {
		updated := syncDrive(testCase.drive, testCase.device)
		if updated != testCase.updated {
			t.Errorf("expected updated value: %v; but got %v", testCase.updated, updated)
		}
		if testCase.drive.GetDriveName() != testCase.expectedDriveName {
			t.Errorf("expected drive name: %v; but got %v", testCase.expectedDriveName, testCase.drive.GetDriveName())
		}
		if testCase.drive.Status.TotalCapacity != testCase.expectedDriveCapacity {
			t.Errorf("expected drive capacity: %v; but got %v", testCase.expectedDriveCapacity, testCase.drive.Status.TotalCapacity)
		}
		if testCase.drive.Status.Make != testCase.expectedMake {
			t.Errorf("expected drive make: %v; but got %v", testCase.expectedMake, testCase.drive.Status.Make)
		}
	}
}
