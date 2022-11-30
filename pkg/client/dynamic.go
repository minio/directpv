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

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/converter"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type dynamicInterface struct {
	resourceInterface dynamic.ResourceInterface
	groupVersion      schema.GroupVersion
}

func dynamicInterfaceForConfig(config *rest.Config, kind, resource string) (*dynamicInterface, error) {
	gvk, err := k8s.GetGroupVersionKind(consts.GroupName, kind, types.Versions...)
	if err != nil && !meta.IsNoMatchError(err) {
		return nil, err
	}
	resourceInterface, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	version := consts.LatestAPIVersion
	if gvk != nil {
		version = gvk.Version
	}
	group := consts.GroupName
	if gvk != nil {
		group = gvk.Group
	}
	return &dynamicInterface{
		resourceInterface: resourceInterface.Resource(
			schema.GroupVersionResource{
				Group:    consts.GroupName,
				Version:  version,
				Resource: resource,
			},
		),
		groupVersion: schema.GroupVersion{Group: group, Version: version},
	}, nil
}

// Create creates a resource object and returns server's representation of the object or an error on failure.
func (d *dynamicInterface) Create(ctx context.Context, object map[string]interface{}, opts metav1.CreateOptions) (map[string]interface{}, error) {
	unstructured := unstructured.Unstructured{Object: object}
	result, err := d.resourceInterface.Create(ctx, &unstructured, opts, "")
	if err != nil {
		return nil, err
	}
	return result.Object, nil
}

// Update updates a resource object and returns server's representation of the object or an error on failure.
func (d *dynamicInterface) Update(ctx context.Context, object map[string]interface{}, opts metav1.UpdateOptions) (map[string]interface{}, error) {
	unstructured := unstructured.Unstructured{Object: object}
	result, err := d.resourceInterface.Update(ctx, &unstructured, opts, "")
	if err != nil {
		return nil, err
	}
	return result.Object, nil
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (d *dynamicInterface) UpdateStatus(ctx context.Context, object map[string]interface{}, opts metav1.UpdateOptions) (map[string]interface{}, error) {
	unstructured := unstructured.Unstructured{Object: object}
	result, err := d.resourceInterface.UpdateStatus(ctx, &unstructured, opts)
	if err != nil {
		return nil, err
	}
	return result.Object, nil
}

// Delete deletes a resource object and returns an error on failure.
func (d *dynamicInterface) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return d.resourceInterface.Delete(ctx, name, opts, "")
}

// DeleteCollection deletes a collection of resource objects and returns an error on failure..
func (d *dynamicInterface) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return d.resourceInterface.DeleteCollection(ctx, opts, listOpts)
}

// Get returns a resource by name or an error on failure.
func (d *dynamicInterface) Get(ctx context.Context, name string, options metav1.GetOptions) (map[string]interface{}, error) {
	result, err := d.resourceInterface.Get(ctx, name, options, "")
	if err != nil {
		return nil, err
	}

	var migratedResult unstructured.Unstructured
	err = converter.Migrate(result, &migratedResult, schema.GroupVersion{Version: consts.LatestAPIVersion, Group: consts.GroupName})
	if err != nil {
		return nil, err
	}

	return migratedResult.Object, nil
}

// List returns list of resource object filtered by label and field selectors or an error on failure.
func (d *dynamicInterface) List(ctx context.Context, opts metav1.ListOptions) (map[string]interface{}, []map[string]interface{}, error) {
	result, err := d.resourceInterface.List(ctx, opts)
	if err != nil {
		return nil, nil, err
	}

	var migratedResult unstructured.UnstructuredList
	err = converter.MigrateList(result, &migratedResult, schema.GroupVersion{Version: consts.LatestAPIVersion, Group: consts.GroupName})
	if err != nil {
		return nil, nil, err
	}

	items := []map[string]interface{}{}
	for i := range migratedResult.Items {
		items = append(items, migratedResult.Items[i].Object)
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
			if v, ok := convertedObj["code"]; ok && v == int64(500) {
				break
			}
			intermediateResult := &unstructured.Unstructured{Object: convertedObj}
			finalResult := &unstructured.Unstructured{}
			if err := converter.Migrate(intermediateResult, finalResult, schema.GroupVersion{
				Version: consts.LatestAPIVersion,
				Group:   consts.GroupName,
			}); err != nil {
				klog.ErrorS(err, "unable to migrate to latest version", "version", consts.LatestAPIVersion)
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

// Watch returns a watch interface or an error on failure.
func (d *dynamicInterface) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	watcher, err := d.resourceInterface.Watch(ctx, opts)
	if err != nil {
		return nil, err
	}
	return watchInterfaceWrapper{watchInterface: watcher}, nil
}

// Patch patches a resource by name and returns patched resource object or an error on failure.
func (d *dynamicInterface) Patch(ctx context.Context, name string, pt apimachinerytypes.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (map[string]interface{}, error) {
	result, err := d.resourceInterface.Patch(ctx, name, pt, data, opts, subresources...)
	if err != nil {
		return nil, err
	}
	return result.Object, nil
}

// APIVersion returns the APIVersion this interface is expected to use.
func (d *dynamicInterface) APIVersion() schema.GroupVersion {
	return d.groupVersion
}

func toDrive(object map[string]interface{}) (*types.Drive, error) {
	var drive types.Drive
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object, &drive); err != nil {
		return nil, err
	}
	return &drive, nil
}

// latestDriveClient is a dynamic drive interface.
type latestDriveClient struct {
	dynamicInterface
}

// latestDriveClientForConfig creates new dynamic drive interface.
func latestDriveClientForConfig(config *rest.Config) (*latestDriveClient, error) {
	inter, err := dynamicInterfaceForConfig(config, consts.DriveKind, consts.DriveResource)
	if err != nil {
		return nil, err
	}

	return &latestDriveClient{*inter}, nil
}

// Create creates a drive and returns server's representation of the drive or an error on failure.
func (d *latestDriveClient) Create(ctx context.Context, drive *types.Drive, opts metav1.CreateOptions) (*types.Drive, error) {
	drive.TypeMeta = types.NewDriveTypeMeta()
	unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(drive)
	if err != nil {
		return nil, err
	}

	object, err := d.dynamicInterface.Create(ctx, unstructured, opts)
	if err != nil {
		return nil, err
	}

	return toDrive(object)
}

// Update updates a drive and returns server's representation of the drive or an error on failure.
func (d *latestDriveClient) Update(ctx context.Context, drive *types.Drive, opts metav1.UpdateOptions) (*types.Drive, error) {
	drive.TypeMeta = types.NewDriveTypeMeta()
	unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(drive)
	if err != nil {
		return nil, err
	}

	object, err := d.dynamicInterface.Update(ctx, unstructured, opts)
	if err != nil {
		return nil, err
	}

	return toDrive(object)
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (d *latestDriveClient) UpdateStatus(ctx context.Context, drive *types.Drive, opts metav1.UpdateOptions) (*types.Drive, error) {
	drive.TypeMeta = types.NewDriveTypeMeta()
	unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(drive)
	if err != nil {
		return nil, err
	}

	object, err := d.dynamicInterface.UpdateStatus(ctx, unstructured, opts)
	if err != nil {
		return nil, err
	}

	return toDrive(object)
}

// Get returns a drive by name or an error on failure.
func (d *latestDriveClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*types.Drive, error) {
	object, err := d.dynamicInterface.Get(ctx, name, opts)
	if err != nil {
		return nil, err
	}

	var drive types.Drive
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(object, &drive); err != nil {
		return nil, err
	}
	return &drive, nil
}

// List returns list of drive filtered by label and field selectors or an error on failure.
func (d *latestDriveClient) List(ctx context.Context, opts metav1.ListOptions) (*types.DriveList, error) {
	object, items, err := d.dynamicInterface.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	var driveList types.DriveList
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(object, &driveList)
	if err != nil {
		return nil, err
	}

	drives := []types.Drive{}
	for i := range items {
		drive, err := toDrive(items[i])
		if err != nil {
			return nil, err
		}
		drives = append(drives, *drive)
	}
	driveList.Items = drives

	return &driveList, nil
}

// Patch patches a drive by name and returns patched drive or an error on failure.
func (d *latestDriveClient) Patch(ctx context.Context, name string, pt apimachinerytypes.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *types.Drive, err error) {
	object, err := d.dynamicInterface.Patch(ctx, name, pt, data, opts, subresources...)
	if err != nil {
		return nil, err
	}

	return toDrive(object)
}

func toVolume(object map[string]interface{}) (*types.Volume, error) {
	var volume types.Volume
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object, &volume); err != nil {
		return nil, err
	}
	return &volume, nil
}

// latestVolumeClient is a dynamic volume interface.
type latestVolumeClient struct {
	dynamicInterface
}

// latestVolumeClientForConfig creates new dynamic volume interface.
func latestVolumeClientForConfig(config *rest.Config) (*latestVolumeClient, error) {
	inter, err := dynamicInterfaceForConfig(config, consts.VolumeKind, consts.VolumeResource)
	if err != nil {
		return nil, err
	}

	return &latestVolumeClient{*inter}, nil
}

// Create creates a volume and returns server's representation of the volume or an error on failure.
func (d *latestVolumeClient) Create(ctx context.Context, volume *types.Volume, opts metav1.CreateOptions) (*types.Volume, error) {
	volume.TypeMeta = types.NewVolumeTypeMeta()
	unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(volume)
	if err != nil {
		return nil, err
	}

	object, err := d.dynamicInterface.Create(ctx, unstructured, opts)
	if err != nil {
		return nil, err
	}

	return toVolume(object)
}

// Update updates a volume and returns server's representation of the volume or an error on failure.
func (d *latestVolumeClient) Update(ctx context.Context, volume *types.Volume, opts metav1.UpdateOptions) (*types.Volume, error) {
	volume.TypeMeta = types.NewVolumeTypeMeta()
	unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(volume)
	if err != nil {
		return nil, err
	}

	object, err := d.dynamicInterface.Update(ctx, unstructured, opts)
	if err != nil {
		return nil, err
	}

	return toVolume(object)
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (d *latestVolumeClient) UpdateStatus(ctx context.Context, volume *types.Volume, opts metav1.UpdateOptions) (*types.Volume, error) {
	volume.TypeMeta = types.NewVolumeTypeMeta()
	unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(volume)
	if err != nil {
		return nil, err
	}

	object, err := d.dynamicInterface.UpdateStatus(ctx, unstructured, opts)
	if err != nil {
		return nil, err
	}

	return toVolume(object)
}

// Get returns a volume by name or an error on failure.
func (d *latestVolumeClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*types.Volume, error) {
	object, err := d.dynamicInterface.Get(ctx, name, opts)
	if err != nil {
		return nil, err
	}

	var volume types.Volume
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(object, &volume); err != nil {
		return nil, err
	}
	return &volume, nil
}

// List returns list of volume filtered by label and field selectors or an error on failure.
func (d *latestVolumeClient) List(ctx context.Context, opts metav1.ListOptions) (*types.VolumeList, error) {
	object, items, err := d.dynamicInterface.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	var volumeList types.VolumeList
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(object, &volumeList)
	if err != nil {
		return nil, err
	}

	volumes := []types.Volume{}
	for i := range items {
		volume, err := toVolume(items[i])
		if err != nil {
			return nil, err
		}
		volumes = append(volumes, *volume)
	}
	volumeList.Items = volumes

	return &volumeList, nil
}

// Patch patches a volume by name and returns patched volume or an error on failure.
func (d *latestVolumeClient) Patch(ctx context.Context, name string, pt apimachinerytypes.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *types.Volume, err error) {
	object, err := d.dynamicInterface.Patch(ctx, name, pt, data, opts, subresources...)
	if err != nil {
		return nil, err
	}

	return toVolume(object)
}

func toNode(object map[string]interface{}) (*types.Node, error) {
	var node types.Node
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object, &node); err != nil {
		return nil, err
	}
	return &node, nil
}

// latestNodeClient is a dynamic node interface.
type latestNodeClient struct {
	dynamicInterface
}

// latestNodeClientForConfig creates new dynamic node interface.
func latestNodeClientForConfig(config *rest.Config) (*latestNodeClient, error) {
	inter, err := dynamicInterfaceForConfig(config, consts.NodeKind, consts.NodeResource)
	if err != nil {
		return nil, err
	}

	return &latestNodeClient{*inter}, nil
}

// Create creates a node and returns server's representation of the node or an error on failure.
func (n *latestNodeClient) Create(ctx context.Context, node *types.Node, opts metav1.CreateOptions) (*types.Node, error) {
	node.TypeMeta = types.NewNodeTypeMeta()
	unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(node)
	if err != nil {
		return nil, err
	}

	object, err := n.dynamicInterface.Create(ctx, unstructured, opts)
	if err != nil {
		return nil, err
	}

	return toNode(object)
}

// Update updates a node and returns server's representation of the node or an error on failure.
func (n *latestNodeClient) Update(ctx context.Context, node *types.Node, opts metav1.UpdateOptions) (*types.Node, error) {
	node.TypeMeta = types.NewNodeTypeMeta()
	unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(node)
	if err != nil {
		return nil, err
	}

	object, err := n.dynamicInterface.Update(ctx, unstructured, opts)
	if err != nil {
		return nil, err
	}

	return toNode(object)
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (n *latestNodeClient) UpdateStatus(ctx context.Context, node *types.Node, opts metav1.UpdateOptions) (*types.Node, error) {
	node.TypeMeta = types.NewNodeTypeMeta()
	unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(node)
	if err != nil {
		return nil, err
	}
	object, err := n.dynamicInterface.UpdateStatus(ctx, unstructured, opts)
	if err != nil {
		return nil, err
	}
	return toNode(object)
}

// Get returns a node by name or an error on failure.
func (n *latestNodeClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*types.Node, error) {
	object, err := n.dynamicInterface.Get(ctx, name, opts)
	if err != nil {
		return nil, err
	}

	var node types.Node
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(object, &node); err != nil {
		return nil, err
	}
	return &node, nil
}

// List returns list of node filtered by label and field selectors or an error on failure.
func (n *latestNodeClient) List(ctx context.Context, opts metav1.ListOptions) (*types.NodeList, error) {
	object, items, err := n.dynamicInterface.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	var nodeList types.NodeList
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(object, &nodeList)
	if err != nil {
		return nil, err
	}

	nodes := []types.Node{}
	for i := range items {
		node, err := toNode(items[i])
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, *node)
	}
	nodeList.Items = nodes

	return &nodeList, nil
}

// Watch returns a watch interface or an error on failure.
func (n *latestNodeClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return n.dynamicInterface.Watch(ctx, opts)
}

// Patch patches a node by name and returns patched node or an error on failure.
func (n *latestNodeClient) Patch(ctx context.Context, name string, pt apimachinerytypes.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *types.Node, err error) {
	object, err := n.dynamicInterface.Patch(ctx, name, pt, data, opts, subresources...)
	if err != nil {
		return nil, err
	}
	return toNode(object)
}

func toInitRequest(object map[string]interface{}) (*types.InitRequest, error) {
	var initRequest types.InitRequest
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object, &initRequest); err != nil {
		return nil, err
	}
	return &initRequest, nil
}

// latestInitRequestClient is a dynamic initrequest interface.
type latestInitRequestClient struct {
	dynamicInterface
}

// latestInitRequestClientForConfig creates new dynamic initrequest interface.
func latestInitRequestClientForConfig(config *rest.Config) (*latestInitRequestClient, error) {
	inter, err := dynamicInterfaceForConfig(config, consts.InitRequestKind, consts.InitRequestResource)
	if err != nil {
		return nil, err
	}

	return &latestInitRequestClient{*inter}, nil
}

// Create creates a initrequest and returns server's representation of the initrequest or an error on failure.
func (r *latestInitRequestClient) Create(ctx context.Context, initRequest *types.InitRequest, opts metav1.CreateOptions) (*types.InitRequest, error) {
	initRequest.TypeMeta = types.NewInitRequestTypeMeta()
	unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(initRequest)
	if err != nil {
		return nil, err
	}

	object, err := r.dynamicInterface.Create(ctx, unstructured, opts)
	if err != nil {
		return nil, err
	}

	return toInitRequest(object)
}

// Update updates a initrequest and returns server's representation of the initrequest or an error on failure.
func (r *latestInitRequestClient) Update(ctx context.Context, initRequest *types.InitRequest, opts metav1.UpdateOptions) (*types.InitRequest, error) {
	initRequest.TypeMeta = types.NewInitRequestTypeMeta()
	unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(initRequest)
	if err != nil {
		return nil, err
	}
	object, err := r.dynamicInterface.Update(ctx, unstructured, opts)
	if err != nil {
		return nil, err
	}
	return toInitRequest(object)
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (r *latestInitRequestClient) UpdateStatus(ctx context.Context, initRequest *types.InitRequest, opts metav1.UpdateOptions) (*types.InitRequest, error) {
	initRequest.TypeMeta = types.NewInitRequestTypeMeta()
	unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(initRequest)
	if err != nil {
		return nil, err
	}
	object, err := r.dynamicInterface.UpdateStatus(ctx, unstructured, opts)
	if err != nil {
		return nil, err
	}
	return toInitRequest(object)
}

// Get returns a initrequest by name or an error on failure.
func (r *latestInitRequestClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*types.InitRequest, error) {
	object, err := r.dynamicInterface.Get(ctx, name, opts)
	if err != nil {
		return nil, err
	}
	var initRequest types.InitRequest
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(object, &initRequest); err != nil {
		return nil, err
	}
	return &initRequest, nil
}

// List returns list of initrequest filtered by label and field selectors or an error on failure.
func (r *latestInitRequestClient) List(ctx context.Context, opts metav1.ListOptions) (*types.InitRequestList, error) {
	object, items, err := r.dynamicInterface.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	var initRequestList types.InitRequestList
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(object, &initRequestList)
	if err != nil {
		return nil, err
	}

	initRequests := []types.InitRequest{}
	for i := range items {
		initRequest, err := toInitRequest(items[i])
		if err != nil {
			return nil, err
		}
		initRequests = append(initRequests, *initRequest)
	}
	initRequestList.Items = initRequests

	return &initRequestList, nil
}

// Patch patches a initrequest by name and returns patched initrequest or an error on failure.
func (r *latestInitRequestClient) Patch(ctx context.Context, name string, pt apimachinerytypes.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *types.InitRequest, err error) {
	object, err := r.dynamicInterface.Patch(ctx, name, pt, data, opts, subresources...)
	if err != nil {
		return nil, err
	}
	return toInitRequest(object)
}
