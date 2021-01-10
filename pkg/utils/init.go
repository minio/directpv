// This file is part of MinIO Direct CSI
// Copyright (c) 2020 MinIO, Inc.
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
	"os"
	"path/filepath"

	direct "github.com/minio/direct-csi/pkg/clientset"
	directv1alpha1 "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1alpha1"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/golang/glog"
	"github.com/spf13/viper"
)

var directCSIClient directv1alpha1.DirectV1alpha1Interface
var directClientset direct.Interface
var kubeClient kubernetes.Interface
var crdClient apiextensions.CustomResourceDefinitionInterface
var discoveryClient discovery.DiscoveryInterface
var metadataClient metadata.Interface
var gvk *schema.GroupVersionKind

func Init() {
	kubeConfig := GetKubeConfig()
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			glog.Fatalf("could not find client configuration: %v", err)
		}
		glog.Infof("obtained client config successfully")
	}

	kubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("could not initialize kubeclient: %v", err)
	}

	directClientset, err = direct.NewForConfig(config)
	if err != nil {
		glog.Fatalf("could not initialize direct clientset: %v", err)
	}

	directCSIClient, err = directv1alpha1.NewForConfig(config)
	if err != nil {
		glog.Fatalf("could not initialize direct-csi client: %v", err)
	}

	crdClientset, err := apiextensions.NewForConfig(config)
	if err != nil {
		glog.Fatalf("could not initialize crd client: %v", err)
	}
	crdClient = crdClientset.CustomResourceDefinitions()

	discoveryClient, err = discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		glog.Fatalf("could not initialize discovery client: %v", err)
	}

	metadataClient, err = metadata.NewForConfig(config)
	if err != nil {
		glog.Fatalf("could not initialize metadata client: %v", err)
	}
}

func GetKubeConfig() string {
	kubeConfig := viper.GetString("kubeconfig")
	if kubeConfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			glog.Infof("could not find home dir: %v", err)
			return ""
		}
		return filepath.Join(home, ".kube", "config")
	}
	return kubeConfig
}

func GetGroupKindVersions(group, kind string, versions ...string) (*schema.GroupVersionKind, error) {
	if gvk != nil {
		return gvk, nil
	}
	discoveryClient := GetDiscoveryClient()
	apiGroupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		glog.Errorf("could not obtain API group resources: %v", err)
		return nil, err
	}
	restMapper := restmapper.NewDiscoveryRESTMapper(apiGroupResources)
	gk := schema.GroupKind{
		Group: group,
		Kind:  kind,
	}
	mapper, err := restMapper.RESTMapping(gk, versions...)
	if err != nil {
		glog.Errorf("could not find valid restmapping: %v", err)
		return nil, err
	}

	gvk = &schema.GroupVersionKind{
		Group:   mapper.Resource.Group,
		Version: mapper.Resource.Version,
		Kind:    mapper.Resource.Resource,
	}
	return gvk, nil
}
