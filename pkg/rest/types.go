// This file is part of MinIO DirectPV
// Copyright (c) 2021, 2022 MinIO, Inc.
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

package rest

type NodeName string

type Selector string

// DeviceStatusAccessTier denotes device status.
type DeviceStatus string

const (
	// DeviceStatusOnline denotes that the device is managed by DirectPV
	DeviceStatusOnline DeviceStatus = "Online"

	// DeviceStatusOffline denotes that the Online DirectPV drive is corrupted or lost
	DeviceStatusOffline DeviceStatus = "Offline"

	// DeviceStatusAvailable denotes that the device is available for formatting
	DeviceStatusAvailable DeviceStatus = "Available"

	// DeviceStatusUnavailable denotes that the device is NOT available for formatting
	DeviceStatusUnavailable DeviceStatus = "Unavailable"
)

// GetDevicesRequest is the request type to fetch the devices present in the cluster
type GetDevicesRequest struct {
	Nodes    Selector       `json:"nodes,omitempty"`
	Drives   Selector       `json:"drives,omitempty"`
	Statuses []DeviceStatus `json:"statuses,omitempty"`
}

// GetDevicesResponse is the response type to represent the devices from the corresponding node
type GetDevicesResponse struct {
	DeviceInfo map[NodeName][]Device `json:"deviceInfo"`
}

// Device holds Disk information
type Device struct {
	Name        string       `json:"name"`
	Major       int          `json:"major"`
	Minor       int          `json:"minor"`
	Size        int64        `json:"size,omitempty"`
	Model       string       `json:"model,omitempty"`
	Vendor      string       `json:"vendor,omitmepty"`
	Filesystem  string       `json:"filesystem,omitempty"`
	Mountpoints []string     `json:"mountpoints,omitempty"`
	Status      DeviceStatus `json:"status"`
	// UDevData holds the device metadata info probed from `/run/udev/data/b<maj><min>`
	UDevData map[string]string `json:"udevData,omitempty"`
}

// FormatDevicesRequest is the request type to represent the format request
type FormatDevicesRequest struct {
	FormatInfo map[NodeName][]FormatDevice `json:"formatInfo"`
}

// FormatDevice represents the devices requested to be formatted
type FormatDevice struct {
	Name  string `json:"name"`
	Major int    `json:"major"`
	Minor int    `json:"minor"`
	Force bool   `json:"force,omitempty"`
	// UDevData holds the device metadata sent in the fetch drives response
	UDevData map[string]string `json:"udevData"`
}

// FormatDevicesResponse represents the format status of the devices requested for formatting
type FormatDevicesResponse struct {
	DeviceInfo map[NodeName][]FormatDeviceStatus `json:"deviceInfo"`
}

// FormatDeviceStatus represents the status of the device requested for formatting
type FormatDeviceStatus struct {
	Name       string `json:"name"`
	FSUUID     string `json"fsuuid,omitempty`
	Error      string `json:"error,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

// FormatMetadata represents the format metadata to be saved on the drive
type FormatMetadata struct {
	FSUUID      string `json:"fsuuid"`
	FormattedBy string `json:formattedBy`
}
