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
	"fmt"
	"strings"

	"github.com/minio/direct-csi/pkg/matcher"
)

func ValidateAccessTier(at string) (AccessTier, error) {
	switch AccessTier(strings.Title(at)) {
	case AccessTierWarm:
		return AccessTierWarm, nil
	case AccessTierHot:
		return AccessTierHot, nil
	case AccessTierCold:
		return AccessTierCold, nil
	case AccessTierUnknown:
		return AccessTierUnknown, fmt.Errorf("Please set any one among ['hot','warm', 'cold']")
	default:
		return AccessTierUnknown, fmt.Errorf("Invalid 'access-tier' value, Please set any one among ['hot','warm','cold']")
	}
}

func GetAccessTierSet(accessTiers []string) ([]AccessTier, error) {
	var atSet []AccessTier
	for i := range accessTiers {
		if accessTiers[i] == "*" {
			return []AccessTier{
				AccessTierHot,
				AccessTierWarm,
				AccessTierCold,
			}, nil
		}
		at, err := ValidateAccessTier(strings.TrimSpace(accessTiers[i]))
		if err != nil {
			return atSet, err
		}
		atSet = append(atSet, at)
	}
	return atSet, nil
}

func AccessTiersToStrings(accessTiers []AccessTier) (slice []string) {
	for _, accessTier := range accessTiers {
		slice = append(slice, string(accessTier))
	}
	return slice
}

func (drive *DirectCSIDrive) MatchGlob(nodes, drives, status []string) bool {
	return matcher.GlobMatchNodesDrivesStatuses(nodes, drives, status, drive.Status.NodeName, drive.Status.Path, string(drive.Status.DriveStatus))
}

func (drive *DirectCSIDrive) MatchAccessTier(accessTierList []AccessTier) bool {
	return len(accessTierList) == 0 || matcher.StringIn(AccessTiersToStrings(accessTierList), string(drive.Status.AccessTier))
}
