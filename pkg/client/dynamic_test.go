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
	"fmt"
	"reflect"
	"testing"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	fakedynamic "k8s.io/client-go/dynamic/fake"
)

const (
	MiB            = 1024 * 1024
	testNodeName   = consts.GroupName + "/node"
	testDrivePath  = consts.GroupName + "/drive-path"
	testTenantName = "tenant-1"
	tenantLabel    = consts.GroupName + "/tenant"
)

var expectedGV = schema.GroupVersion{Group: consts.GroupName, Version: consts.LatestAPIVersion}

func newFakeDynamicInterface(backendVersion, resource string, object *unstructured.Unstructured) dynamicInterface {
	return dynamicInterface{
		resourceInterface: fakedynamic.NewSimpleDynamicClient(
			runtime.NewScheme(),
			object,
		).Resource(
			schema.GroupVersionResource{
				Group:    consts.GroupName,
				Version:  backendVersion,
				Resource: resource,
			},
		),
		groupVersion: schema.GroupVersion{Group: consts.GroupName, Version: backendVersion},
	}
}

func newFakeLatestDriveClient(backendVersion, resource string, object *unstructured.Unstructured) *latestDriveClient {
	return &latestDriveClient{newFakeDynamicInterface(backendVersion, resource, object)}
}

func newFakeLatestVolumeClient(backendVersion, resource string, object *unstructured.Unstructured) *latestVolumeClient {
	return &latestVolumeClient{newFakeDynamicInterface(backendVersion, resource, object)}
}

func newFakeDynamicInterfaceForList(backendVersion, resource, value string, unstructuredObjects []*unstructured.Unstructured) dynamicInterface {
	toRuntimeObjects := func(objs []*unstructured.Unstructured) (objects []runtime.Object) {
		for _, unObj := range objs {
			objects = append(objects, unObj)
		}
		return objects
	}

	return dynamicInterface{
		resourceInterface: fakedynamic.NewSimpleDynamicClientWithCustomListKinds(
			runtime.NewScheme(),
			map[schema.GroupVersionResource]string{
				{Group: consts.GroupName, Version: backendVersion, Resource: resource}: value,
			},
			toRuntimeObjects(unstructuredObjects)...,
		).Resource(
			schema.GroupVersionResource{
				Group:    consts.GroupName,
				Version:  backendVersion,
				Resource: resource,
			},
		),
		groupVersion: schema.GroupVersion{Group: consts.GroupName, Version: backendVersion},
	}
}

func newFakeLatestDriveClientForList(backendVersion, resource, value string, unstructuredObjects ...*unstructured.Unstructured) *latestDriveClient {
	return &latestDriveClient{newFakeDynamicInterfaceForList(backendVersion, resource, value, unstructuredObjects)}
}

func newFakeLatestVolumeClientForList(backendVersion, resource, value string, unstructuredObjects ...*unstructured.Unstructured) *latestVolumeClient {
	return &latestVolumeClient{newFakeDynamicInterfaceForList(backendVersion, resource, value, unstructuredObjects)}
}

func createTestDrive(node, drive, backendVersion string, labels map[string]string) *types.Drive {
	return &types.Drive{
		TypeMeta: metav1.TypeMeta{
			APIVersion: consts.GroupName + "/" + backendVersion,
			Kind:       consts.DriveKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: drive,
			Finalizers: []string{
				string(consts.DriveFinalizerDataProtection),
			},
			Labels: labels,
		},
		Status: types.DriveStatus{
			NodeName:          node,
			Status:            directpvtypes.DriveStatusOK,
			FreeCapacity:      20 * MiB,
			AllocatedCapacity: 0,
			TotalCapacity:     20 * MiB,
		},
	}
}

func getFakeLatestDriveClient(t *testing.T, i int, drive runtime.Object, version string) *latestDriveClient {
	unstructuredObject, err := runtime.DefaultUnstructuredConverter.ToUnstructured(drive)
	if err != nil {
		t.Fatalf("case %v: unexpected error: %v", i+1, err)
	}
	return newFakeLatestDriveClient(version, consts.DriveResource, &unstructured.Unstructured{Object: unstructuredObject})
}

func TestGetDrive(t *testing.T) {
	testCases := []struct {
		apiVersion string
		name       string
		object     runtime.Object
	}{
		{
			apiVersion: consts.LatestAPIVersion,
			name:       "drive1",
			object:     createTestDrive("node1", "drive1", consts.LatestAPIVersion, map[string]string{}),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	for i, testCase := range testCases {
		client := getFakeLatestDriveClient(t, i, testCase.object, testCase.apiVersion)
		drive, err := client.Get(ctx, testCase.name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}
		if expectedGV != drive.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: groupVersion: expected: %v, got: %v", i+1, expectedGV, drive.GetObjectKind().GroupVersionKind().GroupVersion())
		}
	}
}

func TestListDrive(t *testing.T) {
	testCases := []struct {
		apiVersion string
		name       string
		objects    []runtime.Object
	}{
		{
			apiVersion: consts.LatestAPIVersion,
			name:       "drive1",
			objects: []runtime.Object{
				createTestDrive("node1", "drive1", consts.LatestAPIVersion, map[string]string{}),
				createTestDrive("node1", "drive2", consts.LatestAPIVersion, map[string]string{}),
			},
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	var unstructuredObjects []*unstructured.Unstructured
	for i, testCase := range testCases {
		for j := range testCase.objects {
			obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(testCase.objects[j])
			if err != nil {
				t.Fatalf("case %v: object %v: unexpected error: %v", i+1, j, err)
			}
			unstructuredObjects = append(unstructuredObjects, &unstructured.Unstructured{Object: obj})
		}

		client := newFakeLatestDriveClientForList(testCase.apiVersion, consts.DriveResource, "DirectPVDriveList", unstructuredObjects...)
		driveList, err := client.List(ctx, metav1.ListOptions{})
		if err != nil {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}
		if expectedGV != driveList.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: groupVersion: expected: %v, got: %v", i+1, expectedGV, driveList.GetObjectKind().GroupVersionKind().GroupVersion())
		}
	}
}

func TestListDriveWithOption(t *testing.T) {
	testCases := []struct {
		apiVersion string
		names      []string
		objects    []runtime.Object
	}{
		{
			apiVersion: consts.LatestAPIVersion,
			names:      []string{"drive2", "drive3"},
			objects: []runtime.Object{
				createTestDrive("node1", "drive1", consts.LatestAPIVersion,
					map[string]string{string(types.NodeLabelKey): "node1", string(types.AccessTierLabelKey): "Hot"}),
				createTestDrive("node2", "drive2", consts.LatestAPIVersion,
					map[string]string{string(types.NodeLabelKey): "node2", string(types.AccessTierLabelKey): "Hot"}),
				createTestDrive("node2", "drive3", consts.LatestAPIVersion,
					map[string]string{string(types.NodeLabelKey): "node2", string(types.AccessTierLabelKey): "Hot"}),
			},
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	var unstructuredObjects []*unstructured.Unstructured
	for i, testCase := range testCases {
		for j := range testCase.objects {
			obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(testCase.objects[j])
			if err != nil {
				t.Fatalf("case %v: object %v: unexpected error: %v", i+1, j, err)
			}
			unstructuredObjects = append(unstructuredObjects, &unstructured.Unstructured{Object: obj})
		}

		client := newFakeLatestDriveClientForList(testCase.apiVersion, consts.DriveResource, "DirectPVDriveList", unstructuredObjects...)
		labelMap := map[types.LabelKey][]types.LabelValue{
			types.NodeLabelKey: {types.NewLabelValue("node2")},
		}
		if testCase.apiVersion == consts.LatestAPIVersion {
			labelMap[types.AccessTierLabelKey] = []types.LabelValue{types.NewLabelValue("Hot")}
		}
		driveList, err := client.List(ctx, metav1.ListOptions{LabelSelector: types.ToLabelSelector(labelMap)})
		if err != nil {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}
		var names []string
		for _, item := range driveList.Items {
			names = append(names, item.Name)
		}
		if expectedGV != driveList.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: groupVersion: expected: %v, got: %v", i+1, expectedGV, driveList.GetObjectKind().GroupVersionKind().GroupVersion())
		}
		if !reflect.DeepEqual(names, testCase.names) {
			t.Fatalf("case %v: names: expected: %v, got: %v", i+1, testCase.names, names)
		}
	}
}

func TestCreateDrive(t *testing.T) {
	testCases := []struct {
		apiVersion   string
		name         string
		newDriveName string
		inputDrive   *types.Drive
		newDrive     *types.Drive
	}{
		{
			apiVersion: consts.LatestAPIVersion,
			name:       "drive3",
			inputDrive: createTestDrive("node1", "drive3", consts.LatestAPIVersion, map[string]string{string(types.NodeLabelKey): "node1"}),
			newDrive:   createTestDrive("node1", "new-drive3", consts.LatestAPIVersion, map[string]string{string(types.NodeLabelKey): "node1"}),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		client := getFakeLatestDriveClient(t, i, testCase.inputDrive, testCase.apiVersion)
		createdDrive, err := client.Create(ctx, testCase.newDrive, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}
		if expectedGV != createdDrive.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: groupVersion: expected: %v, got: %v", i+1, expectedGV, createdDrive.GetObjectKind().GroupVersionKind().GroupVersion())
		}
	}
}

func TestDeleteDrive(t *testing.T) {
	testCases := []struct {
		apiVersion string
		name       string
		inputDrive runtime.Object
	}{
		{
			apiVersion: consts.LatestAPIVersion,
			name:       "drive2",
			inputDrive: createTestDrive("node1", "drive2", consts.LatestAPIVersion, map[string]string{string(types.NodeLabelKey): "node1"}),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		client := getFakeLatestDriveClient(t, i, testCase.inputDrive, testCase.apiVersion)
		if err := client.Delete(ctx, testCase.name, metav1.DeleteOptions{}); err != nil {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}
		if _, err := client.Get(ctx, testCase.name, metav1.GetOptions{}); err != nil && !apierrors.IsNotFound(err) {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}
	}
}

func TestDeleteCollectionDrive(t *testing.T) {
	testCases := []struct {
		apiVersion string
		name       string
		inputDrive runtime.Object
	}{
		{
			apiVersion: consts.LatestAPIVersion,
			name:       "drive2",
			inputDrive: createTestDrive("node1", "drive2", consts.LatestAPIVersion, map[string]string{string(types.NodeLabelKey): "node1"}),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		client := getFakeLatestDriveClient(t, i, testCase.inputDrive, testCase.apiVersion)
		if err := client.DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{}); err != nil {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}
		if _, err := client.Get(ctx, testCase.name, metav1.GetOptions{}); err != nil && !apierrors.IsNotFound(err) {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}
	}
}

func TestUpdateDrive(t *testing.T) {
	testCases := []struct {
		apiVersion string
		name       string
		accessTier directpvtypes.AccessTier
		inputDrive *types.Drive
	}{
		{
			apiVersion: consts.LatestAPIVersion,
			name:       "drive2",
			accessTier: directpvtypes.AccessTierHot,
			inputDrive: createTestDrive("node1", "drive2", consts.LatestAPIVersion, map[string]string{string(types.NodeLabelKey): "node1"}),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		client := getFakeLatestDriveClient(t, i, testCase.inputDrive, testCase.apiVersion)
		testCase.inputDrive.Status.AccessTier = testCase.accessTier
		updatedDrive, err := client.Update(ctx, testCase.inputDrive, metav1.UpdateOptions{})
		if err != nil {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}
		if expectedGV != updatedDrive.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: groupVersion: expected: %v, got: %v", i+1, expectedGV, updatedDrive.GetObjectKind().GroupVersionKind().GroupVersion())
		}
		if updatedDrive.Status.AccessTier != testCase.accessTier {
			t.Fatalf("case %v: accessTier: expected: %v, got: %v", i+1, updatedDrive.Status.AccessTier, testCase.accessTier)
		}
	}
}

func TestUpdateStatusDrive(t *testing.T) {
	testCases := []struct {
		apiVersion string
		name       string
		accessTier directpvtypes.AccessTier
		inputDrive *types.Drive
	}{
		{
			apiVersion: consts.LatestAPIVersion,
			name:       "drive2",
			accessTier: directpvtypes.AccessTierHot,
			inputDrive: createTestDrive("node1", "drive2", consts.LatestAPIVersion, map[string]string{string(types.NodeLabelKey): "node1"}),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		client := getFakeLatestDriveClient(t, i, testCase.inputDrive, testCase.apiVersion)
		testCase.inputDrive.Status.AccessTier = testCase.accessTier
		updatedDrive, err := client.UpdateStatus(ctx, testCase.inputDrive, metav1.UpdateOptions{})
		if err != nil {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}
		if expectedGV != updatedDrive.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: groupVersion: expected: %v, got: %v", i+1, expectedGV, updatedDrive.GetObjectKind().GroupVersionKind().GroupVersion())
		}
		if updatedDrive.Status.AccessTier != testCase.accessTier {
			t.Fatalf("case %v: accessTier: expected: %v, got: %v", i+1, updatedDrive.Status.AccessTier, testCase.accessTier)
		}
	}
}

func TestWatcher(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	checkEvent := func(event watch.Event, expectedEventType watch.EventType, expectedObject *types.Drive) error {
		if event.Type != expectedEventType {
			return fmt.Errorf("eventType: expected: %v, got: %v", expectedEventType, event.Type)
		}
		var drive types.Drive
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Object.(*unstructured.Unstructured).Object, &drive); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(&drive, expectedObject) {
			return fmt.Errorf("eventType %v: expected: %v, got: %v", expectedEventType, expectedObject, drive)
		}
		return nil
	}

	inputDrive := createTestDrive("node1", "drive1", consts.LatestAPIVersion, map[string]string{string(types.NodeLabelKey): "node1"})
	fakeDriveClient := getFakeLatestDriveClient(t, 0, inputDrive, consts.LatestAPIVersion)
	fakeWatchInterface, err := fakeDriveClient.Watch(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resultCh := watchInterfaceWrapper{fakeWatchInterface}.ResultChan()

	// Test create event
	testCreateObject := &types.Drive{
		TypeMeta: types.NewDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "drive2",
		},
	}
	if _, err = fakeDriveClient.Create(ctx, testCreateObject, metav1.CreateOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	event, ok := <-resultCh
	if !ok {
		t.Fatalf("no event received for create")
	}
	if err := checkEvent(event, watch.Added, testCreateObject); err != nil {
		t.Fatalf(err.Error())
	}

	// Test Update Event
	testUpdateObject := &types.Drive{
		TypeMeta: types.NewDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "drive1",
		},
		Status: types.DriveStatus{Status: directpvtypes.DriveStatusError},
	}
	if _, err = fakeDriveClient.Update(ctx, testUpdateObject, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	event, ok = <-resultCh
	if !ok {
		t.Fatalf("no event received for update")
	}
	if err := checkEvent(event, watch.Modified, testUpdateObject); err != nil {
		t.Fatalf(err.Error())
	}

	// Test Delete Event

	if err = fakeDriveClient.Delete(ctx, "drive1", metav1.DeleteOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	event, ok = <-resultCh
	if !ok {
		t.Fatalf("no event received for delete")
	}
	if err := checkEvent(event, watch.Deleted, testUpdateObject); err != nil {
		t.Fatalf(err.Error())
	}
}

func createTestVolume(volName string, apiVersion string) *types.Volume {
	return &types.Volume{
		TypeMeta: metav1.TypeMeta{
			APIVersion: string(consts.GroupName + "/" + apiVersion),
			Kind:       consts.VolumeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: volName,
			Labels: map[string]string{
				tenantLabel: testTenantName,
			},
		},
		Status: types.VolumeStatus{
			NodeName:   testNodeName,
			DriveName:  testDrivePath,
			TargetPath: "/path/targetpath",
		},
	}
}

func getFakeLatestVolumeClient(drive runtime.Object, i int, version string, t *testing.T) *latestVolumeClient {
	unstructuredObject, err := runtime.DefaultUnstructuredConverter.ToUnstructured(drive)
	if err != nil {
		t.Errorf("case %v: Error in converting to unstructured object, Expected err to nil, got %v", i+1, err)
	}
	return newFakeLatestVolumeClient(version, consts.VolumeResource, &unstructured.Unstructured{Object: unstructuredObject})
}

func TestGetVolume(t *testing.T) {
	testCases := []struct {
		apiVersion string
		name       string
		volume     runtime.Object
	}{
		{
			apiVersion: consts.LatestAPIVersion,
			name:       "volume2",
			volume:     createTestVolume("volume2", consts.LatestAPIVersion),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		volumeClient := getFakeLatestVolumeClient(testCase.volume, i, testCase.apiVersion, t)
		volume, err := volumeClient.Get(ctx, testCase.name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}
		if expectedGV != volume.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: groupVersion: expected: %v, got: %v", i+1, expectedGV, volume.GetObjectKind().GroupVersionKind().GroupVersion())
		}
	}
}

func TestListVolume(t *testing.T) {
	testCases := []struct {
		apiVersion string
		name       string
		volumes    []runtime.Object
	}{
		{
			apiVersion: consts.LatestAPIVersion,
			name:       "drive2",
			volumes:    []runtime.Object{createTestVolume("volume1", consts.LatestAPIVersion), createTestVolume("volume2", consts.LatestAPIVersion)},
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	var unstructuredObjects []*unstructured.Unstructured
	for i, testCase := range testCases {
		for j, value := range testCase.volumes {
			obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(value)
			if err != nil {
				t.Fatalf("case %v: volume %v: unexpected error: %v", i+1, j, err)
			}
			unstructuredObjects = append(unstructuredObjects, &unstructured.Unstructured{Object: obj})
		}

		client := newFakeLatestVolumeClientForList(testCase.apiVersion, consts.DriveResource, "DirectPVVolumeList", unstructuredObjects...)
		volumeList, err := client.List(ctx, metav1.ListOptions{})
		if err != nil {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}
		if expectedGV != volumeList.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: groupVersion: expected: %v, got: %v", i+1, expectedGV, volumeList.GetObjectKind().GroupVersionKind().GroupVersion())
		}
	}
}

func TestCreateVolume(t *testing.T) {
	testCases := []struct {
		apiVersion string
		name       string
		volume     *types.Volume
		newVolume  *types.Volume
	}{
		{
			apiVersion: consts.LatestAPIVersion,
			name:       "volume2",
			volume:     createTestVolume("volume2", consts.LatestAPIVersion),
			newVolume:  createTestVolume("new-volume2", consts.LatestAPIVersion),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		volumeClient := getFakeLatestVolumeClient(testCase.volume, i, testCase.apiVersion, t)
		createdVolume, err := volumeClient.Create(ctx, testCase.newVolume, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}
		if expectedGV != createdVolume.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: groupVersion: expected: %v, got: %v", i+1, expectedGV, createdVolume.GetObjectKind().GroupVersionKind().GroupVersion())
		}
	}
}

func TestDeleteVolume(t *testing.T) {
	testCases := []struct {
		apiVersion string
		name       string
		volume     runtime.Object
	}{
		{
			apiVersion: consts.LatestAPIVersion,
			name:       "volume2",
			volume:     createTestVolume("volume2", consts.LatestAPIVersion),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		volumeClient := getFakeLatestVolumeClient(testCase.volume, i, testCase.apiVersion, t)
		if err := volumeClient.Delete(ctx, testCase.name, metav1.DeleteOptions{}); err != nil {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}
		if _, err := volumeClient.Get(ctx, testCase.name, metav1.GetOptions{}); err != nil && !apierrors.IsNotFound(err) {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}
	}
}

func TestVolumeDeleteCollection(t *testing.T) {
	testCases := []struct {
		apiVersion string
		name       string
		volume     runtime.Object
	}{
		{
			apiVersion: consts.LatestAPIVersion,
			name:       "volume2",
			volume:     createTestVolume("volume2", consts.LatestAPIVersion),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		volumeClient := getFakeLatestVolumeClient(testCase.volume, i, testCase.apiVersion, t)
		if err := volumeClient.DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{}); err != nil {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}
		if _, err := volumeClient.Get(ctx, testCase.name, metav1.GetOptions{}); err != nil && !apierrors.IsNotFound(err) {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}
	}
}

func TestUpdateVolume(t *testing.T) {
	testCases := []struct {
		apiVersion        string
		name              string
		availableCapacity int64
		volume            *types.Volume
	}{
		{
			apiVersion:        consts.LatestAPIVersion,
			name:              "volume2",
			availableCapacity: 20 * MiB,
			volume:            createTestVolume("volume2", consts.LatestAPIVersion),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		volumeClient := getFakeLatestVolumeClient(testCase.volume, i, testCase.apiVersion, t)
		testCase.volume.Status.AvailableCapacity = testCase.availableCapacity
		updatedVolume, err := volumeClient.Update(ctx, testCase.volume, metav1.UpdateOptions{})
		if err != nil {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}
		if expectedGV != updatedVolume.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: groupVersion: expected: %v, got: %v", i+1, expectedGV, updatedVolume.GetObjectKind().GroupVersionKind().GroupVersion())
		}
		if updatedVolume.Status.AvailableCapacity != testCase.availableCapacity {
			t.Fatalf("case %v: availableCapacity: expected: %v, got: %v", i+1, updatedVolume.Status.AvailableCapacity, testCase.availableCapacity)
		}
	}
}

func TestUpdateStatusVolume(t *testing.T) {
	testCases := []struct {
		apiVersion        string
		name              string
		availableCapacity int64
		volume            *types.Volume
	}{
		{
			apiVersion:        consts.LatestAPIVersion,
			name:              "volume2",
			availableCapacity: 20 * MiB,
			volume:            createTestVolume("volume2", consts.LatestAPIVersion),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		volumeClient := getFakeLatestVolumeClient(testCase.volume, i, testCase.apiVersion, t)
		testCase.volume.Status.AvailableCapacity = testCase.availableCapacity
		updatedVolume, err := volumeClient.UpdateStatus(ctx, testCase.volume, metav1.UpdateOptions{})
		if err != nil {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}
		if expectedGV != updatedVolume.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: groupVersion: expected: %v, got: %v", i+1, expectedGV, updatedVolume.GetObjectKind().GroupVersionKind().GroupVersion())
		}
		if updatedVolume.Status.AvailableCapacity != testCase.availableCapacity {
			t.Fatalf("case %v: availableCapacity: expected: %v, got: %v", i+1, updatedVolume.Status.AvailableCapacity, testCase.availableCapacity)
		}
	}
}
