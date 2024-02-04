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
	"fmt"
	"strconv"
	"strings"
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
	client, err = NewClient(kubeConfig)
	if err != nil {
		klog.Fatalf("unable to create new kubernetes client interface; %v", err)
	}
	return nil
}

// Client represents the kubernetes client set.
type Client struct {
	KubeConfig          *rest.Config
	KubeClient          kubernetes.Interface
	APIextensionsClient apiextensions.ApiextensionsV1Interface
	CRDClient           apiextensions.CustomResourceDefinitionInterface
	DiscoveryClient     discovery.DiscoveryInterface
}

// GetKubeVersion returns the k8s version info
func (client Client) GetKubeVersion() (major, minor uint, err error) {
	versionInfo, err := client.DiscoveryClient.ServerVersion()
	if err != nil {
		return 0, 0, err
	}

	var u64 uint64
	if u64, err = strconv.ParseUint(versionInfo.Major, 10, 64); err != nil {
		return 0, 0, fmt.Errorf("unable to parse major version %v; %v", versionInfo.Major, err)
	}
	major = uint(u64)

	minorString := versionInfo.Minor
	if strings.Contains(versionInfo.GitVersion, "-eks-") {
		// Do trimming only for EKS.
		// Refer https://github.com/aws/containers-roadmap/issues/1404
		i := strings.IndexFunc(minorString, func(r rune) bool { return r < '0' || r > '9' })
		if i > -1 {
			minorString = minorString[:i]
		}
	}
	if u64, err = strconv.ParseUint(minorString, 10, 64); err != nil {
		return 0, 0, fmt.Errorf("unable to parse minor version %v; %v", minor, err)
	}
	minor = uint(u64)
	return major, minor, nil
}

// NewClient initializes the client with the provided kube config.
func NewClient(kubeConfig *rest.Config) (*Client, error) {
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create new kubernetes client interface; %v", err)
	}
	apiextensionsClient, err := apiextensions.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create new API extensions client interface; %v", err)
	}
	crdClient := apiextensionsClient.CustomResourceDefinitions()
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create new discovery client interface; %v", err)
	}
	return &Client{
		KubeConfig:          kubeConfig,
		KubeClient:          kubeClient,
		APIextensionsClient: apiextensionsClient,
		CRDClient:           crdClient,
		DiscoveryClient:     discoveryClient,
	}, nil
}
