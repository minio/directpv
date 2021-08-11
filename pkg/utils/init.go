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

package utils

import (
	"fmt"
	"os"
	"sync/atomic"

	direct "github.com/minio/direct-csi/pkg/clientset"
	directcsi "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1beta2"
	directcsifake "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1beta2/fake"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"

	"k8s.io/klog/v2"
)

const MaxThreadCount = 40

var (
	initialized         int32
	kubeClient          kubernetes.Interface
	directCSIClient     directcsi.DirectV1beta2Interface
	directClientset     direct.Interface
	apiextensionsClient apiextensions.ApiextensionsV1Interface
	crdClient           apiextensions.CustomResourceDefinitionInterface
	discoveryClient     discovery.DiscoveryInterface
	metadataClient      metadata.Interface
)

func GetKubeClient() kubernetes.Interface {
	return kubeClient
}

func GetDirectCSIClient() directcsi.DirectV1beta2Interface {
	return directCSIClient
}

func GetDirectClientset() direct.Interface {
	return directClientset
}

func GetAPIExtensionsClient() apiextensions.ApiextensionsV1Interface {
	return apiextensionsClient
}

func GetCRDClient() apiextensions.CustomResourceDefinitionInterface {
	return crdClient
}

func GetDiscoveryClient() discovery.DiscoveryInterface {
	return discoveryClient
}

func GetMetadataClient() metadata.Interface {
	return metadataClient
}

func Init() {
	if atomic.AddInt32(&initialized, 1) != 1 {
		return
	}

	config, kubeConfig, err := getKubeConfig()
	if err != nil {
		fmt.Printf("%s: Could not connect to kubernetes. %s=%s\n", Bold("Error"), "KUBECONFIG", kubeConfig)
		os.Exit(1)
	}
	klog.V(1).Infof("obtained client config successfully")

	kubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("%s: could not initialize kube client: err=%v\n", Bold("Error"), err)
		os.Exit(1)
	}

	directClientset, err = direct.NewForConfig(config)
	if err != nil {
		fmt.Printf("%s: could not initialize direct-csi client: err=%v\n", Bold("Error"), err)
		os.Exit(1)
	}

	directCSIClient, err = directcsi.NewForConfig(config)
	if err != nil {
		klog.Fatalf("could not initialize direct-csi client: %v", err)
	}

	crdClientset, err := apiextensions.NewForConfig(config)
	if err != nil {
		fmt.Printf("%s: could not initialize crd client: err=%v\n", Bold("Error"), err)
		os.Exit(1)
	}
	apiextensionsClient = crdClientset
	crdClient = crdClientset.CustomResourceDefinitions()

	discoveryClient, err = discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		fmt.Printf("%s: could not initialize discovery client: err=%v\n", Bold("Error"), err)
		os.Exit(1)
	}

	metadataClient, err = metadata.NewForConfig(config)
	if err != nil {
		fmt.Printf("%s: could not initialize metadata client: err=%v\n", Bold("Error"), err)
		os.Exit(1)
	}
}

func SetDirectCSIClient(fakeClient *directcsifake.FakeDirectV1beta2) {
	directCSIClient = fakeClient
}
