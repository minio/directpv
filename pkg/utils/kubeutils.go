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
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	// support gcp, azure, and oidc client auth
	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func SetLabels(object metav1.Object, labels map[LabelKey]LabelValue) {
	values := make(map[string]string)
	for key, value := range labels {
		values[string(key)] = string(value)
	}
	object.SetLabels(values)
}

// UpdateLabels updates labels in object.
func UpdateLabels(object metav1.Object, labels map[LabelKey]LabelValue) {
	values := object.GetLabels()
	if values == nil {
		values = make(map[string]string)
	}

	for key, value := range labels {
		values[string(key)] = string(value)
	}

	object.SetLabels(values)
}

// SanitizeKubeResourceName - Sanitize given name to a valid kubernetes name format.
// RegEx for a kubernetes name is
//
//      ([a-z0-9][-a-z0-9]*)?[a-z0-9]
//
// with a max length of 253
//
// WARNING: This function will truncate to 253 bytes if the input is longer
func SanitizeKubeResourceName(name string) string {
	if len(name) > 253 {
		name = name[:253]
	}

	result := []rune(strings.ToLower(name))
	for i, r := range result {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
		default:
			if i == 0 {
				result[i] = '0'
			} else {
				result[i] = '-'
			}
		}
	}

	return string(result)
}

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

// GetKubeConfig gets kubernetes configuration.
func GetKubeConfig() (*rest.Config, error) {
	config, _, err := getKubeConfig()
	return config, err
}

// GetGroupKindVersions gets group/version/kind of given versions.
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
