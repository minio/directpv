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

package client

import (
	"github.com/minio/directpv/pkg/types"
	"k8s.io/client-go/rest"
)

var (
	initialized int32
	client      *Client
)

// GetClient returns the client
func GetClient() *Client {
	return client
}

// RESTClient gets latest versioned REST client.
func RESTClient() rest.Interface {
	return client.REST()
}

// DriveClient gets latest versioned drive interface.
func DriveClient() types.LatestDriveInterface {
	return client.Drive()
}

// VolumeClient gets latest versioned volume interface.
func VolumeClient() types.LatestVolumeInterface {
	return client.Volume()
}

// NodeClient gets latest versioned node interface.
func NodeClient() types.LatestNodeInterface {
	return client.Node()
}

// InitRequestClient gets latest versioned init request interface.
func InitRequestClient() types.LatestInitRequestInterface {
	return client.InitRequest()
}

// NewDriveLister returns the new drive lister
func NewDriveLister() *DriveLister {
	return client.NewDriveLister()
}

// NewVolumeLister returns the new volume lister
func NewVolumeLister() *VolumeLister {
	return client.NewVolumeLister()
}

// NewNodeLister returns the new node lister
func NewNodeLister() *NodeLister {
	return client.NewNodeLister()
}

// NewInitRequestLister returns the new initrequest lister
func NewInitRequestLister() *InitRequestLister {
	return client.NewInitRequestLister()
}
