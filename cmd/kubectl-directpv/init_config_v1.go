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

package main

import directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"

// InitConfigV1 defines the config to initialize the devices
type InitConfigV1 struct {
	Version string       `yaml:"version"`
	Nodes   []NodeInfoV1 `yaml:"nodes,omitempty"`
}

// NodeInfoV1 holds the node information
type NodeInfoV1 struct {
	Name   directpvtypes.NodeID `yaml:"name"`
	Drives []DriveInfoV1        `yaml:"drives,omitempty"`
}

// DriveInfoV1 represents the drives that are to be initialized
type DriveInfoV1 struct {
	ID         string `yaml:"id"`
	Name       string `yaml:"name"`
	MajorMinor string `yaml:"majorMinor"`
	FS         string `yaml:"fs,omitempty"`
}
