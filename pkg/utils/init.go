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
	"path/filepath"

	direct "github.com/minio/direct-csi/pkg/clientset"
	directcsi "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1beta2"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/spf13/viper"

	"k8s.io/klog/v2"
)

var directCSIClient directcsi.DirectV1beta2Interface
var directClientset direct.Interface
var kubeClient kubernetes.Interface
var crdClient apiextensions.CustomResourceDefinitionInterface
var apiextensionsClient apiextensions.ApiextensionsV1Interface
var discoveryClient discovery.DiscoveryInterface
var metadataClient metadata.Interface

var (
	initialized = false
)

const (
	MaxThreadCount = 40
)

func getKubeConfig() (*rest.Config, string, error) {
	kubeConfig := viper.GetString("kubeconfig")
	if kubeConfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			klog.Infof("could not find home dir: %v", err)
		}
		kubeConfig = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		if config, err = rest.InClusterConfig(); err != nil {
			return nil, "", err
		}
	}
	config.QPS = float32(MaxThreadCount / 2)
	config.Burst = MaxThreadCount
	return config, "", nil
}

func GetKubeConfig() (*rest.Config, error) {
	config, _, err := getKubeConfig()
	return config, err
}

func Init() {
	if initialized {
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

	initialized = true
}

func GetGroupKindVersions(group, kind string, versions ...string) (*schema.GroupVersionKind, error) {
	discoveryClient := GetDiscoveryClient()
	apiGroupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		klog.Errorf("could not obtain API group resources: %v", err)
		return nil, err
	}
	restMapper := restmapper.NewDiscoveryRESTMapper(apiGroupResources)
	gk := schema.GroupKind{
		Group: group,
		Kind:  kind,
	}
	mapper, err := restMapper.RESTMapping(gk, versions...)
	if err != nil {
		klog.Errorf("could not find valid restmapping: %v", err)
		return nil, err
	}

	gvk := &schema.GroupVersionKind{
		Group:   mapper.Resource.Group,
		Version: mapper.Resource.Version,
		Kind:    mapper.Resource.Resource,
	}
	return gvk, nil
}
