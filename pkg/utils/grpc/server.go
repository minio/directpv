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

package grpc

import (
	"context"
	"net"
	"net/url"
	"os"

	"google.golang.org/grpc"
	"k8s.io/klog/v2"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
)

// Run starts identity/controller/node servers.
func Run(ctx context.Context,
	endpoint string,
	identity csi.IdentityServer,
	controller csi.ControllerServer,
	node csi.NodeServer) error {
	
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return err
	}

	klog.V(5).Infof("listening on: %v", endpoint)
	if parsedURL.Scheme == "unix" {
		if err := os.Remove(parsedURL.RequestURI()); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		}
	}

	lc := &net.ListenConfig{}
	listener, err := lc.Listen(ctx, parsedURL.Scheme, parsedURL.RequestURI())
	if err != nil {
		return err
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(logGRPC),
	}
	server := grpc.NewServer(opts...)

	go func() {
		<-ctx.Done()
		server.GracefulStop()
		os.Remove(parsedURL.Host)
	}()

	if identity != nil {
		csi.RegisterIdentityServer(server, identity)
	}
	if controller != nil {
		csi.RegisterControllerServer(server, controller)
	}
	if node != nil {
		csi.RegisterNodeServer(server, node)
	}

	return server.Serve(listener)
}

func logGRPC(ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (interface{}, error) {
	
	klog.V(5).Infof("GRPC call: %s", info.FullMethod)
	klog.V(5).Infof("GRPC request: %s", protosanitizer.StripSecrets(req))
	resp, err := handler(ctx, req)
	if err != nil {
		klog.Errorf("GRPC error: %v", err)
	} else {
		klog.V(5).Infof("GRPC response: %s", protosanitizer.StripSecrets(resp))
	}
	return resp, err
}
