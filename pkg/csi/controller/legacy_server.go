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

package controller

import (
	"context"
	"fmt"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/minio/directpv/pkg/consts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LegacyServer denotes legacy controller server.
type LegacyServer struct {
	Server
}

// NewLegacyServer creates new legacy controller server.
func NewLegacyServer() *LegacyServer {
	return &LegacyServer{}
}

// CreateVolume - Creates a volume
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#createvolume
func (c *LegacyServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	return nil, status.Errorf(
		codes.InvalidArgument,
		fmt.Sprintf("legacy volume creation not supported; use %v storage class", consts.Identity),
	)
}
