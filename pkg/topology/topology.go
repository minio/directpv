// This file is part of MinIO Kubernetes Cloud
// Copyright (c) 2020 MinIO, Inc.
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

package topology

const (
	TopologyDriverIdentity = "direct.csi.driver.min.io/identity"
	TopologyDriverNode     = "direct.csi.driver.min.io/node"
	TopologyDriverRack     = "direct.csi.driver.min.io/rack"
	TopologyDriverZone     = "direct.csi.driver.min.io/zone"
	TopologyDriverRegion   = "direct.csi.driver.min.io/region"
)

type TopologyConstraint struct {
	DriverIdentity string `json:"driverIdentity,omitempty"`
	DriverNode     string `json:"driverNode,omitempty"`
	DriverRack     string `json:"driverRack,omitempty"`
	DriverZone     string `json:"driverZone,omitempty"`
	DriverRegion   string `json:"driverRegion,omitempty"`
}

func (in *TopologyConstraint) DeepCopyInto(out *TopologyConstraint) {
	if out == nil {
		out = new(TopologyConstraint)
	}
	out.DriverIdentity = in.DriverIdentity
	out.DriverNode = in.DriverNode
	out.DriverRack = in.DriverRack
	out.DriverZone = in.DriverZone
	out.DriverRegion = in.DriverRegion
}
