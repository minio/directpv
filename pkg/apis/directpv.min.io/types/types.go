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

// DriveName is drive name type.
type DriveName string

// NodeID is node ID type.
type NodeID string

// DriveID is drive ID type.
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

	// DriveStatusRemoved denotes drive is removed.
	DriveStatusRemoved DriveStatus = "Removed"

	// DriveStatusMoving denotes drive is moving volumes.
	DriveStatusMoving DriveStatus = "Moving"
)

// ToDriveStatus converts string value to DriveStatus.
func ToDriveStatus(value string) (status DriveStatus, err error) {
	status = DriveStatus(strings.Title(value))
	switch status {
	case DriveStatusReady, DriveStatusLost, DriveStatusError, DriveStatusRemoved, DriveStatusMoving:
		return status, nil
	}

	err = fmt.Errorf("unknown drive status %v", value)
	return
}

// VolumeStatus represents status of a volume.
type VolumeStatus string

// Enum of VolumeStatus type.
const (
	VolumeStatusPending VolumeStatus = "Pending"
	VolumeStatusReady   VolumeStatus = "Ready"
)

// ToVolumeStatus converts string value to VolumeStatus.
func ToVolumeStatus(value string) (status VolumeStatus, err error) {
	status = VolumeStatus(strings.Title(value))
	switch status {
	case VolumeStatusReady, VolumeStatusPending:
		return status, nil
	}

	err = fmt.Errorf("unknown volume status %v", value)
	return
}

// AccessTier denotes access tier.
type AccessTier string

// Enum values of AccessTier type.
const (
	AccessTierDefault AccessTier = "Default"
	AccessTierWarm    AccessTier = "Warm"
	AccessTierHot     AccessTier = "Hot"
	AccessTierCold    AccessTier = "Cold"
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

// VolumeConditionType denotes volume condition. Allows maximum upto 316 chars.
type VolumeConditionType string

// Enum value of VolumeConditionType type.
const (
	VolumeConditionTypeLost VolumeConditionType = "Lost"
)

// VolumeConditionReason denotes volume reason. Allows maximum upto 1024 chars.
type VolumeConditionReason string

// Enum values of VolumeConditionReason type.
const (
	VolumeConditionReasonDriveLost VolumeConditionReason = "DriveLost"
)

// VolumeConditionMessage denotes drive message. Allows maximum upto 32768 chars.
type VolumeConditionMessage string

// Enum values of VolumeConditionMessage type.
const (
	VolumeConditionMessageDriveLost VolumeConditionMessage = "Associated drive was removed. Refer https://github.com/minio/directpv/blob/master/docs/troubleshooting.md"
)

// DriveConditionType denotes drive condition. Allows maximum upto 316 chars.
type DriveConditionType string

// Enum values of DriveConditionType type.
const (
	DriveConditionTypeMountError      DriveConditionType = "MountError"
	DriveConditionTypeMultipleMatches DriveConditionType = "MultipleMatches"
	DriveConditionTypeIOError         DriveConditionType = "IOError"
	DriveConditionTypeRelabelError    DriveConditionType = "RelabelError"
)

// DriveConditionReason denotes the reason for the drive condition type. Allows maximum upto 1024 chars.
type DriveConditionReason string

// Enum values of DriveConditionReason type.
const (
	DriveConditionReasonMountError      DriveConditionReason = "DriveHasMountError"
	DriveConditionReasonMultipleMatches DriveConditionReason = "DriveHasMultipleMatches"
	DriveConditionReasonIOError         DriveConditionReason = "DriveHasIOError"
	DriveConditionReasonRelabelError    DriveConditionReason = "DriveHasRelabelError"
)

// DriveConditionMessage denotes drive message. Allows maximum upto 32768 chars
type DriveConditionMessage string

// Enum values of DriveConditionMessage type.
const (
	DriveConditionMessageIOError DriveConditionMessage = "Drive has I/O error"
)

// InitStatus denotes initialization status
type InitStatus string

const (
	// InitStatusPending denotes that the initialization request is still pending.
	InitStatusPending InitStatus = "Pending"
	// InitStatusProcessed denotes that the initialization request has been processed.
	InitStatusProcessed InitStatus = "Processed"
	// InitStatusError denotes that the initialization request has failed due to an error.
	InitStatusError InitStatus = "Error"
)
