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
	direct "github.com/minio/direct-csi/pkg/clientset"
	directcsi "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1beta2"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/klog"
)

var (
	fakeMode bool
)

func SetFake() {
	fakeMode = true

	InitFake()
}

func GetFake() bool {
	return fakeMode
}

func GetKubeClient() kubernetes.Interface {
	if fakeMode {
		return fakeKubeClient
	}
	return kubeClient
}

func GetDirectCSIClient() directcsi.DirectV1beta2Interface {
	if fakeMode {
		return fakeDirectCSIClient
	}
	return directCSIClient
}

func GetDirectClientset() direct.Interface {
	if fakeMode {
		return fakeDirectClientset
	}
	return directClientset
}

func GetCRDClient() apiextensions.CustomResourceDefinitionInterface {
	if fakeMode {
		return fakeCRDClient
	}
	return crdClient
}

func GetAPIExtensionsClient() apiextensions.ApiextensionsV1Interface {
	if fakeMode {
		return fakeAPIExtenstionsClient
	}
	return apiextensionsClient
}

func GetDiscoveryClient() discovery.DiscoveryInterface {
	if fakeMode {
		return fakeDiscoveryClient
	}
	return discoveryClient
}

func GetMetadataClient() metadata.Interface {
	if fakeMode {
		return fakeMetadataClient
	}
	return metadataClient
}

func GetClientForNonCoreGroupKindVersions(group, kind string, versions ...string) (rest.Interface, *schema.GroupVersionKind, error) {
	gvk, err := GetGroupKindVersions(group, kind, versions...)
	if err != nil {
		return nil, nil, err
	}

	gv := &schema.GroupVersion{
		Group:   gvk.Group,
		Version: gvk.Version,
	}
	kubeConfig := GetKubeConfig()
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			klog.Fatalf("could not find client configuration: %v", err)
		}
		klog.V(1).Infof("obtained client config successfully")
	}
	config.GroupVersion = gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	client, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, nil, err
	}
	return client, gvk, nil
}
