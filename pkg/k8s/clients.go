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
	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// MaxThreadCount is maximum thread count.
const MaxThreadCount = 200

var (
	initialized int32
	client      *Client
)

// GetClient returns kubernetes client.
func GetClient() *Client {
	return client
}

// KubeConfig gets kubernetes client configuration.
func KubeConfig() *rest.Config {
	return client.KubeConfig
}

// KubeClient gets kubernetes client.
func KubeClient() kubernetes.Interface {
	return client.KubeClient
}

// CRDClient gets kubernetes CRD client.
func CRDClient() apiextensions.CustomResourceDefinitionInterface {
	return client.CRDClient
}

// DiscoveryClient gets kubernetes discovery client.
func DiscoveryClient() discovery.DiscoveryInterface {
	return client.DiscoveryClient
}
