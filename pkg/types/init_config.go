// This file is part of MinIO DirectPV
// Copyright (c) 2023 MinIO, Inc.
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
	"errors"
	"io"
	"strings"

	"github.com/google/uuid"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"gopkg.in/yaml.v3"
)

const (
	// DriveSelectedValue denotes the option in InitConfig
	DriveSelectedValue = "yes"
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

// ParseInitConfig reads and parses the init config
func ParseInitConfig(r io.Reader) (*InitConfig, error) {
	var config InitConfig
	if err := yaml.NewDecoder(r).Decode(&config); err != nil {
		return nil, err
	}
	if config.Version != latestInitConfigVersion {
		return nil, errUnsupportedInitConfigVersion
	}
	return &config, nil
}

// Write encodes the YAML to the stream provided
func (config InitConfig) Write(w io.Writer) error {
	encoder := yaml.NewEncoder(w)
	defer encoder.Close()
	return encoder.Encode(config)
}

// ToInitConfig converts the map to InitConfig
func ToInitConfig(resultMap map[directpvtypes.NodeID][]Device) InitConfig {
	nodeInfo := []NodeInfo{}
	initConfig := NewInitConfig()
	for node, devices := range resultMap {
		driveInfo := []DriveInfo{}
		for _, device := range devices {
			if device.DeniedReason != "" {
				continue
			}
			driveInfo = append(driveInfo, DriveInfo{
				ID:     device.ID,
				Name:   device.Name,
				Size:   device.Size,
				Make:   device.Make,
				FS:     device.FSType,
				Select: DriveSelectedValue,
			})
		}
		nodeInfo = append(nodeInfo, NodeInfo{
			Name:   node,
			Drives: driveInfo,
		})
	}
	initConfig.Nodes = nodeInfo
	return initConfig
}

// ToInitRequestObjects converts initConfig to init request objects.
//
// NOTE: After initrequest object creation, use the requestID for the cleanup
//
// Example :-
//
//	 ```
//		defer func() {
//			labelMap := map[directpvtypes.LabelKey][]directpvtypes.LabelValue{
//				directpvtypes.RequestIDLabelKey: toLabelValues([]string{requestID}),
//			}
//			client.InitRequestClient().DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
//				LabelSelector: directpvtypes.ToLabelSelector(labelMap),
//			})
//		}()
//	```
func (config *InitConfig) ToInitRequestObjects() (initRequests []InitRequest, requestID string) {
	requestID = uuid.New().String()
	for _, node := range config.Nodes {
		initDevices := []InitDevice{}
		for _, device := range node.Drives {
			if strings.ToLower(device.Select) != DriveSelectedValue {
				continue
			}
			initDevices = append(initDevices, InitDevice{
				ID:    device.ID,
				Name:  device.Name,
				Force: device.FS != "",
			})
		}
		if len(initDevices) > 0 {
			initRequests = append(initRequests, *NewInitRequest(requestID, node.Name, initDevices))
		}
	}
	return
}
