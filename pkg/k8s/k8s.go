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
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/viper"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

// GetKubeConfig gets kubernetes configuration from command line argument,
// ~/.kube/config or in-cluster configuration.
func GetKubeConfig() (*rest.Config, error) {
	kubeconfigPath := viper.GetString("kubeconfig")
	if kubeconfigPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			klog.ErrorS(err, "unable to find user home directory")
		}
		kubeconfigPath = path.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		if config, err = rest.InClusterConfig(); err != nil {
			return nil, err
		}
	}

	config.QPS = float32(MaxThreadCount / 2)
	config.Burst = MaxThreadCount
	return config, nil
}

// GetGroupVersionKind gets group/version/kind of given versions.
func GetGroupVersionKind(group, kind string, versions ...string) (*schema.GroupVersionKind, error) {
	apiGroupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		klog.ErrorS(err, "unable to get API group resources")
		return nil, err
	}
	restMapper := restmapper.NewDiscoveryRESTMapper(apiGroupResources)
	mapper, err := restMapper.RESTMapping(
		schema.GroupKind{
			Group: group,
			Kind:  kind,
		},
		versions...,
	)
	if err != nil {
		return nil, err
	}

	return &schema.GroupVersionKind{
		Group:   mapper.Resource.Group,
		Version: mapper.Resource.Version,
		Kind:    mapper.Resource.Resource,
	}, nil
}

// GetClientForNonCoreGroupVersionKind gets client for group/kind of given versions.
func GetClientForNonCoreGroupVersionKind(group, kind string, versions ...string) (rest.Interface, *schema.GroupVersionKind, error) {
	gvk, err := GetGroupVersionKind(group, kind, versions...)
	if err != nil {
		return nil, nil, err
	}

	config := *kubeConfig
	config.GroupVersion = &schema.GroupVersion{
		Group:   gvk.Group,
		Version: gvk.Version,
	}
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, nil, err
	}

	return client, gvk, nil
}

// IsCondition checks whether type/status/reason/message in conditions or not.
func IsCondition(conditions []metav1.Condition, ctype string, status metav1.ConditionStatus, reason, message string) bool {
	for i := range conditions {
		if conditions[i].Type == ctype &&
			conditions[i].Status == status &&
			conditions[i].Reason == reason &&
			conditions[i].Message == message {
			return true
		}
	}
	return false
}

// UpdateCondition updates type/status/reason/message of conditions matched by condition type.
func UpdateCondition(conditions []metav1.Condition, ctype string, status metav1.ConditionStatus, reason, message string) {
	for i := range conditions {
		if conditions[i].Type == ctype {
			conditions[i].Status = status
			conditions[i].Reason = reason
			conditions[i].Message = message
			conditions[i].LastTransitionTime = metav1.Now()
			break
		}
	}
}

// MatchTrueConditions matches whether types and status list are in a true conditions or not.
func MatchTrueConditions(conditions []metav1.Condition, types, statusList []string) bool {
	for i := range types {
		types[i] = strings.ToLower(types[i])
	}
	for i := range statusList {
		statusList[i] = strings.ToLower(statusList[i])
	}

	statusMatches := 0
	for _, condition := range conditions {
		ctype := strings.ToLower(condition.Type)
		if condition.Status == metav1.ConditionTrue && utils.Contains(types, ctype) && utils.Contains(statusList, ctype) {
			statusMatches++
		}
	}

	return statusMatches == len(statusList)
}

// BoolToConditionStatus converts boolean value to condition status.
func BoolToConditionStatus(val bool) metav1.ConditionStatus {
	if val {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}

// SanitizeResourceName - Sanitize given name to a valid kubernetes name format.
// RegEx for a kubernetes name is
//
//	([a-z0-9][-a-z0-9]*)?[a-z0-9]
//
// with a max length of 253
//
// WARNING: This function will truncate to 253 bytes if the input is longer
func SanitizeResourceName(name string) string {
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

// GetCSINodes fetches the CSI Node list
func GetCSINodes(ctx context.Context) (nodes []string, err error) {
	storageClient, gvk, err := GetClientForNonCoreGroupVersionKind("storage.k8s.io", "CSINode", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return nil, err
	}

	switch gvk.Version {
	case "v1apha1":
		err = fmt.Errorf("unsupported CSINode storage.k8s.io/v1alpha1")
	case "v1":
		result := &storagev1.CSINodeList{}
		if err = storageClient.Get().
			Resource("csinodes").
			VersionedParams(&metav1.ListOptions{}, scheme.ParameterCodec).
			Timeout(10 * time.Second).
			Do(ctx).
			Into(result); err != nil {
			err = fmt.Errorf("unable to get csinodes; %w", err)
			break
		}
		for _, csiNode := range result.Items {
			for _, driver := range csiNode.Spec.Drivers {
				if driver.Name == consts.Identity {
					nodes = append(nodes, csiNode.Name)
					break
				}
			}
		}
	case "v1beta1":
		result := &storagev1beta1.CSINodeList{}
		if err = storageClient.Get().
			Resource(gvk.Kind).
			VersionedParams(&metav1.ListOptions{}, scheme.ParameterCodec).
			Timeout(10 * time.Second).
			Do(ctx).
			Into(result); err != nil {
			err = fmt.Errorf("unable to get csinodes; %w", err)
			break
		}
		for _, csiNode := range result.Items {
			for _, driver := range csiNode.Spec.Drivers {
				if driver.Name == consts.Identity {
					nodes = append(nodes, csiNode.Name)
					break
				}
			}
		}
	}

	return nodes, err
}
