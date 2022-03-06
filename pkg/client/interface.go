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

package client

import (
	"context"

	directv1alpha1 "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1alpha1"
	directv1beta1 "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta1"
	directv1beta2 "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta2"
	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/converter"
	"github.com/minio/directpv/pkg/utils"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

// GetLatestDirectCSIRESTClient gets REST client of the latest direct-csi.
func GetLatestDirectCSIRESTClient() rest.Interface {
	return GetDirectClientset().DirectV1beta4().RESTClient()
}

func toDirectCSIDrive(object map[string]interface{}) (*directcsi.DirectCSIDrive, error) {
	var drive directcsi.DirectCSIDrive
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object, &drive); err != nil {
		return nil, err
	}
	return &drive, nil
}

func toDirectCSIVolume(object map[string]interface{}) (*directcsi.DirectCSIVolume, error) {
	var volume directcsi.DirectCSIVolume
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object, &volume); err != nil {
		return nil, err
	}
	return &volume, nil
}

type directCSIInterface struct {
	resourceInterface dynamic.ResourceInterface
	groupVersion      schema.GroupVersion
}

func directCSIInterfaceForConfig(config *rest.Config, kind, resource string) (*directCSIInterface, error) {
	gvk, err := GetGroupKindVersions(
		directcsi.Group,
		kind,
		directcsi.Version,
		directv1beta2.Version,
		directv1beta1.Version,
		directv1alpha1.Version,
	)
	if err != nil && !meta.IsNoMatchError(err) {
		return nil, err
	}
	resourceInterface, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	version := directcsi.Version
	if gvk != nil {
		version = gvk.Version
	}
	group := directcsi.Group
	if gvk != nil {
		group = gvk.Group
	}
	return &directCSIInterface{
		resourceInterface: resourceInterface.Resource(
			schema.GroupVersionResource{
				Group:    directcsi.Group,
				Version:  version,
				Resource: resource,
			},
		),
		groupVersion: schema.GroupVersion{Group: group, Version: version},
	}, nil
}

// Create takes the representation of a resource object and creates it.  Returns the server's representation of the object, and an error, if there is any.
func (d *directCSIInterface) Create(ctx context.Context, object map[string]interface{}, opts metav1.CreateOptions) (map[string]interface{}, error) {
	unstructured := unstructured.Unstructured{Object: object}
	result, err := d.resourceInterface.Create(ctx, &unstructured, opts, "")
	if err != nil {
		return nil, err
	}
	return result.Object, nil
}

// Update takes the representation of a resource object and updates it. Returns the server's representation of the object, and an error, if there is any.
func (d *directCSIInterface) Update(ctx context.Context, object map[string]interface{}, opts metav1.UpdateOptions) (map[string]interface{}, error) {
	unstructured := unstructured.Unstructured{Object: object}
	result, err := d.resourceInterface.Update(ctx, &unstructured, opts, "")
	if err != nil {
		return nil, err
	}
	return result.Object, nil
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (d *directCSIInterface) UpdateStatus(ctx context.Context, object map[string]interface{}, opts metav1.UpdateOptions) (map[string]interface{}, error) {
	unstructured := unstructured.Unstructured{Object: object}
	result, err := d.resourceInterface.UpdateStatus(ctx, &unstructured, opts)
	if err != nil {
		return nil, err
	}
	return result.Object, nil
}

// Delete takes name of the resource object and deletes it. Returns an error if one occurs.
func (d *directCSIInterface) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return d.resourceInterface.Delete(ctx, name, opts, "")
}

// DeleteCollection deletes a collection of resource objects.
func (d *directCSIInterface) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return d.resourceInterface.DeleteCollection(ctx, opts, listOpts)
}

// Get takes name of the resource, and returns the latest resource object, and an error if there is any.
func (d *directCSIInterface) Get(ctx context.Context, name string, options metav1.GetOptions) (map[string]interface{}, error) {
	result, err := d.resourceInterface.Get(ctx, name, options, "")
	if err != nil {
		return nil, err
	}

	var migratedResult unstructured.Unstructured
	err = converter.Migrate(result, &migratedResult, schema.GroupVersion{Version: directcsi.Version, Group: directcsi.Group})
	if err != nil {
		return nil, err
	}

	return migratedResult.Object, nil
}

// List takes label and field selectors, and returns the list of resource objects that match those selectors.
func (d *directCSIInterface) List(ctx context.Context, opts metav1.ListOptions) (map[string]interface{}, []map[string]interface{}, error) {
	var labelSelector labels.Selector
	var err error
	switch d.groupVersion.Version {
	case "v1alpha1", "v1beta1":
		// As v1alpha1 and v1beta1 objects do not support labels, we fallback to filter here i.e. client side.
		if labelSelector, err = labels.Parse(opts.LabelSelector); err != nil {
			return nil, nil, err
		}
		opts.LabelSelector = ""
	}

	result, err := d.resourceInterface.List(ctx, opts)
	if err != nil {
		return nil, nil, err
	}

	var migratedResult unstructured.UnstructuredList
	err = converter.MigrateList(result, &migratedResult, schema.GroupVersion{Version: directcsi.Version, Group: directcsi.Group})
	if err != nil {
		return nil, nil, err
	}

	items := []map[string]interface{}{}
	for i := range migratedResult.Items {
		if labelSelector == nil || labelSelector.Matches(labels.Set(migratedResult.Items[i].GetLabels())) {
			items = append(items, migratedResult.Items[i].Object)
		}
	}
	return migratedResult.Object, items, nil
}

type watchInterfaceWrapper struct {
	watchInterface watch.Interface
}

func (w watchInterfaceWrapper) Stop() {
	w.watchInterface.Stop()
}

func (w watchInterfaceWrapper) ResultChan() <-chan watch.Event {
	wrapperCh := make(chan watch.Event)
	go func() {
		defer close(wrapperCh)
		resultCh := w.watchInterface.ResultChan()
		for {
			result, ok := <-resultCh
			if !ok {
				break
			}
			convertedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&result.Object)
			if err != nil {
				break
			}
			intermediateResult := &unstructured.Unstructured{Object: convertedObj}
			finalResult := &unstructured.Unstructured{}
			if err := converter.Migrate(intermediateResult, finalResult, schema.GroupVersion{
				Version: directcsi.Version,
				Group:   directcsi.Group,
			}); err != nil {
				klog.Infof("error while migrating: %v", err)
				break
			}

			wrapperCh <- watch.Event{
				Type:   result.Type,
				Object: finalResult,
			}
		}
	}()
	return wrapperCh
}

// Watch returns a watch.Interface that watches the requested directCSIDrives.
func (c *directCSIInterface) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	watcher, err := c.resourceInterface.Watch(ctx, opts)
	if err != nil {
		return nil, err
	}
	return watchInterfaceWrapper{watchInterface: watcher}, nil
}

// Patch applies the patch and returns the patched resource object.
func (d *directCSIInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (map[string]interface{}, error) {
	result, err := d.resourceInterface.Patch(ctx, name, pt, data, opts, subresources...)
	if err != nil {
		return nil, err
	}
	return result.Object, nil
}

// APIVersion returns the APIVersion this RESTClient is expected to use.
func (d *directCSIInterface) APIVersion() schema.GroupVersion {
	return d.groupVersion
}

// directCSIDriveInterface has methods to work with DirectCSIDrive resources.
type directCSIDriveInterface struct {
	directCSIInterface
}

func directCSIDriveInterfaceForConfig(config *rest.Config) (*directCSIDriveInterface, error) {
	inter, err := directCSIInterfaceForConfig(config, "DirectCSIDrive", "directcsidrives")
	if err != nil {
		return nil, err
	}

	return &directCSIDriveInterface{*inter}, nil
}

// Create takes the representation of a directCSIDrive and creates it.  Returns the server's representation of the directCSIDrive, and an error, if there is any.
func (d *directCSIDriveInterface) Create(ctx context.Context, directCSIDrive *directcsi.DirectCSIDrive, opts metav1.CreateOptions) (*directcsi.DirectCSIDrive, error) {
	directCSIDrive.TypeMeta = utils.DirectCSIDriveTypeMeta()
	unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(directCSIDrive)
	if err != nil {
		return nil, err
	}

	object, err := d.directCSIInterface.Create(ctx, unstructured, opts)
	if err != nil {
		return nil, err
	}

	return toDirectCSIDrive(object)
}

// Update takes the representation of a directCSIDrive and updates it. Returns the server's representation of the directCSIDrive, and an error, if there is any.
func (d *directCSIDriveInterface) Update(ctx context.Context, directCSIDrive *directcsi.DirectCSIDrive, opts metav1.UpdateOptions) (*directcsi.DirectCSIDrive, error) {
	directCSIDrive.TypeMeta = utils.DirectCSIDriveTypeMeta()
	unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(directCSIDrive)
	if err != nil {
		return nil, err
	}

	object, err := d.directCSIInterface.Update(ctx, unstructured, opts)
	if err != nil {
		return nil, err
	}

	return toDirectCSIDrive(object)
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (d *directCSIDriveInterface) UpdateStatus(ctx context.Context, directCSIDrive *directcsi.DirectCSIDrive, opts metav1.UpdateOptions) (*directcsi.DirectCSIDrive, error) {
	directCSIDrive.TypeMeta = utils.DirectCSIDriveTypeMeta()
	unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(directCSIDrive)
	if err != nil {
		return nil, err
	}

	object, err := d.directCSIInterface.UpdateStatus(ctx, unstructured, opts)
	if err != nil {
		return nil, err
	}

	return toDirectCSIDrive(object)
}

// Get takes name of the directCSIDrive, and returns the corresponding directCSIDrive object, and an error if there is any.
func (d *directCSIDriveInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*directcsi.DirectCSIDrive, error) {
	object, err := d.directCSIInterface.Get(ctx, name, opts)
	if err != nil {
		return nil, err
	}

	var drive directcsi.DirectCSIDrive
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(object, &drive); err != nil {
		return nil, err
	}
	return &drive, nil
}

// List takes label and field selectors, and returns the list of DirectCSIDrives that match those selectors.
func (d *directCSIDriveInterface) List(ctx context.Context, opts metav1.ListOptions) (*directcsi.DirectCSIDriveList, error) {
	object, items, err := d.directCSIInterface.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	var directCSIDriveList directcsi.DirectCSIDriveList
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(object, &directCSIDriveList)
	if err != nil {
		return nil, err
	}

	drives := []directcsi.DirectCSIDrive{}
	for i := range items {
		drive, err := toDirectCSIDrive(items[i])
		if err != nil {
			return nil, err
		}
		drives = append(drives, *drive)
	}
	directCSIDriveList.Items = drives

	return &directCSIDriveList, nil
}

// Patch applies the patch and returns the patched directCSIDrive.
func (d *directCSIDriveInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *directcsi.DirectCSIDrive, err error) {
	object, err := d.directCSIInterface.Patch(ctx, name, pt, data, opts, subresources...)
	if err != nil {
		return nil, err
	}

	return toDirectCSIDrive(object)
}

// directCSIVolumeInterface has methods to work with DirectCSIVolume resources.
type directCSIVolumeInterface struct {
	directCSIInterface
}

func directCSIVolumeInterfaceForConfig(config *rest.Config) (*directCSIVolumeInterface, error) {
	inter, err := directCSIInterfaceForConfig(config, "DirectCSIVolume", "directcsivolumes")
	if err != nil {
		return nil, err
	}

	return &directCSIVolumeInterface{*inter}, nil
}

// Create takes the representation of a directCSIVolume and creates it.  Returns the server's representation of the directCSIVolume, and an error, if there is any.
func (d *directCSIVolumeInterface) Create(ctx context.Context, directCSIVolume *directcsi.DirectCSIVolume, opts metav1.CreateOptions) (*directcsi.DirectCSIVolume, error) {
	directCSIVolume.TypeMeta = utils.DirectCSIVolumeTypeMeta()
	unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(directCSIVolume)
	if err != nil {
		return nil, err
	}

	object, err := d.directCSIInterface.Create(ctx, unstructured, opts)
	if err != nil {
		return nil, err
	}

	return toDirectCSIVolume(object)
}

// Update takes the representation of a directCSIVolume and updates it. Returns the server's representation of the directCSIVolume, and an error, if there is any.
func (d *directCSIVolumeInterface) Update(ctx context.Context, directCSIVolume *directcsi.DirectCSIVolume, opts metav1.UpdateOptions) (*directcsi.DirectCSIVolume, error) {
	directCSIVolume.TypeMeta = utils.DirectCSIVolumeTypeMeta()
	unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(directCSIVolume)
	if err != nil {
		return nil, err
	}

	object, err := d.directCSIInterface.Update(ctx, unstructured, opts)
	if err != nil {
		return nil, err
	}

	return toDirectCSIVolume(object)
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (d *directCSIVolumeInterface) UpdateStatus(ctx context.Context, directCSIVolume *directcsi.DirectCSIVolume, opts metav1.UpdateOptions) (*directcsi.DirectCSIVolume, error) {
	directCSIVolume.TypeMeta = utils.DirectCSIVolumeTypeMeta()
	unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(directCSIVolume)
	if err != nil {
		return nil, err
	}

	object, err := d.directCSIInterface.UpdateStatus(ctx, unstructured, opts)
	if err != nil {
		return nil, err
	}

	return toDirectCSIVolume(object)
}

// Get takes name of the directCSIVolume, and returns the corresponding directCSIVolume object, and an error if there is any.
func (d *directCSIVolumeInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*directcsi.DirectCSIVolume, error) {
	object, err := d.directCSIInterface.Get(ctx, name, opts)
	if err != nil {
		return nil, err
	}

	var volume directcsi.DirectCSIVolume
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(object, &volume); err != nil {
		return nil, err
	}
	return &volume, nil
}

// List takes label and field selectors, and returns the list of DirectCSIVolumes that match those selectors.
func (d *directCSIVolumeInterface) List(ctx context.Context, opts metav1.ListOptions) (*directcsi.DirectCSIVolumeList, error) {
	object, items, err := d.directCSIInterface.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	var directCSIVolumeList directcsi.DirectCSIVolumeList
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(object, &directCSIVolumeList)
	if err != nil {
		return nil, err
	}

	volumes := []directcsi.DirectCSIVolume{}
	for i := range items {
		volume, err := toDirectCSIVolume(items[i])
		if err != nil {
			return nil, err
		}
		volumes = append(volumes, *volume)
	}
	directCSIVolumeList.Items = volumes

	return &directCSIVolumeList, nil
}

// Patch applies the patch and returns the patched directCSIVolume.
func (d *directCSIVolumeInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *directcsi.DirectCSIVolume, err error) {
	object, err := d.directCSIInterface.Patch(ctx, name, pt, data, opts, subresources...)
	if err != nil {
		return nil, err
	}

	return toDirectCSIVolume(object)
}
