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

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/utils"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	fakedynamic "k8s.io/client-go/dynamic/fake"
)

const (
	mb20           = 20 * 1024 * 1024
	testNodeName   = "direct.csi.min.io/node"
	testDrivePath  = "direct.csi.min.io/drive-path"
	testTenantName = "tenant-1"
	tenantLabel    = "direct.csi.min.io/tenant"
)

func getFakeDirectCSIAdapter(backendVersion, resource string, object *unstructured.Unstructured) (*directCSIInterface, error) {
	fakeResourceInterface := fakedynamic.NewSimpleDynamicClient(
		runtime.NewScheme(),
		object,
	)
	dynamicResourceClient := fakeResourceInterface.Resource(
		schema.GroupVersionResource{
			Group:    directcsi.Group,
			Version:  backendVersion,
			Resource: resource,
		},
	)
	groupVersion := schema.GroupVersion{Group: directcsi.Group, Version: backendVersion}
	return &directCSIInterface{resourceInterface: dynamicResourceClient, groupVersion: groupVersion}, nil
}

func getFakeDirectCSIDriveListAdapter(backendVersion, resource, value string, unstructuredObjects ...*unstructured.Unstructured) (*directCSIInterface, error) {
	toRuntimeObjects := func(objs []*unstructured.Unstructured) (objects []runtime.Object) {
		for _, unObj := range objs {
			objects = append(objects, unObj)
		}
		return objects
	}
	fakeResourceInterface := fakedynamic.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			{Group: directcsi.Group, Version: backendVersion, Resource: resource}: value,
		},
		toRuntimeObjects(unstructuredObjects)...,
	)
	dynamicResourceClient := fakeResourceInterface.Resource(
		schema.GroupVersionResource{
			Group:    directcsi.Group,
			Version:  backendVersion,
			Resource: resource,
		},
	)
	groupVersion := schema.GroupVersion{Group: directcsi.Group, Version: backendVersion}
	return &directCSIInterface{resourceInterface: dynamicResourceClient, groupVersion: groupVersion}, nil
}

func createTestDrive(node, drive, backendVersion string, labels map[string]string) *directcsi.DirectCSIDrive {
	return &directcsi.DirectCSIDrive{
		TypeMeta: metav1.TypeMeta{
			APIVersion: string(directcsi.Group + "/" + backendVersion),
			Kind:       "DirectCSIDrive",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: drive,
			Finalizers: []string{
				string(directcsi.DirectCSIDriveFinalizerDataProtection),
			},
			Labels: labels,
		},
		Status: directcsi.DirectCSIDriveStatus{
			NodeName:          node,
			Filesystem:        "xfs",
			DriveStatus:       directcsi.DriveStatusReady,
			FreeCapacity:      mb20,
			AllocatedCapacity: int64(0),
			TotalCapacity:     mb20,
		},
	}
}

func getFakeDirectCSIDriveAdapter(drive runtime.Object, i int, version string, t *testing.T) *directCSIDriveInterface {
	unstructuredObject, err := runtime.DefaultUnstructuredConverter.ToUnstructured(drive)
	if err != nil {
		t.Errorf("case %v: Error in converting to unstructured object, Expected err to nil, got %v", i+1, err)
	}
	fakeDirecCSIClient, err := getFakeDirectCSIAdapter(version, "directcsidrives", &unstructured.Unstructured{Object: unstructuredObject})
	if err != nil {
		t.Errorf("case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
	}
	return &directCSIDriveInterface{*fakeDirecCSIClient}
}

func TestGetDrive(t *testing.T) {
	testCases := []struct {
		backendVersion string
		name           string
		inputDrive     runtime.Object
	}{
		{
			backendVersion: "v1alpha1",
			name:           "D1",
			inputDrive:     createTestDrive("N1", "D1", "v1alpha1", map[string]string{}),
		},
		{
			backendVersion: "v1beta1",
			name:           "D2",
			inputDrive:     createTestDrive("N1", "D2", "v1beta1", map[string]string{}),
		},
		{
			backendVersion: "v1beta2",
			name:           "D3",
			inputDrive:     createTestDrive("N1", "D3", "v1beta2", map[string]string{string(utils.NodeLabelKey): "N2"}),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	for i, testCase := range testCases {
		fakeDirectCSIDriveAdapter := getFakeDirectCSIDriveAdapter(testCase.inputDrive, i, testCase.backendVersion, t)
		directCSIDrive, err := fakeDirectCSIDriveAdapter.Get(ctx, testCase.name, metav1.GetOptions{})
		if err != nil {
			t.Errorf("case %v: Error in Get, Expected  err to nil, got %v", i+1, err)
		}
		expectedGV := schema.GroupVersion{Group: directcsi.Group, Version: directcsi.Version}
		if expectedGV != directCSIDrive.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: result: expected: %v, got: %v", i+1, expectedGV, directCSIDrive.GetObjectKind().GroupVersionKind().GroupVersion())
		}
	}
}

func TestListDrive(t *testing.T) {
	testCases := []struct {
		backendVersion string
		name           string
		inputDrives    []runtime.Object
	}{
		{
			backendVersion: "v1alpha1",
			name:           "D1",
			inputDrives:    []runtime.Object{createTestDrive("N1", "D1", "v1alpha1", map[string]string{}), createTestDrive("N1", "A1", "v1alpha1", map[string]string{})},
		},
		{
			backendVersion: "v1beta1",
			name:           "D2",
			inputDrives:    []runtime.Object{createTestDrive("N1", "D1", "v1beta1", map[string]string{}), createTestDrive("N1", "A1", "v1beta1", map[string]string{})},
		},
		{
			backendVersion: "v1beta2",
			name:           "D3",
			inputDrives:    []runtime.Object{createTestDrive("N1", "D1", "v1beta2", map[string]string{string(utils.NodeLabelKey): "N2"}), createTestDrive("N1", "A1", "v1beta2", map[string]string{string(utils.NodeLabelKey): "N2"})},
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	var unstructuredObjects []*unstructured.Unstructured
	for i, testCase := range testCases {
		for i := range testCase.inputDrives {
			obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(testCase.inputDrives[i])
			if err != nil {
				t.Errorf("case %v: Error in converting to unstructured object, Expected err to nil, got %v", i+1, err)
			}
			unstructuredObjects = append(unstructuredObjects, &unstructured.Unstructured{Object: obj})
		}

		fakeDirecCSIClient, err := getFakeDirectCSIDriveListAdapter(testCase.backendVersion, "directcsidrives", "DirectCSIDriveList", unstructuredObjects...)
		if err != nil {
			t.Errorf(" case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
		}
		fakeDirectCSIListAdapter := &directCSIDriveInterface{*fakeDirecCSIClient}
		driveList, err := fakeDirectCSIListAdapter.List(ctx, metav1.ListOptions{})
		if err != nil {
			t.Errorf("case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
		}
		expectedGV := schema.GroupVersion{Group: directcsi.Group, Version: directcsi.Version}
		if expectedGV != driveList.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: result: expected: %v, got: %v", i+1, expectedGV, driveList.GetObjectKind().GroupVersionKind().GroupVersion())
		}
	}
}

func TestListDriveWithOption(t *testing.T) {
	testCases := []struct {
		backendVersion string
		names          []string
		inputDrives    []runtime.Object
	}{
		{
			backendVersion: "v1alpha1",
			names:          []string{"D2", "D3"},
			inputDrives: []runtime.Object{
				createTestDrive("N1", "D1", "v1alpha1", map[string]string{}),
				createTestDrive("N2", "D2", "v1alpha1", map[string]string{}),
				createTestDrive("N2", "D3", "v1alpha1", map[string]string{}),
			},
		},
		{
			backendVersion: "v1beta1",
			names:          []string{"D2", "D3"},
			inputDrives: []runtime.Object{
				createTestDrive("N1", "D1", "v1beta1", map[string]string{}),
				createTestDrive("N2", "D2", "v1beta1", map[string]string{}),
				createTestDrive("N2", "D3", "v1beta1", map[string]string{}),
			},
		},
		{
			backendVersion: "v1beta2",
			names:          []string{"D2", "D3"},
			inputDrives: []runtime.Object{
				createTestDrive("N1", "D1", "v1beta2", map[string]string{string(utils.NodeLabelKey): "N1", string(utils.AccessTierLabelKey): "Hot"}),
				createTestDrive("N2", "D2", "v1beta2", map[string]string{string(utils.NodeLabelKey): "N2", string(utils.AccessTierLabelKey): "Hot"}),
				createTestDrive("N2", "D3", "v1beta2", map[string]string{string(utils.NodeLabelKey): "N2", string(utils.AccessTierLabelKey): "Hot"}),
			},
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	var unstructuredObjects []*unstructured.Unstructured
	for i, testCase := range testCases {
		for i := range testCase.inputDrives {
			obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(testCase.inputDrives[i])
			if err != nil {
				t.Errorf("case %v: Error in converting to unstructured object, Expected err to nil, got %v", i+1, err)
			}
			unstructuredObjects = append(unstructuredObjects, &unstructured.Unstructured{Object: obj})
		}

		fakeDirecCSIClient, err := getFakeDirectCSIDriveListAdapter(testCase.backendVersion, "directcsidrives", "DirectCSIDriveList", unstructuredObjects...)
		if err != nil {
			t.Errorf(" case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
		}
		fakeDirectCSIListAdapter := &directCSIDriveInterface{*fakeDirecCSIClient}
		labelMap := map[utils.LabelKey][]utils.LabelValue{
			utils.NodeLabelKey: {utils.NewLabelValue("N2")},
		}
		if testCase.backendVersion == "v1beta2" {
			labelMap[utils.AccessTierLabelKey] = []utils.LabelValue{utils.NewLabelValue("Hot")}
		}
		driveList, err := fakeDirectCSIListAdapter.List(ctx, metav1.ListOptions{LabelSelector: utils.ToLabelSelector(labelMap)})
		if err != nil {
			t.Errorf("case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
		}
		var names []string
		for _, item := range driveList.Items {
			names = append(names, item.Name)
		}
		expectedGV := schema.GroupVersion{Group: directcsi.Group, Version: directcsi.Version}
		if expectedGV != driveList.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: result: expected: %v, got: %v", i+1, expectedGV, driveList.GetObjectKind().GroupVersionKind().GroupVersion())
		}
		if !reflect.DeepEqual(names, testCase.names) {
			t.Fatalf("case %v: result: expected: %v, got: %v", i+1, testCase.names, names)
		}
	}
}

func TestCreateDrive(t *testing.T) {
	testCases := []struct {
		backendVersion string
		name           string
		newDriveName   string
		inputDrive     *directcsi.DirectCSIDrive
		newDrive       *directcsi.DirectCSIDrive
	}{
		{
			backendVersion: "v1alpha1",
			name:           "D1",
			inputDrive:     createTestDrive("N1", "D1", "v1alpha1", map[string]string{}),
			newDrive:       createTestDrive("N1", "New-D1", "v1alpha1", map[string]string{}),
		},
		{
			backendVersion: "v1beta1",
			name:           "D2",
			inputDrive:     createTestDrive("N1", "D2", "v1beta1", map[string]string{}),
			newDrive:       createTestDrive("N1", "New-D2", "v1beta1", map[string]string{}),
		},
		{
			backendVersion: "v1beta2",
			name:           "D3",
			inputDrive:     createTestDrive("N1", "D3", "v1beta2", map[string]string{string(utils.NodeLabelKey): "N1"}),
			newDrive:       createTestDrive("N1", "New-D3", "v1beta2", map[string]string{string(utils.NodeLabelKey): "N1"}),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		fakeDirectCSIDriveAdapter := getFakeDirectCSIDriveAdapter(testCase.inputDrive, i, testCase.backendVersion, t)
		createdDrive, err := fakeDirectCSIDriveAdapter.Create(ctx, testCase.newDrive, metav1.CreateOptions{})
		if err != nil {
			t.Errorf("case %v: Error in Create, Expected  err to nil, got %v", i+1, err)
		}
		expectedGV := schema.GroupVersion{Group: directcsi.Group, Version: directcsi.Version}
		if expectedGV != createdDrive.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: result: expected: %v, got: %v", i+1, expectedGV, createdDrive.GetObjectKind().GroupVersionKind().GroupVersion())
		}
	}
}

func TestDeleteDrive(t *testing.T) {
	testCases := []struct {
		backendVersion string
		name           string
		inputDrive     runtime.Object
	}{
		{
			backendVersion: "v1alpha1",
			name:           "D1",
			inputDrive:     createTestDrive("N1", "D1", "v1alpha1", map[string]string{}),
		},
		{
			backendVersion: "v1beta1",
			name:           "D2",
			inputDrive:     createTestDrive("N1", "D2", "v1beta1", map[string]string{}),
		},
		{
			backendVersion: "v1beta2",
			name:           "D3",
			inputDrive:     createTestDrive("N1", "D3", "v1beta2", map[string]string{string(utils.NodeLabelKey): "N1"}),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		fakeDirectCSIDriveAdapter := getFakeDirectCSIDriveAdapter(testCase.inputDrive, i, testCase.backendVersion, t)
		err := fakeDirectCSIDriveAdapter.Delete(ctx, testCase.name, metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("case %v: Error in Delete, Expected  err to nil, got %v", i+1, err)
		}
		_, err = fakeDirectCSIDriveAdapter.Get(ctx, testCase.name, metav1.GetOptions{})
		if err != nil && !k8serror.IsNotFound(err) {
			t.Errorf("case %v: Error in Get after delete, Expected  err to nil, got %v", i+1, err)
		}

	}
}

func TestDeleteCollectionDrive(t *testing.T) {
	testCases := []struct {
		backendVersion string
		name           string
		inputDrive     runtime.Object
	}{
		{
			backendVersion: "v1alpha1",
			name:           "D1",
			inputDrive:     createTestDrive("N1", "D1", "v1alpha1", map[string]string{}),
		},
		{
			backendVersion: "v1beta1",
			name:           "D2",
			inputDrive:     createTestDrive("N1", "D2", "v1beta1", map[string]string{}),
		},
		{
			backendVersion: "v1beta2",
			name:           "D3",
			inputDrive:     createTestDrive("N1", "D3", "v1beta2", map[string]string{string(utils.NodeLabelKey): "N1"}),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		fakeDirectCSIDriveAdapter := getFakeDirectCSIDriveAdapter(testCase.inputDrive, i, testCase.backendVersion, t)
		err := fakeDirectCSIDriveAdapter.DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{})
		if err != nil {
			t.Errorf("case %v: Error in Delete, Expected  err to nil, got %v", i+1, err)
		}
		_, err = fakeDirectCSIDriveAdapter.Get(ctx, testCase.name, metav1.GetOptions{})
		if err != nil && !k8serror.IsNotFound(err) {
			t.Errorf("case %v: Error in Get after delete, Expected  err to nil, got %v", i+1, err)
		}
	}
}

func TestUpdateDrive(t *testing.T) {
	testCases := []struct {
		backendVersion string
		name           string
		accessTier     directcsi.AccessTier
		inputDrive     *directcsi.DirectCSIDrive
	}{
		{
			backendVersion: "v1alpha1",
			name:           "D1",
			accessTier:     directcsi.AccessTierWarm,
			inputDrive:     createTestDrive("N1", "D1", "v1alpha1", map[string]string{}),
		},
		{
			backendVersion: "v1beta1",
			name:           "D2",
			accessTier:     directcsi.AccessTierHot,
			inputDrive:     createTestDrive("N1", "D2", "v1beta1", map[string]string{}),
		},
		{
			backendVersion: "v1beta2",
			name:           "D3",
			accessTier:     directcsi.AccessTierHot,
			inputDrive:     createTestDrive("N1", "D3", "v1beta2", map[string]string{string(utils.NodeLabelKey): "N1"}),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		fakeDirectCSIDriveAdapter := getFakeDirectCSIDriveAdapter(testCase.inputDrive, i, testCase.backendVersion, t)
		testCase.inputDrive.Status.AccessTier = testCase.accessTier
		updatedDrive, err := fakeDirectCSIDriveAdapter.Update(ctx, testCase.inputDrive, metav1.UpdateOptions{})
		if err != nil {
			t.Errorf("case %v: Error in Get, Expected  err to nil, got %v", i+1, err)
		}
		expectedGV := schema.GroupVersion{Group: directcsi.Group, Version: directcsi.Version}
		if expectedGV != updatedDrive.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: result: expected: %v, got: %v", i+1, expectedGV, updatedDrive.GetObjectKind().GroupVersionKind().GroupVersion())
		}

		if updatedDrive.Status.AccessTier != testCase.accessTier {
			t.Fatalf("case %v: result: expected Access tier : %v, got: %v", i+1, updatedDrive.Status.AccessTier, testCase.accessTier)
		}

	}
}

func TestUpdateStatusDrive(t *testing.T) {
	testCases := []struct {
		backendVersion string
		name           string
		accessTier     directcsi.AccessTier
		inputDrive     *directcsi.DirectCSIDrive
	}{
		{
			backendVersion: "v1alpha1",
			name:           "D1",
			accessTier:     directcsi.AccessTierWarm,
			inputDrive:     createTestDrive("N1", "D1", "v1alpha1", map[string]string{}),
		},
		{
			backendVersion: "v1beta1",
			name:           "D2",
			accessTier:     directcsi.AccessTierHot,
			inputDrive:     createTestDrive("N1", "D2", "v1beta1", map[string]string{}),
		},
		{
			backendVersion: "v1beta2",
			name:           "D3",
			accessTier:     directcsi.AccessTierHot,
			inputDrive:     createTestDrive("N1", "D3", "v1beta2", map[string]string{string(utils.NodeLabelKey): "N1"}),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		fakeDirectCSIDriveAdapter := getFakeDirectCSIDriveAdapter(testCase.inputDrive, i, testCase.backendVersion, t)
		testCase.inputDrive.Status.AccessTier = testCase.accessTier
		updatedDrive, err := fakeDirectCSIDriveAdapter.UpdateStatus(ctx, testCase.inputDrive, metav1.UpdateOptions{})
		if err != nil {
			t.Errorf("case %v: Error in Get, Expected  err to nil, got %v", i+1, err)
		}
		expectedGV := schema.GroupVersion{Group: directcsi.Group, Version: directcsi.Version}
		if expectedGV != updatedDrive.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: result: expected: %v, got: %v", i+1, expectedGV, updatedDrive.GetObjectKind().GroupVersionKind().GroupVersion())
		}

		if updatedDrive.Status.AccessTier != testCase.accessTier {
			t.Fatalf("case %v: result: expected Access tier : %v, got: %v", i+1, updatedDrive.Status.AccessTier, testCase.accessTier)
		}
	}
}

func TestWatcher(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	checkEvent := func(event watch.Event, expectedEventType watch.EventType, expectedObject *directcsi.DirectCSIDrive) error {
		if event.Type != expectedEventType {
			return fmt.Errorf("Expected event type: %v, Got: %v", expectedEventType, event.Type)
		}
		var directCSIDrive directcsi.DirectCSIDrive
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Object.(*unstructured.Unstructured).Object, &directCSIDrive); err != nil {
			return fmt.Errorf("Error while converting event object %v", err)
		}
		if !reflect.DeepEqual(&directCSIDrive, expectedObject) {
			return fmt.Errorf("invalid object received in the %s event. Expected: %v, Got: %v", expectedEventType, expectedObject, directCSIDrive)
		}
		return nil
	}
	inputDrive := createTestDrive("N1", "drive-1", "v1beta1", map[string]string{string(utils.NodeLabelKey): "N1"})
	fakeDriveClient := getFakeDirectCSIDriveAdapter(inputDrive, 0, "v1beta1", t)
	fakeWatchInterface, err := fakeDriveClient.Watch(ctx, metav1.ListOptions{})
	if err != nil {
		t.Errorf("Error in creating watch interface: %v", err)
	}
	resultCh := watchInterfaceWrapper{fakeWatchInterface}.ResultChan()

	// Test create event
	testCreateObject := &directcsi.DirectCSIDrive{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "drive-2",
		},
	}
	_, err = fakeDriveClient.Create(ctx, testCreateObject, metav1.CreateOptions{})
	if err != nil {
		t.Errorf("Error while fetching newly created drive, Expected  err to nil, got %v", err)
	}
	event, ok := <-resultCh
	if !ok {
		t.Fatalf("no event received for create")
	}
	if err := checkEvent(event, watch.Added, testCreateObject); err != nil {
		t.Fatalf(err.Error())
	}

	// Test Update Event
	testUpdateObject := &directcsi.DirectCSIDrive{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "drive-1",
		},
		Status: directcsi.DirectCSIDriveStatus{
			DriveStatus: directcsi.DriveStatusReleased,
		},
	}
	_, err = fakeDriveClient.Update(ctx, testUpdateObject, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Error while updating fake drive object: %v", err)
	}
	event, ok = <-resultCh
	if !ok {
		t.Fatalf("no event received for update")
	}
	if err := checkEvent(event, watch.Modified, testUpdateObject); err != nil {
		t.Fatalf(err.Error())
	}

	// Test Delete Event
	err = fakeDriveClient.Delete(ctx, "drive-1", metav1.DeleteOptions{})
	if err != nil {
		t.Errorf("error while deleting fake object: %v", err)
	}
	event, ok = <-resultCh
	if !ok {
		t.Fatalf("no event received for delete")
	}
	if err := checkEvent(event, watch.Deleted, testUpdateObject); err != nil {
		t.Fatalf(err.Error())
	}
}

func createTestVolume(volName string, backendVersion string) *directcsi.DirectCSIVolume {
	return &directcsi.DirectCSIVolume{
		TypeMeta: metav1.TypeMeta{
			APIVersion: string(directcsi.Group + "/" + backendVersion),
			Kind:       "DirectCSIVolume",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: volName,
			Labels: map[string]string{
				tenantLabel: testTenantName,
			},
		},
		Status: directcsi.DirectCSIVolumeStatus{
			NodeName:      testNodeName,
			Drive:         testDrivePath,
			ContainerPath: "/path/containerpath",
		},
	}
}

func getFakeDirectCSIVolumeAdapter(drive runtime.Object, i int, version string, t *testing.T) *directCSIVolumeInterface {
	unstructuredObject, err := runtime.DefaultUnstructuredConverter.ToUnstructured(drive)
	if err != nil {
		t.Errorf("case %v: Error in converting to unstructured object, Expected err to nil, got %v", i+1, err)
	}
	fakeDirecCSIClient, err := getFakeDirectCSIAdapter(version, "directcsivolumes", &unstructured.Unstructured{Object: unstructuredObject})
	if err != nil {
		t.Errorf("case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
	}
	return &directCSIVolumeInterface{*fakeDirecCSIClient}
}
func TestGetVolume(t *testing.T) {
	testCases := []struct {
		backendVersion string
		name           string
		inputVolume    runtime.Object
	}{
		{
			backendVersion: "v1alpha1",
			name:           "V1",
			inputVolume:    createTestVolume("V1", "v1alpha1"),
		},
		{
			backendVersion: "v1beta1",
			name:           "V2",
			inputVolume:    createTestVolume("V2", "v1beta1"),
		},
		{
			backendVersion: "v1beta2",
			name:           "V3",
			inputVolume:    createTestVolume("V3", "v1beta2"),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		fakeDirectCSIVolumeAdapter := getFakeDirectCSIVolumeAdapter(testCase.inputVolume, i, testCase.backendVersion, t)
		directCSIVolume, err := fakeDirectCSIVolumeAdapter.Get(ctx, testCase.name, metav1.GetOptions{})
		if err != nil {
			t.Errorf("case %v: Error in Get, Expected  err to nil, got %v", i+1, err)
		}

		expectedGV := schema.GroupVersion{Group: directcsi.Group, Version: directcsi.Version}
		if expectedGV != directCSIVolume.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: result: expected: %v, got: %v", i+1, expectedGV, directCSIVolume.GetObjectKind().GroupVersionKind().GroupVersion())
		}
	}
}

func TestListVolume(t *testing.T) {
	testCases := []struct {
		backendVersion string
		name           string
		inputVolumes   []runtime.Object
	}{
		{
			backendVersion: "v1alpha1",
			name:           "D1",
			inputVolumes:   []runtime.Object{createTestVolume("V1", "v1alpha1"), createTestVolume("V2", "v1alpha1")},
		},
		{
			backendVersion: "v1beta1",
			name:           "D2",
			inputVolumes:   []runtime.Object{createTestVolume("V1", "v1beta1"), createTestVolume("V2", "v1beta1")},
		},
		{
			backendVersion: "v1beta2",
			name:           "D3",
			inputVolumes:   []runtime.Object{createTestVolume("V1", "v1beta2"), createTestVolume("V2", "v1beta2")},
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	var unstructuredObjects []*unstructured.Unstructured
	for i, testCase := range testCases {
		for i, value := range testCase.inputVolumes {
			obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(value)
			if err != nil {
				t.Errorf("case %v: Error in converting to unstructured object, Expected err to nil, got %v", i+1, err)
			}
			unstructuredObjects = append(unstructuredObjects, &unstructured.Unstructured{Object: obj})
		}

		fakeDirecCSIClient, err := getFakeDirectCSIDriveListAdapter(testCase.backendVersion, "directcsidrives", "DirectCSIDriveList", unstructuredObjects...)
		if err != nil {
			t.Errorf(" case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
		}
		fakeDirectCSIListAdapter := &directCSIVolumeInterface{*fakeDirecCSIClient}
		volumeList, err := fakeDirectCSIListAdapter.List(ctx, metav1.ListOptions{})
		if err != nil {
			t.Errorf("case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
		}
		expectedGV := schema.GroupVersion{Group: directcsi.Group, Version: directcsi.Version}
		if expectedGV != volumeList.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: result: expected: %v, got: %v", i+1, expectedGV, volumeList.GetObjectKind().GroupVersionKind().GroupVersion())
		}
	}

}

func TestCreateVolume(t *testing.T) {
	testCases := []struct {
		backendVersion string
		name           string
		inputVolume    *directcsi.DirectCSIVolume
		newVolume      *directcsi.DirectCSIVolume
	}{
		{
			backendVersion: "v1alpha1",
			name:           "V1",
			inputVolume:    createTestVolume("V1", "v1alpha1"),
			newVolume:      createTestVolume("V1-New", "v1alpha1"),
		},
		{
			backendVersion: "v1beta1",
			name:           "V2",
			inputVolume:    createTestVolume("V2", "v1beta1"),
			newVolume:      createTestVolume("V2-New", "v1beta1"),
		},
		{
			backendVersion: "v1beta2",
			name:           "V3",
			inputVolume:    createTestVolume("V3", "v1beta2"),
			newVolume:      createTestVolume("V3-New", "v1beta2"),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		fakeDirectCSIVolumeAdapter := getFakeDirectCSIVolumeAdapter(testCase.inputVolume, i, testCase.backendVersion, t)
		createdVolume, err := fakeDirectCSIVolumeAdapter.Create(ctx, testCase.newVolume, metav1.CreateOptions{})
		if err != nil {
			t.Errorf("case %v: Error in Create, Expected  err to nil, got %v", i+1, err)
		}

		expectedGV := schema.GroupVersion{Group: directcsi.Group, Version: directcsi.Version}
		if expectedGV != createdVolume.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: result: expected: %v, got: %v", i+1, expectedGV, createdVolume.GetObjectKind().GroupVersionKind().GroupVersion())
		}
	}
}

func TestDeleteVolume(t *testing.T) {
	testCases := []struct {
		backendVersion string
		name           string
		inputVolume    runtime.Object
	}{
		{
			backendVersion: "v1alpha1",
			name:           "V1",
			inputVolume:    createTestVolume("V1", "v1alpha1"),
		},
		{
			backendVersion: "v1beta1",
			name:           "V2",
			inputVolume:    createTestVolume("V2", "v1beta1"),
		},
		{
			backendVersion: "v1beta2",
			name:           "V3",
			inputVolume:    createTestVolume("V3", "v1beta2"),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		fakeDirectCSIVolumeAdapter := getFakeDirectCSIVolumeAdapter(testCase.inputVolume, i, testCase.backendVersion, t)

		err := fakeDirectCSIVolumeAdapter.Delete(ctx, testCase.name, metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("case %v: Error in Delete, Expected  err to nil, got %v", i+1, err)
		}
		_, err = fakeDirectCSIVolumeAdapter.Get(ctx, testCase.name, metav1.GetOptions{})

		if err != nil && !k8serror.IsNotFound(err) {
			t.Errorf("case %v: Error in Get after delete, Expected  err to nil, got %v", i+1, err)
		}
	}
}

func TestVolumeDeleteCollection(t *testing.T) {
	testCases := []struct {
		backendVersion string
		name           string
		inputVolume    runtime.Object
	}{
		{
			backendVersion: "v1alpha1",
			name:           "V1",
			inputVolume:    createTestVolume("V1", "v1alpha1"),
		},
		{
			backendVersion: "v1beta1",
			name:           "V2",
			inputVolume:    createTestVolume("V2", "v1beta1"),
		},
		{
			backendVersion: "v1beta2",
			name:           "V3",
			inputVolume:    createTestVolume("V3", "v1beta2"),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		fakeDirectCSIVolumeAdapter := getFakeDirectCSIVolumeAdapter(testCase.inputVolume, i, testCase.backendVersion, t)

		err := fakeDirectCSIVolumeAdapter.DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{})
		if err != nil {
			t.Errorf("case %v: Error in Delete, Expected  err to nil, got %v", i+1, err)
		}
		_, err = fakeDirectCSIVolumeAdapter.Get(ctx, testCase.name, metav1.GetOptions{})

		if err != nil && !k8serror.IsNotFound(err) {
			t.Errorf("case %v: Error in Get after delete, Expected  err to nil, got %v", i+1, err)
		}
	}
}

func TestUpdateVolume(t *testing.T) {
	testCases := []struct {
		backendVersion    string
		name              string
		availableCapacity int
		inputVolume       *directcsi.DirectCSIVolume
	}{
		{
			backendVersion:    "v1alpha1",
			name:              "V1",
			availableCapacity: mb20,
			inputVolume:       createTestVolume("V1", "v1alpha1"),
		},
		{
			backendVersion:    "v1beta1",
			name:              "V2",
			availableCapacity: mb20,
			inputVolume:       createTestVolume("V2", "v1beta1"),
		},
		{
			backendVersion:    "v1beta2",
			name:              "V3",
			availableCapacity: mb20,
			inputVolume:       createTestVolume("V3", "v1beta2"),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		fakeDirectCSIVolumeAdapter := getFakeDirectCSIVolumeAdapter(testCase.inputVolume, i, testCase.backendVersion, t)
		testCase.inputVolume.Status.AvailableCapacity = int64(testCase.availableCapacity)
		updatedVolume, err := fakeDirectCSIVolumeAdapter.Update(ctx, testCase.inputVolume, metav1.UpdateOptions{})
		if err != nil {
			t.Errorf("case %v: Error in Get, Expected  err to nil, got %v", i+1, err)
		}
		expectedGV := schema.GroupVersion{Group: directcsi.Group, Version: directcsi.Version}
		if expectedGV != updatedVolume.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: result: expected: %v, got: %v", i+1, expectedGV, updatedVolume.GetObjectKind().GroupVersionKind().GroupVersion())
		}

		if updatedVolume.Status.AvailableCapacity != int64(testCase.availableCapacity) {
			t.Fatalf("case %v: result: expected available capacity : %v, got: %v", i+1, updatedVolume.Status.AvailableCapacity, testCase.availableCapacity)
		}
	}
}

func TestUpdateStatusVolume(t *testing.T) {
	testCases := []struct {
		backendVersion    string
		name              string
		availableCapacity int
		inputVolume       *directcsi.DirectCSIVolume
	}{
		{
			backendVersion:    "v1alpha1",
			name:              "V1",
			availableCapacity: mb20,
			inputVolume:       createTestVolume("V1", "v1alpha1"),
		},
		{
			backendVersion:    "v1beta1",
			name:              "V2",
			availableCapacity: mb20,
			inputVolume:       createTestVolume("V2", "v1beta1"),
		},
		{
			backendVersion:    "v1beta2",
			name:              "V3",
			availableCapacity: mb20,
			inputVolume:       createTestVolume("V3", "v1beta2"),
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		fakeDirectCSIVolumeAdapter := getFakeDirectCSIVolumeAdapter(testCase.inputVolume, i, testCase.backendVersion, t)
		testCase.inputVolume.Status.AvailableCapacity = int64(testCase.availableCapacity)
		updatedVolume, err := fakeDirectCSIVolumeAdapter.UpdateStatus(ctx, testCase.inputVolume, metav1.UpdateOptions{})
		if err != nil {
			t.Errorf("case %v: Error in Get, Expected  err to nil, got %v", i+1, err)
		}
		expectedGV := schema.GroupVersion{Group: directcsi.Group, Version: directcsi.Version}
		if expectedGV != updatedVolume.GetObjectKind().GroupVersionKind().GroupVersion() {
			t.Fatalf("case %v: result: expected: %v, got: %v", i+1, expectedGV, updatedVolume.GetObjectKind().GroupVersionKind().GroupVersion())
		}

		if updatedVolume.Status.AvailableCapacity != int64(testCase.availableCapacity) {
			t.Fatalf("case %v: result: expected available capacity : %v, got: %v", i+1, updatedVolume.Status.AvailableCapacity, testCase.availableCapacity)
		}
	}
}
