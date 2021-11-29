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

	"k8s.io/apimachinery/pkg/api/meta"
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
	if err != nil && !meta.IsNoMatchError(err) {
		return nil, err
	}
	version := directcsi.Version
	if gvk != nil {
		version = gvk.Version
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	dyncamicResourceClient := dynamicClient.Resource(
		schema.GroupVersionResource{
			Group:    directcsi.Group,
			Version:  version,
			Resource: "directcsidrives",
		},
	)
	return &directCSIDriveAdapter{dynamicClient: dyncamicResourceClient, gvk: gvk}, nil
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
	opts.Watch = true
	return c.dynamicClient.Watch(ctx, opts)
}

// Create takes the representation of a directCSIDrive and creates it.  Returns the server's representation of the directCSIDrive, and an error, if there is any.
func (c *directCSIDriveAdapter) Create(ctx context.Context, directCSIDrive *directcsi.DirectCSIDrive, opts v1.CreateOptions) (result *directcsi.DirectCSIDrive, err error) {
	unstructured := &unstructured.Unstructured{}
	convertedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(directCSIDrive)
	if err != nil {
		return nil, err
	}
	unstructured.Object = convertedObj

	createdUnstructuredObj, err := c.dynamicClient.Create(ctx, unstructured, opts, "")
	if err != nil {
		return nil, err
	}

	var createdDirectCSIDrive directcsi.DirectCSIDrive
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(createdUnstructuredObj.Object, &createdDirectCSIDrive)
	if err != nil {
		return nil, err
	}
	return &createdDirectCSIDrive, nil
}

// Update takes the representation of a directCSIDrive and updates it. Returns the server's representation of the directCSIDrive, and an error, if there is any.
func (c *directCSIDriveAdapter) Update(ctx context.Context, directCSIDrive *directcsi.DirectCSIDrive, opts v1.UpdateOptions) (result *directcsi.DirectCSIDrive, err error) {
	unstructured := &unstructured.Unstructured{}
	convertedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(directCSIDrive)
	if err != nil {
		return nil, err
	}
	unstructured.Object = convertedObj

	updatedUnstructuredObj, err := c.dynamicClient.Update(ctx, unstructured, opts, "")
	if err != nil {
		return nil, err
	}

	var updatedDirectCSIDrive directcsi.DirectCSIDrive
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(updatedUnstructuredObj.Object, &updatedDirectCSIDrive)
	if err != nil {
		return nil, err
	}
	return &updatedDirectCSIDrive, nil
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *directCSIDriveAdapter) UpdateStatus(ctx context.Context, directCSIDrive *directcsi.DirectCSIDrive, opts v1.UpdateOptions) (result *directcsi.DirectCSIDrive, err error) {
	unstructured := &unstructured.Unstructured{}
	convertedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(directCSIDrive)
	if err != nil {
		return nil, err
	}
	unstructured.Object = convertedObj

	updatedUnstructuredObj, err := c.dynamicClient.UpdateStatus(ctx, unstructured, opts)
	if err != nil {
		return nil, err
	}

	var updatedDirectCSIDrive directcsi.DirectCSIDrive
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(updatedUnstructuredObj.Object, &updatedDirectCSIDrive)
	if err != nil {
		return nil, err
	}
	return &updatedDirectCSIDrive, nil
}

// Delete takes name of the directCSIDrive and deletes it. Returns an error if one occurs.
func (c *directCSIDriveAdapter) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.dynamicClient.Delete(ctx, name, opts, "")
}

// DeleteCollection deletes a collection of objects.
func (c *directCSIDriveAdapter) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	return c.dynamicClient.DeleteCollection(ctx, opts, listOpts)
}

// Patch applies the patch and returns the patched directCSIDrive.
func (c *directCSIDriveAdapter) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *directcsi.DirectCSIDrive, err error) {
	patchedUnsrtucturedObj, err := c.dynamicClient.Patch(ctx, name, pt, data, opts, subresources...)
	if err != nil {
		return nil, err
	}
	var patchedDirectCSIDrive directcsi.DirectCSIDrive
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(patchedUnsrtucturedObj.Object, &patchedDirectCSIDrive)
	if err != nil {
		return nil, err
	}
	return &patchedDirectCSIDrive, nil
}

// APIVersion returns the APIVersion this RESTClient is expected to use.
func (c *directCSIDriveAdapter) APIVersion() schema.GroupVersion {
	return c.gvk.GroupVersion()
}
