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

	// DriveStatusError denotes drive is in error state to prevent volume schedule.
	DriveStatusError DriveStatus = "Error"

	// DriveStatusDeleted denotes drive is deleted.
	DriveStatusDeleted DriveStatus = "Deleted"

	// DriveStatusFenced denotes drive is fenced to prevent volume schedule.
	DriveStatusFenced DriveStatus = "Fenced"
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

// StringsToAccessTiers converts strings to access tiers.
func StringsToAccessTiers(values ...string) (accessTiers []AccessTier, err error) {
	for _, value := range values {
		switch at := AccessTier(strings.Title(value)); at {
		case AccessTierWarm, AccessTierHot, AccessTierCold, AccessTierUnknown:
			accessTiers = append(accessTiers, at)
		default:
			return nil, fmt.Errorf("unknown access tier value %v", value)
		}
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

const (
	// VolumeConditionTypePublished denotes "Published" volume condition.
	VolumeConditionTypePublished VolumeConditionType = "Published"

	// VolumeConditionTypeStaged denotes "Staged" volume condition.
	VolumeConditionTypeStaged VolumeConditionType = "Staged"

	// VolumeConditionTypeReady denotes "Ready" volume condition.
	VolumeConditionTypeReady VolumeConditionType = "Ready"
)

// VolumeConditionReason denotes volume reason.
type VolumeConditionReason string

const (
	// VolumeConditionReasonNotInUse denotes "NotInUse" volume reason.
	VolumeConditionReasonNotInUse VolumeConditionReason = "NotInUse"

	// VolumeConditionReasonInUse denotes "InUse" volume reason.
	VolumeConditionReasonInUse VolumeConditionReason = "InUse"

	// VolumeConditionReasonReady denotes "Ready" volume reason.
	VolumeConditionReasonReady VolumeConditionReason = "Ready"

	// VolumeConditionReasonNotReady denotes "NotReady" volume reason.
	VolumeConditionReasonNotReady VolumeConditionReason = "NotReady"

	// VolumeConditionReasonDriveLost denotes "DriveLost" volume reason.
	VolumeConditionReasonDriveLost VolumeConditionReason = "DriveLost"
)

// VolumeConditionMessage denotes drive message.
type VolumeConditionMessage string

const (
	// VolumeConditionMessageDriveLost denotes "DriveLost" drive message.
	VolumeConditionMessageDriveLost VolumeConditionMessage = "Drive Lost"
)
