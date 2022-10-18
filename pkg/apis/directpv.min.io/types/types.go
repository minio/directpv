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

type DriveName string

type NodeID string

type DriveID string

// DriveStatus denotes drive status
type DriveStatus string

const (
	// DriveStatusReady denotes drive is ready for volume schedule.
	DriveStatusReady DriveStatus = "Ready"

	// DriveStatusLost denotes associated data by FSUUID is lost.
	DriveStatusLost DriveStatus = "Lost"

	// DriveStatusError denotes drive is in error state to prevent volume schedule.
	DriveStatusError DriveStatus = "Error"

	// DriveStatusReleased denotes drive is removed.
	DriveStatusReleased DriveStatus = "Released"

	// DriveStatusMoving denotes drive is moving volumes.
	DriveStatusMoving DriveStatus = "Moving"
)

func ToDriveStatus(value string) (status DriveStatus, err error) {
	status = DriveStatus(strings.Title(value))
	switch status {
	case DriveStatusReady, DriveStatusLost, DriveStatusError, DriveStatusReleased, DriveStatusMoving:
		return status, nil
	}

	err = fmt.Errorf("unknown drive status %v", value)
	return
}

type VolumeStatus string

const (
	VolumeStatusPending VolumeStatus = "Pending"
	VolumeStatusReady   VolumeStatus = "Ready"
)

// AccessTier denotes access tier.
type AccessTier string

const (
	// AccessTierDefault denotes "Default" access tier.
	AccessTierDefault AccessTier = "Default"

	// AccessTierWarm denotes "Warm" access tier.
	AccessTierWarm AccessTier = "Warm"

	// AccessTierHot denotes "Hot" access tier.
	AccessTierHot AccessTier = "Hot"

	// AccessTierCold denotes "Cold" access tier.
	AccessTierCold AccessTier = "Cold"
)

// StringsToAccessTiers converts strings to access tiers.
func StringsToAccessTiers(values ...string) (accessTiers []AccessTier, err error) {
	for _, value := range values {
		switch at := AccessTier(strings.Title(value)); at {
		case AccessTierDefault, AccessTierWarm, AccessTierHot, AccessTierCold:
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
