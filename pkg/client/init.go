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
	"fmt"
	"sync/atomic"

	"github.com/minio/directpv/pkg/clientset"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	// support gcp, azure, and oidc client auth
	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

// Init initializes various clients.
func Init() {
	if atomic.AddInt32(&initialized, 1) != 1 {
		return
	}
	var err error
	if err = k8s.Init(); err != nil {
		klog.Fatalf("unable to initialize k8s clients; %v", err)
	}
	client, err = newClient(k8s.GetClient())
	if err != nil {
		klog.Fatalf("unable to initialize client; %v", err)
	}
	initEvent(k8s.KubeClient())
}

// Client represents the directpv client set
type Client struct {
	ClientsetInterface types.ExtClientsetInterface
	RESTClient         rest.Interface
	DriveClient        types.LatestDriveInterface
	VolumeClient       types.LatestVolumeInterface
	NodeClient         types.LatestNodeInterface
	InitRequestClient  types.LatestInitRequestInterface
	K8sClient          *k8s.Client
}

// REST returns the REST client
func (c Client) REST() rest.Interface {
	return c.RESTClient
}

// Drive returns the DirectPV Drive interface
func (c Client) Drive() types.LatestDriveInterface {
	return c.DriveClient
}

// Volume returns the DirectPV Volume interface
func (c Client) Volume() types.LatestVolumeInterface {
	return c.VolumeClient
}

// Node returns the DirectPV Node interface
func (c Client) Node() types.LatestNodeInterface {
	return c.NodeClient
}

// InitRequest returns the DirectPV InitRequest interface
func (c Client) InitRequest() types.LatestInitRequestInterface {
	return c.InitRequestClient
}

// K8s returns the kubernetes client
func (c Client) K8s() *k8s.Client {
	return c.K8sClient
}

// KubeConfig returns the kubeconfig
func (c Client) KubeConfig() *rest.Config {
	return c.K8sClient.KubeConfig
}

// Kube returns the kube client
func (c Client) Kube() kubernetes.Interface {
	return c.K8sClient.KubeClient
}

// APIextensions returns the APIextensionsClient
func (c Client) APIextensions() apiextensions.ApiextensionsV1Interface {
	return c.K8sClient.APIextensionsClient
}

// CRD returns the CRD client
func (c Client) CRD() apiextensions.CustomResourceDefinitionInterface {
	return c.K8sClient.CRDClient
}

// Discovery returns the discovery client
func (c Client) Discovery() discovery.DiscoveryInterface {
	return c.K8sClient.DiscoveryClient
}

// NewClient returns the directpv client
func NewClient(c *rest.Config) (*Client, error) {
	k8sClient, err := k8s.NewClient(c)
	if err != nil {
		return nil, fmt.Errorf("unable to create kubernetes client; %w", err)
	}
	return newClient(k8sClient)
}

// newClient returns the directpv client
func newClient(k8sClient *k8s.Client) (*Client, error) {
	cs, err := clientset.NewForConfig(k8sClient.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create new clientset interface; %w", err)
	}
	clientsetInterface := types.NewExtClientset(cs)
	restClient := clientsetInterface.DirectpvLatest().RESTClient()
	driveClient, err := latestDriveClientForConfig(k8sClient)
	if err != nil {
		return nil, fmt.Errorf("unable to create new drive interface; %w", err)
	}
	volumeClient, err := latestVolumeClientForConfig(k8sClient)
	if err != nil {
		return nil, fmt.Errorf("unable to create new volume interface; %w", err)
	}
	nodeClient, err := latestNodeClientForConfig(k8sClient)
	if err != nil {
		return nil, fmt.Errorf("unable to create new node interface; %w", err)
	}
	initRequestClient, err := latestInitRequestClientForConfig(k8sClient)
	if err != nil {
		return nil, fmt.Errorf("unable to create new initrequest interface; %w", err)
	}
	return &Client{
		ClientsetInterface: clientsetInterface,
		RESTClient:         restClient,
		DriveClient:        driveClient,
		VolumeClient:       volumeClient,
		NodeClient:         nodeClient,
		InitRequestClient:  initRequestClient,
		K8sClient:          k8sClient,
	}, nil
}
