package identity

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	//"github.com/golang/glog"
)

func Run(ident, version string, manifest map[string]string) (csi.IdentityServer, error) {
	return &IdentityServer{
		Identity: ident,
		Version:  version,
		Manifest: manifest,
	}, nil
}

type IdentityServer struct {
	Identity string
	Version  string
	Manifest map[string]string
}

func (i *IdentityServer) GetPluginInfo(ctx context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
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

func (i *IdentityServer) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	return &csi.ProbeResponse{}, nil
}

func (i *IdentityServer) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	return nil, nil
}
