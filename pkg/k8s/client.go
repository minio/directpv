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
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/minio/directpv/pkg/consts"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/klog/v2"
)

// Client represents various kubernetes clients.
type Client struct {
	KubeConfig          *rest.Config
	KubeClient          kubernetes.Interface
	APIextensionsClient apiextensions.ApiextensionsV1Interface
	CRDClient           apiextensions.CustomResourceDefinitionInterface
	DiscoveryClient     discovery.DiscoveryInterface
}

// NewClient returns new kubernetes client bundle.
func NewClient(kubeConfig *rest.Config) (*Client, error) {
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create new kubernetes client interface; %w", err)
	}
	apiextensionsClient, err := apiextensions.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create new API extensions client interface; %w", err)
	}
	crdClient := apiextensionsClient.CustomResourceDefinitions()
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create new discovery client interface; %w", err)
	}
	return &Client{
		KubeConfig:          kubeConfig,
		KubeClient:          kubeClient,
		APIextensionsClient: apiextensionsClient,
		CRDClient:           crdClient,
		DiscoveryClient:     discoveryClient,
	}, nil
}

// GetKubeVersion returns kubernetes version information.
func (client *Client) GetKubeVersion() (major, minor uint, err error) {
	versionInfo, err := client.DiscoveryClient.ServerVersion()
	if err != nil {
		return 0, 0, err
	}

	var u64 uint64
	if u64, err = strconv.ParseUint(versionInfo.Major, 10, 64); err != nil {
		return 0, 0, fmt.Errorf("unable to parse major version %v; %w", versionInfo.Major, err)
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
		return 0, 0, fmt.Errorf("unable to parse minor version %v; %w", minor, err)
	}
	minor = uint(u64)
	return major, minor, nil
}

// GetGroupVersionKind gets group/version/kind of given versions.
func (client *Client) GetGroupVersionKind(group, kind string, versions ...string) (*schema.GroupVersionKind, error) {
	apiGroupResources, err := restmapper.GetAPIGroupResources(client.DiscoveryClient)
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
func (client *Client) GetClientForNonCoreGroupVersionKind(group, kind string, versions ...string) (rest.Interface, *schema.GroupVersionKind, error) {
	gvk, err := client.GetGroupVersionKind(group, kind, versions...)
	if err != nil {
		return nil, nil, err
	}

	config := *client.KubeConfig
	config.GroupVersion = &schema.GroupVersion{
		Group:   gvk.Group,
		Version: gvk.Version,
	}
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	restClient, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, nil, err
	}

	return restClient, gvk, nil
}

// GetCSINodes fetches the CSI Node list
func (client *Client) GetCSINodes(ctx context.Context) (nodes []string, err error) {
	storageClient, gvk, err := client.GetClientForNonCoreGroupVersionKind("storage.k8s.io", "CSINode", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return nil, err
	}

	switch gvk.Version {
	case "v1apha1":
		err = errors.New("unsupported CSINode storage.k8s.io/v1alpha1")
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
