// This file is part of MinIO
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

package types

import (
	"fmt"
	"strings"
)

// DriveStatus denotes drive status
type DriveStatus string

const (
	// DriveStatusOK denotes drive is ready for volume schedule.
	DriveStatusOK DriveStatus = "OK"

	// DriveStatusError denotes drive is in error state and no volumes will be scheduled on it anymore.
	DriveStatusError DriveStatus = "Error"

	// DriveStatusDeleted denotes drive is deleted.
	DriveStatusDeleted DriveStatus = "Deleted"

	// DriveStatusFenced denotes drive is fenced and no volumes will be scheduled on it anymore.
	DriveStatusFenced DriveStatus = "Fenced"

	// DriveStatusError denotes drive is lost and no volumes will be scheduled on it anymore.
	DriveStatusLost DriveStatus = "Lost"

	// DriveStatusReleased denotes drive is released.
	DriveStatusReleased DriveStatus = "Released"
)

// AccessTier denotes access tier.
type AccessTier string

const (
	// AccessTierWarm denotes "Warm" access tier.
	AccessTierWarm AccessTier = "Warm"

	// AccessTierHot denotes "Hot" access tier.
	AccessTierHot AccessTier = "Hot"

	// AccessTierCold denotes "Cold" access tier.
	AccessTierCold AccessTier = "Cold"

	// AccessTierUnknown denotes "Unknown" access tier.
	AccessTierUnknown AccessTier = "Unknown"
)

// SupportedAccessTierValues returns the supported access tier values for filtering and setting
func SupportedAccessTierValues() []string {
	return []string{
		string(AccessTierHot),
		string(AccessTierWarm),
		string(AccessTierCold),
	}
}

// ToAccessTier converts a string to AccessTier
func ToAccessTier(value string) (AccessTier, error) {
	switch at := AccessTier(strings.Title(value)); at {
	case AccessTierWarm, AccessTierHot, AccessTierCold, AccessTierUnknown:
		return at, nil
	default:
		return at, fmt.Errorf("unknown access tier value %v", value)
	}
}

// StringsToAccessTiers converts strings to access tiers.
func StringsToAccessTiers(values ...string) (accessTiers []AccessTier, err error) {
	for _, value := range values {
		aT, err := ToAccessTier(value)
		if err != nil {
			return nil, err
		}
		accessTiers = append(accessTiers, aT)
	}
	return accessTiers, nil
}

// AccessTiersToStrings converts slice of access tiers to its string values
func AccessTiersToStrings(accessTiers ...AccessTier) (slice []string) {
	for _, accessTier := range accessTiers {
		slice = append(slice, string(accessTier))
	}
	return slice
}

// VolumeConditionType denotes volume condition.
type VolumeConditionType string

// Volume condition type values
const (
	VolumeConditionTypeLost VolumeConditionType = "Lost"
)

// VolumeConditionReason denotes volume reason.
type VolumeConditionReason string

// Volume condition reason values
const (
	VolumeConditionReasonDriveLost VolumeConditionReason = "DriveLost"
)

// VolumeConditionMessage denotes drive message.
type VolumeConditionMessage string

// Volume condition message values
const (
	VolumeConditionMessageDriveLost VolumeConditionMessage = "Associated drive was removed. Refer https://github.com/minio/directpv/blob/master/docs/troubleshooting.md"
)
