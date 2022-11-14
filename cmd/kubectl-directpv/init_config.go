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

import (
	"errors"
	"io"

	"gopkg.in/yaml.v2"
)

var errUnsupportedInitConfigVersion = errors.New("unsupported init config version")

const latestInitConfigVersion = "v1"

// InitConfig holds the latest config version
type InitConfig = InitConfigV1

// NodeInfo holds the latest node info
type NodeInfo = NodeInfoV1

// DriveInfo holds the latest drive info
type DriveInfo = DriveInfoV1

// NewInitConfig initializes an init config.
func NewInitConfig() InitConfig {
	return InitConfig{
		Version: latestInitConfigVersion,
	}
}

func parseInitConfig(r io.Reader) (*InitConfig, error) {
	var config InitConfig
	if err := yaml.NewDecoder(r).Decode(&config); err != nil {
		return nil, err
	}
	if config.Version != latestInitConfigVersion {
		return nil, errUnsupportedInitConfigVersion
	}
	return &config, nil
}

func (config InitConfig) Write(w io.Writer) error {
	encoder := yaml.NewEncoder(w)
	defer encoder.Close()
	return encoder.Encode(config)
}
