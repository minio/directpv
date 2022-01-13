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

package client

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"text/template"

	direct "github.com/minio/directpv/pkg/clientset"
	directcsi "github.com/minio/directpv/pkg/clientset/typed/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/utils"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"

	"k8s.io/klog/v2"
)

// MaxThreadCount is maximum thread count.
const MaxThreadCount = 200

var (
	initialized                    int32
	kubeClient                     kubernetes.Interface
	directCSIClient                directcsi.DirectV1beta3Interface
	directClientset                direct.Interface
	apiextensionsClient            apiextensions.ApiextensionsV1Interface
	crdClient                      apiextensions.CustomResourceDefinitionInterface
	discoveryClient                discovery.DiscoveryInterface
	metadataClient                 metadata.Interface
	latestDirectCSIDriveInterface  directcsi.DirectCSIDriveInterface
	latestDirectCSIVolumeInterface directcsi.DirectCSIVolumeInterface

	binaryName = func() string {
		base := filepath.Base(os.Args[0])
		return strings.ReplaceAll(strings.ReplaceAll(base, "kubectl-", ""), "_", "-")
	}
	binaryNameTransform = func(text string) string {
		transformed := &strings.Builder{}
		if err := template.Must(template.
			New("").Parse(text)).Execute(transformed, binaryName()); err != nil {
			panic(err)
		}
		return transformed.String()
	}
)

// GetLatestDirectCSIDriveInterface gets latest versioned direct-csi drive interface.
func GetLatestDirectCSIDriveInterface() directcsi.DirectCSIDriveInterface {
	return latestDirectCSIDriveInterface
}

// GetLatestDirectCSIVolumeInterface gets latest versioned direct-csi volume interface.
func GetLatestDirectCSIVolumeInterface() directcsi.DirectCSIVolumeInterface {
	return latestDirectCSIVolumeInterface
}

// GetKubeClient gets kube client.
func GetKubeClient() kubernetes.Interface {
	return kubeClient
}

// GetDirectCSIClient gets direct-csi client.
func GetDirectCSIClient() directcsi.DirectV1beta3Interface {
	return directCSIClient
}

// GetDirectClientset gets direct-csi clientset.
func GetDirectClientset() direct.Interface {
	return directClientset
}

// GetAPIExtensionsClient gets API extension client.
func GetAPIExtensionsClient() apiextensions.ApiextensionsV1Interface {
	return apiextensionsClient
}

// GetCRDClient gets CRD client.
func GetCRDClient() apiextensions.CustomResourceDefinitionInterface {
	return crdClient
}

// GetDiscoveryClient gets discovery client.
func GetDiscoveryClient() discovery.DiscoveryInterface {
	return discoveryClient
}

// GetMetadataClient gets metadata client.
func GetMetadataClient() metadata.Interface {
	return metadataClient
}

// Init initializes various clients.
func Init() {
	if atomic.AddInt32(&initialized, 1) != 1 {
		return
	}

	config, kubeConfig, err := getKubeConfig()
	if err != nil {
		fmt.Printf("%s: Could not connect to kubernetes. %s=%s\n", utils.Bold("Error"), "KUBECONFIG", kubeConfig)
		os.Exit(1)
	}
	klog.V(1).Infof("obtained client config successfully")

	kubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("%s: could not initialize kube client: err=%v\n", utils.Bold("Error"), err)
		os.Exit(1)
	}

	directClientset, err = direct.NewForConfig(config)
	if err != nil {
		fmt.Printf(binaryNameTransform("%s: could not initialize {{ . }} client: err=%v\n"), utils.Bold("Error"), err)
		os.Exit(1)
	}

	directCSIClient, err = directcsi.NewForConfig(config)
	if err != nil {
		klog.Fatalf(binaryNameTransform("could not initialize {{ . }} client: %v"), err)
	}

	crdClientset, err := apiextensions.NewForConfig(config)
	if err != nil {
		fmt.Printf("%s: could not initialize crd client: err=%v\n", utils.Bold("Error"), err)
		os.Exit(1)
	}
	apiextensionsClient = crdClientset
	crdClient = crdClientset.CustomResourceDefinitions()

	discoveryClient, err = discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		fmt.Printf("%s: could not initialize discovery client: err=%v\n", utils.Bold("Error"), err)
		os.Exit(1)
	}

	metadataClient, err = metadata.NewForConfig(config)
	if err != nil {
		fmt.Printf("%s: could not initialize metadata client: err=%v\n", utils.Bold("Error"), err)
		os.Exit(1)
	}

	latestDirectCSIDriveInterface, err = directCSIDriveInterfaceForConfig(config)
	if err != nil {
		fmt.Printf("%s: could not initialize drive adapter client: err=%v\n", utils.Bold("Error"), err)
		os.Exit(1)
	}

	latestDirectCSIVolumeInterface, err = directCSIVolumeInterfaceForConfig(config)
	if err != nil {
		fmt.Printf("%s: could not initialize volume adapter client: err=%v\n", utils.Bold("Error"), err)
		os.Exit(1)
	}

	initEvent(kubeClient)
}
