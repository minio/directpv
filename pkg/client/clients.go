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
	initialized        int32
	clientsetInterface types.ExtClientsetInterface
	restClient         rest.Interface
	driveClient        types.LatestDriveInterface
	volumeClient       types.LatestVolumeInterface
	nodeClient         types.LatestNodeInterface
	initRequestClient  types.LatestInitRequestInterface
)

// RESTClient gets latest versioned REST client.
func RESTClient() rest.Interface {
	return restClient
}

// DriveClient gets latest versioned drive interface.
func DriveClient() types.LatestDriveInterface {
	return driveClient
}

// VolumeClient gets latest versioned volume interface.
func VolumeClient() types.LatestVolumeInterface {
	return volumeClient
}

// NodeClient gets latest versioned node interface.
func NodeClient() types.LatestNodeInterface {
	return nodeClient
}

// InitRequestClient gets latest versioned init request interface.
func InitRequestClient() types.LatestInitRequestInterface {
	return initRequestClient
}
