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

package main

import (
	"strings"

	"github.com/fatih/color"

	directv1beta1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta1"
	"github.com/minio/direct-csi/pkg/utils"
)

// pretty printing utils
const dot = "•"

var (
	bold   = color.New(color.Bold).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
)

// ListVolumesInDrive returns a slice of all the DirectCSIVolumes created on a given DirectCSIDrive
func ListVolumesInDrive(drive directv1beta1.DirectCSIDrive, volumes *directv1beta1.DirectCSIVolumeList, vols []directv1beta1.DirectCSIVolume) []directv1beta1.DirectCSIVolume {
	for _, volume := range volumes.Items {
		if volume.Status.Drive == drive.ObjectMeta.Name {
			vols = append(vols, volume)
		}
	}
	return vols
}

func getAccessTierSet(accessTiers []string) ([]directv1beta1.AccessTier, error) {
	var atSet []directv1beta1.AccessTier
	for i := range accessTiers {
		if accessTiers[i] == "*" {
			return []directv1beta1.AccessTier{directv1beta1.AccessTierHot,
				directv1beta1.AccessTierWarm,
				directv1beta1.AccessTierCold}, nil
		}
		at, err := utils.ValidateAccessTier(strings.TrimSpace(accessTiers[i]))
		if err != nil {
			return atSet, err
		}
		atSet = append(atSet, at)
	}
	return atSet, nil
}

func printableString(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
