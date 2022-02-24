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

package discovery

import (
	"errors"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
)

var (
	errNoMatchFound = errors.New("no matching drive found")
)

func (d *Discovery) identify(localDriveState directcsi.DirectCSIDriveStatus) (*remoteDrive, error) {
	if len(d.remoteDrives) > 0 {
		return d.identifyDriveByAttributes(localDriveState)
	}
	return nil, errNoMatchFound
}

func (d *Discovery) identifyDriveByAttributes(localDriveState directcsi.DirectCSIDriveStatus) (*remoteDrive, error) {
	if selectedDrive, err := d.selectByFSUUID(localDriveState.FilesystemUUID); err == nil {
		return selectedDrive, nil
	}
	if selectedDrive, err := d.selectByPartitionUUID(localDriveState.PartitionUUID); err == nil {
		return selectedDrive, nil
	}
	return nil, errNoMatchFound
}

func (d *Discovery) selectByFSUUID(fsUUID string) (*remoteDrive, error) {
	if fsUUID == "" {
		// No FSUUID available to match
		return nil, errNoMatchFound
	}
	for i, remoteDrive := range d.remoteDrives {
		if !remoteDrive.matched && remoteDrive.Status.FilesystemUUID == fsUUID {
			d.remoteDrives[i].matched = true
			return d.remoteDrives[i], nil
		}
	}
	return nil, errNoMatchFound
}

func (d *Discovery) selectByPartitionUUID(partUUID string) (*remoteDrive, error) {
	if partUUID == "" {
		// No partUUID available to match
		return nil, errNoMatchFound
	}
	for i, remoteDrive := range d.remoteDrives {
		if !remoteDrive.matched && remoteDrive.Status.PartitionUUID == partUUID {
			d.remoteDrives[i].matched = true
			return d.remoteDrives[i], nil
		}
	}
	return nil, errNoMatchFound
}

func (d *Discovery) identifyDriveByLegacyName(localDriveState directcsi.DirectCSIDriveStatus) (*remoteDrive, bool, error) {
	isNotUpgraded := func(driveObj directcsi.DirectCSIDrive) bool {
		return driveObj.Status.SerialNumber == "" &&
			driveObj.Status.FilesystemUUID == "" &&
			driveObj.Status.PartitionUUID == "" &&
			driveObj.Status.MajorNumber == uint32(0) &&
			driveObj.Status.MinorNumber == uint32(0)
	}

	for i, remoteDrive := range d.remoteDrives {
		if !remoteDrive.matched && remoteDrive.Status.Path == localDriveState.Path {
			notUpgraded := isNotUpgraded(remoteDrive.DirectCSIDrive)
			if notUpgraded {
				d.remoteDrives[i].matched = true
			}
			return d.remoteDrives[i], notUpgraded, nil
		}
	}
	return nil, false, errNoMatchFound
}
