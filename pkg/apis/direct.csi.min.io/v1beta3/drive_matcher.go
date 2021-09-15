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

func (drive *DirectCSIDrive) MatchGlob(nodes, drives, status []string) bool {
	return matcher.GlobMatchNodesDrivesStatuses(nodes, drives, status, drive.Status.NodeName, drive.Status.Path, string(drive.Status.DriveStatus))
}

func (drive *DirectCSIDrive) MatchAccessTier(accessTierList []AccessTier) bool {
	return matcher.StringIn(accessTiersToStrings(accessTierList), string(drive.Status.AccessTier))
}
