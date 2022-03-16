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

package drive

import (
	"github.com/google/uuid"
	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/sys"
)

func isFormatRequested(drive *directcsi.DirectCSIDrive) bool {
	return drive.Spec.DirectCSIOwned &&
		drive.Spec.RequestedFormat != nil &&
		drive.Status.DriveStatus == directcsi.DriveStatusAvailable
}

func getDevice(major, minor uint32) (string, error) {
	name, err := sys.GetDeviceName(major, minor)
	if err != nil {
		return "", err
	}
	return "/dev/" + name, nil
}

func getFSUUIDFromDrive(drive *directcsi.DirectCSIDrive) string {
	fsuuid, err := uuid.Parse(drive.Name)
	if err != nil {
		fsuuid = uuid.New()
	}
	return fsuuid.String()
}

// verify if the drive states match the host info
// --------------------------------------------------------------------------
// - read /run/udev/data/b<maj:min> (refer ReadRunUdevDataByMajorMinor and mapToUdevData funcs)
//   return err if the file does not exist
// - construct sys.Device (refer /pkg/uevent/listener.go)
// - validate the device (refer validateDevice function in pkg uevent)
// - return an error if validation fails
//
// If validation succeeds,
// For Ready, InUse drives
//      - check if the target (/var/lib/direct-csi/mnt/<drive.Name>) is mounted.
//            If mounted, check the mount options, If not matching, return errInvalidMountOptions
//            Else, return errNotMounted
// ----------------------------------------------------------------------------
func verifyHostStateForDrive(drive *directcsi.DirectCSIDrive) error {
	return nil
}
