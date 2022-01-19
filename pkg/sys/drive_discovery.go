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

package sys

// GetMajorMinor gets major/minor number of given device.
func GetMajorMinor(device string) (major, minor uint32, err error) {
	return getDeviceMajorMinor(device)
}

// ProbeDevices probes all devices.
func ProbeDevices() (devices map[string]*Device, err error) {
	return probeDevices()
}

// CreateDevice creates new device structure for given event data.
func CreateDevice(event map[string]string) (device *Device, err error) {
	return createDevice(event)
}

// GetDeviceName returns device name of given major/minor number.
func GetDeviceName(major, minor uint32) (string, error) {
	return getDeviceName(major, minor)
}
