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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

var _ clientset.DirectCSIVolumeInterface = &directCSIVolumeAdapter{}

type directCSIVolumeAdapter struct {
	resourceInterface dynamic.ResourceInterface
	gvk               *schema.GroupVersionKind
}

func directCSIVolumeAdapterForConfig(config *rest.Config) (clientset.DirectCSIVolumeInterface, error) {
	gvk, err := GetGroupKindVersions(
		directcsi.Group,
		"DirectCSIVolume",
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
	resourceInterface, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	dynamicResourceClient := resourceInterface.Resource(
		schema.GroupVersionResource{
			Group:    directcsi.Group,
			Version:  version,
			Resource: "directcsivolumes",
		},
	)

	return &directCSIVolumeAdapter{resourceInterface: dynamicResourceClient, gvk: gvk}, nil
}

// Get takes name of the directCSIVolume, and returns the latest directCSVolume object, and an error if there is any.
func (c *directCSIVolumeAdapter) Get(
	ctx context.Context,
	name string,
	options metav1.GetOptions) (*directcsi.DirectCSIVolume, error) {
	intermediateResult, err := c.resourceInterface.Get(ctx, name, options, "")
	if err != nil {
		klog.Infof("could not get intermediate result: %v", err)
		return nil, err
	}
	finalResult := &unstructured.Unstructured{}
	if err := converter.Migrate(intermediateResult, finalResult, schema.GroupVersion{
		Version: directcsi.Version,
		Group:   directcsi.Group,
	}); err != nil {

		return nil, err
	}
	unstructuredObject := finalResult.Object
	var directCSIVolume directcsi.DirectCSIVolume
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject, &directCSIVolume); err != nil {
		return nil, err
	}
	return &directCSIVolume, nil
}

// List takes label and field selectors, and returns the list of DirectCSIVolume that match those selectors.
func (c *directCSIVolumeAdapter) List(ctx context.Context, opts metav1.ListOptions) (result *directcsi.DirectCSIVolumeList, err error) {
	intermediateResult, err := c.resourceInterface.List(ctx, opts)
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

	var directCSIVolumeList directcsi.DirectCSIVolumeList
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(finalResult.Object, &directCSIVolumeList)
	if err != nil {
		return nil, err
	}

	items := []directcsi.DirectCSIVolume{}
	for i := range finalResult.Items {
		directCSIVolume := directcsi.DirectCSIVolume{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(finalResult.Items[i].Object, &directCSIVolume)
		if err != nil {
			return nil, err
		}
		items = append(items, directCSIVolume)
	}
	directCSIVolumeList.Items = items

	return &directCSIVolumeList, nil
}

// Watch returns a watch.Interface that watches the requested directCSIVolumes.
func (c *directCSIVolumeAdapter) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.resourceInterface.Watch(ctx, opts)
}

// Create takes the representation of a directCSIVolume and creates it.  Returns the server's representation of the directCSIVolume, and an error, if there is any.
func (c *directCSIVolumeAdapter) Create(ctx context.Context, directCSIVolume *directcsi.DirectCSIVolume, opts metav1.CreateOptions) (result *directcsi.DirectCSIVolume, err error) {
	unstructured := &unstructured.Unstructured{}
	convertedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(directCSIVolume)
	if err != nil {
		return nil, err
	}
	unstructured.Object = convertedObj

	createdUnstructuredObj, err := c.resourceInterface.Create(ctx, unstructured, opts, "")
	if err != nil {
		return nil, err
	}

	var createdDirectCSIVolume directcsi.DirectCSIVolume
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(createdUnstructuredObj.Object, &createdDirectCSIVolume)
	if err != nil {
		return nil, err
	}
	return &createdDirectCSIVolume, nil
}

// Update takes the representation of a directCSIVolume and updates it. Returns the server's representation of the directCSIVolume, and an error, if there is any.
func (c *directCSIVolumeAdapter) Update(ctx context.Context, directCSIVolume *directcsi.DirectCSIVolume, opts metav1.UpdateOptions) (result *directcsi.DirectCSIVolume, err error) {
	unstructured := &unstructured.Unstructured{}
	convertedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(directCSIVolume)
	if err != nil {
		return nil, err
	}
	unstructured.Object = convertedObj

	updatedUnstructuredObj, err := c.resourceInterface.Update(ctx, unstructured, opts, "")
	if err != nil {
		return nil, err
	}

	var updatedDirectCSIVolume directcsi.DirectCSIVolume
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(updatedUnstructuredObj.Object, &updatedDirectCSIVolume)
	if err != nil {
		return nil, err
	}
	return &updatedDirectCSIVolume, nil
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *directCSIVolumeAdapter) UpdateStatus(ctx context.Context, directCSIVolume *directcsi.DirectCSIVolume, opts metav1.UpdateOptions) (result *directcsi.DirectCSIVolume, err error) {
	unstructured := &unstructured.Unstructured{}
	convertedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(directCSIVolume)
	if err != nil {
		return nil, err
	}
	unstructured.Object = convertedObj

	updatedUnstructuredObj, err := c.resourceInterface.UpdateStatus(ctx, unstructured, opts)
	if err != nil {
		return nil, err
	}

	var updatedDirectCSIVolume directcsi.DirectCSIVolume
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(updatedUnstructuredObj.Object, &updatedDirectCSIVolume)
	if err != nil {
		return nil, err
	}
	return &updatedDirectCSIVolume, nil
}

// Delete takes name of the directCSIVolume and deletes it. Returns an error if one occurs.
func (c *directCSIVolumeAdapter) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.resourceInterface.Delete(ctx, name, opts, "")
}

// DeleteCollection deletes a collection of objects.
func (c *directCSIVolumeAdapter) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return c.resourceInterface.DeleteCollection(ctx, opts, listOpts)
}

// Patch applies the patch and returns the patched directCSIVolume.
func (c *directCSIVolumeAdapter) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *directcsi.DirectCSIVolume, err error) {
	patchedUnsrtucturedObj, err := c.resourceInterface.Patch(ctx, name, pt, data, opts, subresources...)
	if err != nil {
		return nil, err
	}
	var patchedDirectCSIVolume directcsi.DirectCSIVolume
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(patchedUnsrtucturedObj.Object, &patchedDirectCSIVolume)
	if err != nil {
		return nil, err
	}
	return &patchedDirectCSIVolume, nil
}

// APIVersion returns the APIVersion this RESTClient is expected to use.
func (c *directCSIVolumeAdapter) APIVersion() schema.GroupVersion {
	return c.gvk.GroupVersion()
}
