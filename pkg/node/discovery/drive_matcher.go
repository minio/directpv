// This file is part of MinIO Direct CSI
// Copyright (c) 2021 MinIO, Inc.
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

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
)

var (
	ErrNoMatchFound = errors.New("No matching drive found")
)

func (d *Discovery) Identify(localDriveState directcsi.DirectCSIDriveStatus) (*remoteDrive, error) {
	if len(d.remoteDrives) > 0 {
		return d.identifyDriveByAttributes(localDriveState)
	}
	return nil, ErrNoMatchFound
}

func (d *Discovery) identifyDriveByAttributes(localDriveState directcsi.DirectCSIDriveStatus) (*remoteDrive, error) {
	if selectedDrive, err := d.selectByFSUUID(localDriveState.FilesystemUUID); err == nil {
		return selectedDrive, nil
	}
	if selectedDrive, err := d.selectByPartitionUUID(localDriveState.PartitionUUID); err == nil {
		return selectedDrive, nil
	}
	if selectedDrive, err := d.selectBySerialNumber(localDriveState.SerialNumber, localDriveState.PartitionNum); err == nil {
		return selectedDrive, nil
	}
	return nil, ErrNoMatchFound
}

func (d *Discovery) selectByFSUUID(fsUUID string) (*remoteDrive, error) {
	if fsUUID == "" {
		// No FSUUID available to match
		return nil, ErrNoMatchFound
	}
	for i, remoteDrive := range d.remoteDrives {
		if !remoteDrive.matched && remoteDrive.Status.FilesystemUUID == fsUUID {
			d.remoteDrives[i].matched = true
			return d.remoteDrives[i], nil
		}
	}
	return nil, ErrNoMatchFound
}

func (d *Discovery) selectByPartitionUUID(partUUID string) (*remoteDrive, error) {
	if partUUID == "" {
		// No partUUID available to match
		return nil, ErrNoMatchFound
	}
	for i, remoteDrive := range d.remoteDrives {
		if !remoteDrive.matched && remoteDrive.Status.PartitionUUID == partUUID {
			d.remoteDrives[i].matched = true
			return d.remoteDrives[i], nil
		}
	}
	return nil, ErrNoMatchFound
}

func (d *Discovery) selectBySerialNumber(serialNumber string, partitionNum int) (*remoteDrive, error) {
	if serialNumber == "" {
		// No serialNumber available to match
		return nil, ErrNoMatchFound
	}
	for i, remoteDrive := range d.remoteDrives {
		if !remoteDrive.matched && remoteDrive.Status.SerialNumber == serialNumber && remoteDrive.Status.PartitionNum == partitionNum {
			d.remoteDrives[i].matched = true
			return d.remoteDrives[i], nil
		}
	}
	return nil, ErrNoMatchFound
}

func (d *Discovery) identifyDriveByLegacyName(localDriveState directcsi.DirectCSIDriveStatus) (*remoteDrive, error) {
	v1beta1DriveName := makeV1beta1DriveName(d.NodeID, localDriveState.Path)

	for i, remoteDrive := range d.remoteDrives {
		if !remoteDrive.matched && remoteDrive.Name == v1beta1DriveName {
			d.remoteDrives[i].matched = true
			return d.remoteDrives[i], nil
		}
	}
	return nil, ErrNoMatchFound
}
