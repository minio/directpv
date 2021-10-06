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

package v1beta3

import (
	"github.com/minio/direct-csi/pkg/matcher"
)

func accessTiersToStrings(accessTiers []AccessTier) (slice []string) {
	for _, accessTier := range accessTiers {
		slice = append(slice, string(accessTier))
	}
	return slice
}

func (drive *DirectCSIDrive) MatchGlob(nodes, drives []string) bool {
	return matcher.GlobMatchNodesDrives(nodes, drives, drive.Status.NodeName, drive.Status.Path)
}

func (drive *DirectCSIDrive) MatchAccessTier(accessTierList []AccessTier) bool {
	return len(accessTierList) == 0 || matcher.StringIn(accessTiersToStrings(accessTierList), string(drive.Status.AccessTier))
}

func (drive *DirectCSIDrive) MatchDriveStatus(statusList []DriveStatus) bool {
	if len(statusList) == 0 {
		return true
	}
	for _, st := range statusList {
		if drive.Status.DriveStatus == st {
			return true
		}
	}
	return false
}
