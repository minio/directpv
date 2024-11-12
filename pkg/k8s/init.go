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

package k8s

import (
	"sync/atomic"

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

var (
	initialized   int32
	defaultClient *Client
)

// Init initializes various client interfaces.
func Init() error {
	if atomic.AddInt32(&initialized, 1) != 1 {
		return nil
	}
	kubeConfig, err := GetKubeConfig()
	if err != nil {
		klog.Fatalf("unable to get kubernetes configuration; %v", err)
	}
	kubeConfig.WarningHandler = rest.NoWarnings{}
	defaultClient, err = NewClient(kubeConfig)
	if err != nil {
		klog.Fatalf("unable to create new kubernetes client interface; %v", err)
	}
	return nil
}

// GetClient returns the default global kubernetes client bundle.
// The default global kubernetes client is created by Init() and should be used internally.
// For common usage, create your own client by NewClient().
func GetClient() *Client {
	return defaultClient
}

// KubeConfig gets the default global kubernetes client configuration.
// Ths function should be used internally.
// For common usage, create your own client by NewClient().
func KubeConfig() *rest.Config {
	return GetClient().KubeConfig
}

// KubeClient gets the default global kubernetes client.
// Ths function should be used internally.
// For common usage, create your own client by NewClient().
func KubeClient() kubernetes.Interface {
	return GetClient().KubeClient
}

// CRDClient gets the default global kubernetes CRD client.
// Ths function should be used internally.
// For common usage, create your own client by NewClient().
func CRDClient() apiextensions.CustomResourceDefinitionInterface {
	return GetClient().CRDClient
}

// DiscoveryClient gets the default global kubernetes discovery client.
// Ths function should be used internally.
// For common usage, create your own client by NewClient().
func DiscoveryClient() discovery.DiscoveryInterface {
	return GetClient().DiscoveryClient
}
