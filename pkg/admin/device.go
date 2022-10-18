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

package admin

import (
	"context"
	"strings"

	pkgclient "github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/device"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const minSupportedDeviceSize = 512 * 1024 * 1024 // 512 MiB

type Device struct {
	device.Device
	FormatDenied bool   `json:"formatDenied,omitempty"`
	DeniedReason string `json:"deniedReason,omitempty"`
}

func newDevice(dev device.Device) Device {
	var reasons []string

	if dev.Size < minSupportedDeviceSize {
		reasons = append(reasons, "Too small")
	}

	if dev.Hidden {
		reasons = append(reasons, "Hidden")
	}

	if dev.ReadOnly {
		reasons = append(reasons, "Read only")
	}

	if dev.Partitioned {
		reasons = append(reasons, "Partitioned")
	}

	if len(dev.Holders) != 0 {
		reasons = append(reasons, "Held by other device")
	}

	if len(dev.MountPoints) != 0 {
		reasons = append(reasons, "Mounted")
	}

	if dev.SwapOn {
		reasons = append(reasons, "Swap")
	}

	if dev.CDROM {
		reasons = append(reasons, "CDROM")
	}

	if dev.UDevData["ID_FS_TYPE"] == "xfs" && dev.UDevData["ID_FS_UUID"] != "" {
		if _, err := pkgclient.DriveClient().Get(context.Background(), dev.UDevData["ID_FS_UUID"], metav1.GetOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				reasons = append(reasons, "internal error; "+err.Error())
			}
		} else {
			reasons = append(reasons, "Used by "+consts.AppPrettyName)
		}
	}

	var deniedReason string
	if len(reasons) != 0 {
		deniedReason = strings.Join(reasons, "; ")
	}
	return Device{
		Device:       dev,
		FormatDenied: deniedReason != "",
		DeniedReason: deniedReason,
	}
}

type FormatDevice struct {
	device.Device
	Force bool `json:"force"`
}

func NewFormatDevice(d Device, force bool) FormatDevice {
	return FormatDevice{
		Device: d.Device,
		Force:  force,
	}
}
