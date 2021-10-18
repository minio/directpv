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

func SupportedStatusSelectorValues() []string {
	return []string{
		string(DriveStatusInUse),
		string(DriveStatusAvailable),
		string(DriveStatusUnavailable),
		string(DriveStatusReady),
		string(DriveStatusTerminating),
		string(DriveStatusReleased),
	}
}

func ToAccessTier(value string) (accessTier AccessTier, err error) {
	accessTier = AccessTier(strings.Title(value))
	switch accessTier {
	case AccessTierWarm, AccessTierHot, AccessTierCold, AccessTierUnknown:
	default:
		err = fmt.Errorf("unknown access tier value %v", value)
	}
	return accessTier, err
}

func StringsToAccessTiers(values []string) (accessTiers []AccessTier, err error) {
	var accessTier AccessTier
	for _, value := range values {
		if accessTier, err = ToAccessTier(value); err != nil {
			return nil, err
		}
		accessTiers = append(accessTiers, accessTier)
	}
	return accessTiers, nil
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
