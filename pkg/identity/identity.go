// This file is part of MinIO Direct CSI
// Copyright (c) 2021 MinIO, Inc.
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

package identity

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

// NewIdentityServer creates new identity server.
func NewIdentityServer(ident, version string, manifest map[string]string) (csi.IdentityServer, error) {
	return &identityServer{
		Identity: ident,
		Version:  version,
		Manifest: manifest,
	}, nil
}

type identityServer struct {
	Identity string
	Version  string
	Manifest map[string]string
}

func (i *identityServer) GetPluginInfo(ctx context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	if i.Identity == "" {
		return nil, status.Error(codes.Unavailable, "Driver name not configured")
	}

	if i.Version == "" {
		return nil, status.Error(codes.Unavailable, "Driver is missing version")
	}

	return &csi.GetPluginInfoResponse{
		Name:          i.Identity,
		VendorVersion: i.Version,
	}, nil
}

func (i *identityServer) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	return &csi.ProbeResponse{}, nil
}

func (i *identityServer) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	serviceCap := func(cap csi.PluginCapability_Service_Type) *csi.PluginCapability {
		klog.V(5).Infof("Using plugin capability %v", cap)

		return &csi.PluginCapability{
			Type: &csi.PluginCapability_Service_{
				Service: &csi.PluginCapability_Service{
					Type: cap,
				},
			},
		}
	}

	caps := []*csi.PluginCapability{
		serviceCap(csi.PluginCapability_Service_CONTROLLER_SERVICE),
		serviceCap(csi.PluginCapability_Service_VOLUME_ACCESSIBILITY_CONSTRAINTS),
	}

	return &csi.GetPluginCapabilitiesResponse{
		Capabilities: caps,
	}, nil
}
