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
	"context"

	directv1alpha1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	directv1beta1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta1"
	directv1beta2 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	clientset "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/converter"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	rest "k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

var _ clientset.DirectCSIDriveInterface = &directCSIDriveAdapter{}

type directCSIDriveAdapter struct {
	dynamicClient dynamic.ResourceInterface
	gvk           *schema.GroupVersionKind
}

func directCSIDriveAdapterForConfig(config *rest.Config) (clientset.DirectCSIDriveInterface, error) {
	gvk, err := GetGroupKindVersions(
		directcsi.Group,
		"DirectCSIDrive",
		directcsi.Version,
		directv1beta2.Version,
		directv1beta1.Version,
		directv1alpha1.Version,
	)
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	dyncamicResourceClient := dynamicClient.Resource(
		schema.GroupVersionResource{
			Group:    directcsi.Group,
			Version:  gvk.Version,
			Resource: "directcsidrives",
		},
	)
	return &directCSIDriveAdapter{dynamicClient: dyncamicResourceClient}, nil
}

// Get takes name of the directCSIDrive, and returns the corresponding directCSIDrive object, and an error if there is any.
func (c *directCSIDriveAdapter) Get(
	ctx context.Context,
	name string,
	options v1.GetOptions) (*directcsi.DirectCSIDrive, error) {

	intermediateResult, err := c.dynamicClient.Get(ctx, name, options, "")
	if err != nil {
		klog.Infof("could not get intermediate result: %v", err)
		return nil, err
	}

	finalResult := &unstructured.Unstructured{}
	err = converter.Migrate(intermediateResult, finalResult, schema.GroupVersion{
		Version: directcsi.Version,
		Group:   directcsi.Group,
	})
	if err != nil {
		return nil, err
	}

	unstructuredObject := finalResult.Object
	var directCSIDrive directcsi.DirectCSIDrive
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject, &directCSIDrive)
	if err != nil {
		return nil, err
	}
	return &directCSIDrive, nil
}

// List takes label and field selectors, and returns the list of DirectCSIDrives that match those selectors.
func (c *directCSIDriveAdapter) List(ctx context.Context, opts v1.ListOptions) (result *directcsi.DirectCSIDriveList, err error) {
	intermediateResult, err := c.dynamicClient.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	finalResult := &unstructured.UnstructuredList{}
	err = converter.MigrateList(intermediateResult, finalResult, schema.GroupVersion{
		Version: directcsi.Version,
		Group:   directcsi.Group,
	})
	if err != nil {
		return nil, err
	}

	var directCSIDriveList directcsi.DirectCSIDriveList
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(finalResult.Object, &directCSIDriveList)
	if err != nil {
		return nil, err
	}

	items := []directcsi.DirectCSIDrive{}
	for i := range finalResult.Items {
		directCSIDrive := directcsi.DirectCSIDrive{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(finalResult.Items[i].Object, &directCSIDrive)
		if err != nil {
			return nil, err
		}
		items = append(items, directCSIDrive)
	}
	directCSIDriveList.Items = items

	return &directCSIDriveList, nil
}

// Watch returns a watch.Interface that watches the requested directCSIDrives.
func (c *directCSIDriveAdapter) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return nil, nil
	// var timeout time.Duration
	// if opts.TimeoutSeconds != nil {
	// 	timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	// }
	// opts.Watch = true
	// return c.client.Get().
	// 	Resource("directcsidrives").
	// 	VersionedParams(&opts, scheme.ParameterCodec).
	// 	Timeout(timeout).
	// 	Watch(ctx)
}

// Create takes the representation of a directCSIDrive and creates it.  Returns the server's representation of the directCSIDrive, and an error, if there is any.
func (c *directCSIDriveAdapter) Create(ctx context.Context, directCSIDrive *directcsi.DirectCSIDrive, opts v1.CreateOptions) (result *directcsi.DirectCSIDrive, err error) {
	// result = &directcsi.DirectCSIDrive{}
	// err = c.client.Post().
	// 	Resource("directcsidrives").
	// 	VersionedParams(&opts, scheme.ParameterCodec).
	// 	Body(directCSIDrive).
	// 	Do(ctx).
	// 	Into(result)
	return
}

// Update takes the representation of a directCSIDrive and updates it. Returns the server's representation of the directCSIDrive, and an error, if there is any.
func (c *directCSIDriveAdapter) Update(ctx context.Context, directCSIDrive *directcsi.DirectCSIDrive, opts v1.UpdateOptions) (result *directcsi.DirectCSIDrive, err error) {
	// result = &directcsi.DirectCSIDrive{}
	// err = c.client.Put().
	// 	Resource("directcsidrives").
	// 	Name(directCSIDrive.Name).
	// 	VersionedParams(&opts, scheme.ParameterCodec).
	// 	Body(directCSIDrive).
	// 	Do(ctx).
	// 	Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *directCSIDriveAdapter) UpdateStatus(ctx context.Context, directCSIDrive *directcsi.DirectCSIDrive, opts v1.UpdateOptions) (result *directcsi.DirectCSIDrive, err error) {
	// result = &directcsi.DirectCSIDrive{}
	// err = c.client.Put().
	// 	Resource("directcsidrives").
	// 	Name(directCSIDrive.Name).
	// 	SubResource("status").
	// 	VersionedParams(&opts, scheme.ParameterCodec).
	// 	Body(directCSIDrive).
	// 	Do(ctx).
	// 	Into(result)
	return
}

// Delete takes name of the directCSIDrive and deletes it. Returns an error if one occurs.
func (c *directCSIDriveAdapter) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return nil
	// return c.client.Delete().
	// 	Resource("directcsidrives").
	// 	Name(name).
	// 	Body(&opts).
	// 	Do(ctx).
	// 	Error()
}

// DeleteCollection deletes a collection of objects.
func (c *directCSIDriveAdapter) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	return nil
	// var timeout time.Duration
	// if listOpts.TimeoutSeconds != nil {
	// 	timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	// }
	// return c.client.Delete().
	// 	Resource("directcsidrives").
	// 	VersionedParams(&listOpts, scheme.ParameterCodec).
	// 	Timeout(timeout).
	// 	Body(&opts).
	// 	Do(ctx).
	// 	Error()
}

// Patch applies the patch and returns the patched directCSIDrive.
func (c *directCSIDriveAdapter) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *directcsi.DirectCSIDrive, err error) {
	// result = &directcsi.DirectCSIDrive{}
	// err = c.client.Patch(pt).
	// 	Resource("directcsidrives").
	// 	Name(name).
	// 	SubResource(subresources...).
	// 	VersionedParams(&opts, scheme.ParameterCodec).
	// 	Body(data).
	// 	Do(ctx).
	// 	Into(result)
	return
}

// APIVersion returns the APIVersion this RESTClient is expected to use.
func (c *directCSIDriveAdapter) APIVersion() schema.GroupVersion {
	return c.gvk.GroupVersion()
}
