// This file is part of MinIO DirectPV
// Copyright (c) 2022 MinIO, Inc.
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
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"github.com/minio/directpv/pkg/apis/directpv.min.io/types"
)

// Device is a block device information.
type Device struct {
	Name        string            `json:"name"`        // Read from /sys/dev/block/<Major:Minor>/uevent
	MajorMinor  string            `json:"majorMinor"`  // Read from /run/udev/data
	Size        uint64            `json:"size"`        // Read from /sys/class/block/<NAME>/size
	Hidden      bool              `json:"hidden"`      // Read from /sys/class/block/<NAME>/hidden
	Removable   bool              `json:"removable"`   // Read from /sys/class/block/<NAME>/removable
	ReadOnly    bool              `json:"readOnly"`    // Read from /sys/class/block/<NAME>/ro
	Partitioned bool              `json:"partitioned"` // Read from /sys/block/<NAME>/<NAME>*
	Holders     []string          `json:"holders"`     // Read from /sys/class/block/<NAME>/holders
	MountPoints []string          `json:"mountPoints"` // Read from /proc/1/mountinfo or /proc/mounts
	SwapOn      bool              `json:"swapOn"`      // Read from /proc/swaps
	CDROM       bool              `json:"cdrom"`       // Read from /proc/sys/dev/cdrom/info
	DMName      string            `json:"dmName"`      // Read from /sys/class/block/<NAME>/dm/name
	UDevData    map[string]string `json:"udevData"`    // Read from /run/udev/data/b<Major:Minor>
}

// ID generates a unique ID by hashing the properties of the Device.
func (d *Device) ID(nodeID types.NodeID) string {
	sort.Strings(d.Holders)
	sort.Strings(d.MountPoints)

	deviceMap := map[string]string{
		"node":        string(nodeID),
		"name":        d.Name,
		"majorminor":  d.MajorMinor,
		"size":        fmt.Sprintf("%v", d.Size),
		"hidden":      fmt.Sprintf("%v", d.Hidden),
		"removable":   fmt.Sprintf("%v", d.Removable),
		"readonly":    fmt.Sprintf("%v", d.ReadOnly),
		"partitioned": fmt.Sprintf("%v", d.Partitioned),
		"holders":     strings.Join(d.Holders, ","),
		"mountpoints": strings.Join(d.MountPoints, ","),
		"swapon":      fmt.Sprintf("%v", d.SwapOn),
		"cdrom":       fmt.Sprintf("%v", d.CDROM),
		"dmname":      d.DMName,
		"udevdata":    strings.Join(toSlice(d.UDevData, "="), ";"),
	}

	stringToHash := strings.Join(toSlice(deviceMap, ":"), "\n")
	h := sha256.Sum256([]byte(stringToHash))
	return base64.StdEncoding.EncodeToString(h[:])
}

// Make returns device make information.
func (d Device) Make() string {
	var tokens []string

	if d.DMName != "" {
		tokens = append(tokens, d.DMName)
	}

	if d.UDevData["ID_VENDOR"] != "" {
		tokens = append(tokens, d.UDevData["ID_VENDOR"])
	}

	if d.UDevData["ID_MODEL"] != "" {
		tokens = append(tokens, d.UDevData["ID_MODEL"])
	}

	if number, found := d.UDevData["ID_PART_ENTRY_NUMBER"]; found {
		tokens = append(tokens, fmt.Sprintf("(Part %v)", number))
	}

	return strings.Join(tokens, " ")
}

// FSType returns filesystem type.
func (d Device) FSType() string {
	return d.UDevData["ID_FS_TYPE"]
}

// FSUUID returns the filesystem UUID.
func (d Device) FSUUID() string {
	return d.UDevData["ID_FS_UUID"]
}

// Probe returns block devices from udev.
func Probe() ([]Device, error) {
	return probe()
}

// ProbeDevices returns block devices from udev.
func ProbeDevices(majorMinor ...string) ([]Device, error) {
	return probeDevices(majorMinor...)
}
