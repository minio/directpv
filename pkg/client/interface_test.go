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
	"fmt"
	"reflect"
	"testing"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/utils"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	fakedynamic "k8s.io/client-go/dynamic/fake"
)

func newUnstructured(backendVersion, kind, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", directcsi.Group, backendVersion),
			"kind":       kind,
			"metadata": map[string]interface{}{
				"namespace": metav1.NamespaceNone,
				"name":      name,
			},
		},
	}
}

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

func TestGetDrive(t *testing.T) {
	testCases := []struct {
		backendVersion string
		kind           string
		name           string
		expectedDrive  *directcsi.DirectCSIDrive
	}{
		{
			backendVersion: "v1alpha1",
			kind:           "DirectCSIDrive",
			name:           "drive-1",
			expectedDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{Name: "drive-1", Labels: map[string]string{
					"direct.csi.min.io/access-tier": "Unknown",
					"direct.csi.min.io/created-by":  "directcsi-driver",
					"direct.csi.min.io/node":        "",
					"direct.csi.min.io/path":        "dev",
					"direct.csi.min.io/version":     "v1alpha1",
				}},
				Status: directcsi.DirectCSIDriveStatus{Path: "/dev", AccessTier: "Unknown"},
			},
		},
		{
			backendVersion: "v1beta1",
			kind:           "DirectCSIDrive",
			name:           "drive-2",
			expectedDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{Name: "drive-2", Labels: map[string]string{
					"direct.csi.min.io/access-tier": "",
					"direct.csi.min.io/created-by":  "directcsi-driver",
					"direct.csi.min.io/node":        "",
					"direct.csi.min.io/path":        "dev",
					"direct.csi.min.io/version":     "v1beta1",
				}},
				Status: directcsi.DirectCSIDriveStatus{Path: "/dev"},
			},
		},
		{
			backendVersion: "v1beta2",
			kind:           "DirectCSIDrive",
			name:           "drive-3",
			expectedDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{Name: "drive-3", Labels: map[string]string{
					"direct.csi.min.io/access-tier": "",
					"direct.csi.min.io/created-by":  "directcsi-driver",
					"direct.csi.min.io/node":        "",
					"direct.csi.min.io/path":        "dev",
					"direct.csi.min.io/version":     "v1beta2"}},
				Status: directcsi.DirectCSIDriveStatus{Path: "/dev"},
			},
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	for i, testCase := range testCases {
		fakeDirecCSIClient, err := getFakeDirectCSIAdapter(testCase.backendVersion, "directcsidrives", newUnstructured(testCase.backendVersion, testCase.kind, testCase.name))
		if err != nil {
			t.Errorf("case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
		}
		fakeDirectCSIDriveAdapter := &directCSIDriveInterface{*fakeDirecCSIClient}
		directCSIDrive, err := fakeDirectCSIDriveAdapter.Get(ctx, testCase.name, metav1.GetOptions{})
		if err != nil {
			t.Errorf("case %v: Error in Get, Expected  err to nil, got %v", i+1, err)
		}

		if !reflect.DeepEqual(directCSIDrive, testCase.expectedDrive) {
			t.Fatalf("case %v: result: expected: %+v, got: %+v", i+1, testCase.expectedDrive, directCSIDrive)
		}
	}
}

func TestListDrive(t *testing.T) {
	testCases := []struct {
		backendVersion    string
		kind              string
		name              []string
		expectedDriveList *directcsi.DirectCSIDriveList
	}{
		{
			backendVersion: "v1alpha1",
			kind:           "DirectCSIDrive",
			name:           []string{"drive-1", "drive-2"},
			expectedDriveList: &directcsi.DirectCSIDriveList{
				TypeMeta: metav1.TypeMeta{
					Kind:       "DirectCSIDriveList",
					APIVersion: string(utils.DirectCSIVersionLabelKey),
				},
				Items: []directcsi.DirectCSIDrive{
					{
						TypeMeta: utils.DirectCSIDriveTypeMeta(),
						ObjectMeta: metav1.ObjectMeta{Name: "drive-1",
							Labels: map[string]string{
								"direct.csi.min.io/access-tier": "Unknown",
								"direct.csi.min.io/created-by":  "directcsi-driver",
								"direct.csi.min.io/node":        "",
								"direct.csi.min.io/path":        "dev",
								"direct.csi.min.io/version":     "v1alpha1",
							},
						},
						Status: directcsi.DirectCSIDriveStatus{Path: "/dev", AccessTier: "Unknown"},
					},
					{
						TypeMeta: utils.DirectCSIDriveTypeMeta(),
						ObjectMeta: metav1.ObjectMeta{Name: "drive-2",
							Labels: map[string]string{
								"direct.csi.min.io/access-tier": "Unknown",
								"direct.csi.min.io/created-by":  "directcsi-driver",
								"direct.csi.min.io/node":        "",
								"direct.csi.min.io/path":        "dev",
								"direct.csi.min.io/version":     "v1alpha1",
							},
						},
						Status: directcsi.DirectCSIDriveStatus{Path: "/dev", AccessTier: "Unknown"},
					},
				},
			},
		},
		{
			backendVersion: "v1beta1",
			kind:           "DirectCSIDrive",
			name:           []string{"drive-1", "drive-2"},
			expectedDriveList: &directcsi.DirectCSIDriveList{
				TypeMeta: metav1.TypeMeta{
					Kind:       "DirectCSIDriveList",
					APIVersion: string(utils.DirectCSIVersionLabelKey),
				},
				Items: []directcsi.DirectCSIDrive{
					{
						TypeMeta: utils.DirectCSIDriveTypeMeta(),
						ObjectMeta: metav1.ObjectMeta{Name: "drive-1",
							Labels: map[string]string{
								"direct.csi.min.io/access-tier": "",
								"direct.csi.min.io/created-by":  "directcsi-driver",
								"direct.csi.min.io/node":        "",
								"direct.csi.min.io/path":        "dev",
								"direct.csi.min.io/version":     "v1beta1",
							},
						},
						Status: directcsi.DirectCSIDriveStatus{Path: "/dev"},
					},
					{
						TypeMeta: utils.DirectCSIDriveTypeMeta(),
						ObjectMeta: metav1.ObjectMeta{Name: "drive-2", Labels: map[string]string{
							"direct.csi.min.io/access-tier": "",
							"direct.csi.min.io/created-by":  "directcsi-driver",
							"direct.csi.min.io/node":        "",
							"direct.csi.min.io/path":        "dev",
							"direct.csi.min.io/version":     "v1beta1",
						}},
						Status: directcsi.DirectCSIDriveStatus{Path: "/dev"},
					},
				},
			},
		},
		{
			backendVersion: "v1beta2",
			kind:           "DirectCSIDrive",
			name:           []string{"drive-1", "drive-2"},
			expectedDriveList: &directcsi.DirectCSIDriveList{
				TypeMeta: metav1.TypeMeta{
					Kind:       "DirectCSIDriveList",
					APIVersion: string(utils.DirectCSIVersionLabelKey),
				},
				Items: []directcsi.DirectCSIDrive{
					{
						TypeMeta: utils.DirectCSIDriveTypeMeta(),
						ObjectMeta: metav1.ObjectMeta{Name: "drive-1", Labels: map[string]string{
							"direct.csi.min.io/access-tier": "",
							"direct.csi.min.io/created-by":  "directcsi-driver",
							"direct.csi.min.io/node":        "",
							"direct.csi.min.io/path":        "dev",
							"direct.csi.min.io/version":     "v1beta2"}},
						Status: directcsi.DirectCSIDriveStatus{Path: "/dev"},
					},
					{
						TypeMeta: utils.DirectCSIDriveTypeMeta(),
						ObjectMeta: metav1.ObjectMeta{Name: "drive-2", Labels: map[string]string{
							"direct.csi.min.io/access-tier": "",
							"direct.csi.min.io/created-by":  "directcsi-driver",
							"direct.csi.min.io/node":        "",
							"direct.csi.min.io/path":        "dev",
							"direct.csi.min.io/version":     "v1beta2"}},
						Status: directcsi.DirectCSIDriveStatus{Path: "/dev"},
					},
				},
			},
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {

		fakeDirecCSIClient, err := getFakeDirectCSIDriveListAdapter(testCase.backendVersion, "directcsidrives", "DirectCSIDriveList",
			newUnstructured(testCase.backendVersion, testCase.kind, testCase.name[0]),
			newUnstructured(testCase.backendVersion, testCase.kind, testCase.name[1]))
		if err != nil {
			t.Errorf(" case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
		}

		fakeDirectCSIListAdapter := &directCSIDriveInterface{*fakeDirecCSIClient}

		driveList, err := fakeDirectCSIListAdapter.List(ctx, metav1.ListOptions{})
		if err != nil {
			t.Errorf("case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
		}

		if !reflect.DeepEqual(driveList, testCase.expectedDriveList) {
			t.Fatalf("case %v: result: expected: %+v, got: %+v", i+1, driveList, testCase.expectedDriveList)
		}
	}

}

func TestCreateDrive(t *testing.T) {
	testCases := []struct {
		backendVersion      string
		kind                string
		name                string
		newDriveName        string
		driveRepresentation *directcsi.DirectCSIDrive
	}{
		{
			backendVersion: "v1alpha1",
			kind:           "DirectCSIDrive",
			name:           "drive-1",
			newDriveName:   "newdrive-1",
			driveRepresentation: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "newdrive-1",
				},
			},
		},
		{
			backendVersion: "vbeta1",
			kind:           "DirectCSIDrive",
			name:           "drive-2",
			newDriveName:   "newdrive-2",
			driveRepresentation: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "newdrive-2",
				},
			},
		},
		{
			backendVersion: "vbeta3",
			kind:           "DirectCSIDrive",
			name:           "drive-3",
			newDriveName:   "newdrive-3",
			driveRepresentation: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "newdrive-3",
				},
			},
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		fakeDirecCSIClient, err := getFakeDirectCSIAdapter(testCase.backendVersion, "directcsidrives", newUnstructured(testCase.backendVersion, testCase.kind, testCase.name))
		if err != nil {
			t.Errorf("case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
		}
		fakeDirectCSIDriveAdapter := &directCSIDriveInterface{*fakeDirecCSIClient}
		_, err = fakeDirectCSIDriveAdapter.Create(ctx, testCase.driveRepresentation, metav1.CreateOptions{})
		if err != nil {
			t.Errorf("case %v: Error in Create, Expected  err to nil, got %v", i+1, err)
		}

		_, err = fakeDirectCSIDriveAdapter.Get(ctx, testCase.newDriveName, metav1.GetOptions{})
		if err != nil {
			t.Errorf("case %v: Error while fetching newly created drive, Expected  err to nil, got %v", i+1, err)
		}
	}
}

func TestDeleteDrive(t *testing.T) {
	testCases := []struct {
		backendVersion string
		kind           string
		name           string
	}{
		{
			backendVersion: "v1alpha1",
			kind:           "DirectCSIDrive",
			name:           "direct-1",
		},
		{
			backendVersion: "v1beta1",
			kind:           "DirectCSIDrive",
			name:           "drive-2",
		},
		{
			backendVersion: "v1beta2",
			kind:           "DirectCSIDrive",
			name:           "drive-3",
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {

		fakeDirecCSIClient, err := getFakeDirectCSIAdapter(testCase.backendVersion, "directcsidrives", newUnstructured(testCase.backendVersion, testCase.kind, testCase.name))
		if err != nil {
			t.Errorf("case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
		}
		fakeDirectCSIDriveAdapter := &directCSIDriveInterface{*fakeDirecCSIClient}

		err = fakeDirectCSIDriveAdapter.Delete(ctx, testCase.name, metav1.DeleteOptions{})
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
		kind           string
		name           string
	}{
		{
			backendVersion: "v1alpha1",
			kind:           "DirectCSIDrive",
			name:           "drive-1",
		},
		{
			backendVersion: "v1beta1",
			kind:           "DirectCSIDrive",
			name:           "drive-2",
		},
		{
			backendVersion: "v1beta2",
			kind:           "DirectCSIDrive",
			name:           "drive-3",
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		fakeDirecCSIClient, err := getFakeDirectCSIAdapter(testCase.backendVersion, "directcsidrives", newUnstructured(testCase.backendVersion, testCase.kind, testCase.name))
		if err != nil {
			t.Errorf("case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
		}
		fakeDirectCSIDriveAdapter := &directCSIDriveInterface{*fakeDirecCSIClient}
		err = fakeDirectCSIDriveAdapter.DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{})
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
		kind           string
		name           string
		presentDrive   *directcsi.DirectCSIDrive
		expectedDrive  *directcsi.DirectCSIDrive
	}{
		{
			backendVersion: "v1alpha1",
			kind:           "DirectCSIDrive",
			name:           "drive-1",
			presentDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "drive-1",
				},
				Status: directcsi.DirectCSIDriveStatus{AccessTier: "Warm"},
			},
			expectedDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "drive-1",
				},
				Status: directcsi.DirectCSIDriveStatus{AccessTier: "Warm"},
			},
		},
		{
			backendVersion: "v1beta1",
			kind:           "DirectCSIDrive",
			name:           "drive-2",
			presentDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "drive-2",
				},
				Status: directcsi.DirectCSIDriveStatus{AccessTier: "Hot"},
			},
			expectedDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "drive-2",
				},
				Status: directcsi.DirectCSIDriveStatus{AccessTier: "Hot"},
			},
		},
		{
			backendVersion: "v1beta2",
			kind:           "DirectCSIDrive",
			name:           "drive-3",
			presentDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "drive-3",
				},
				Status: directcsi.DirectCSIDriveStatus{AccessTier: "Cold"},
			},
			expectedDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "drive-3",
				},
				Status: directcsi.DirectCSIDriveStatus{AccessTier: "Cold"},
			},
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		fakeDirecCSIClient, err := getFakeDirectCSIAdapter(testCase.backendVersion, "directcsidrives", newUnstructured(testCase.backendVersion, testCase.kind, testCase.name))
		if err != nil {
			t.Errorf("case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
		}
		fakeDirectCSIDriveAdapter := &directCSIDriveInterface{*fakeDirecCSIClient}
		updatedDrive, err := fakeDirectCSIDriveAdapter.Update(ctx, testCase.presentDrive, metav1.UpdateOptions{})
		if err != nil {
			t.Errorf("case %v: Error in Get, Expected  err to nil, got %v", i+1, err)
		}

		if !reflect.DeepEqual(updatedDrive, testCase.expectedDrive) {
			t.Fatalf("case %v: result: expected: %v, got: %v", i+1, testCase.expectedDrive, updatedDrive)
		}

	}
}

func TestUpdateStatusDrive(t *testing.T) {
	testCases := []struct {
		backendVersion string
		kind           string
		name           string
		presentDrive   *directcsi.DirectCSIDrive
		expectedDrive  *directcsi.DirectCSIDrive
	}{
		{
			backendVersion: "v1alpha1",
			kind:           "DirectCSIDrive",
			name:           "drive-1",
			presentDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "drive-1",
				},
				Status: directcsi.DirectCSIDriveStatus{AccessTier: "Warm"},
			},
			expectedDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "drive-1",
				},
				Status: directcsi.DirectCSIDriveStatus{AccessTier: "Warm"},
			},
		},
		{
			backendVersion: "v1beta1",
			kind:           "DirectCSIDrive",
			name:           "drive-2",
			presentDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "drive-2",
				},
				Status: directcsi.DirectCSIDriveStatus{AccessTier: "Hot"},
			},
			expectedDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "drive-2",
				},
				Status: directcsi.DirectCSIDriveStatus{AccessTier: "Hot"},
			},
		},
		{
			backendVersion: "v1beta2",
			kind:           "DirectCSIDrive",
			name:           "drive-3",
			presentDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "drive-3",
				},
				Status: directcsi.DirectCSIDriveStatus{AccessTier: "Cold"},
			},
			expectedDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "drive-3",
				},
				Status: directcsi.DirectCSIDriveStatus{AccessTier: "Cold"},
			},
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		fakeDirecCSIClient, err := getFakeDirectCSIAdapter(testCase.backendVersion, "directcsidrives", newUnstructured(testCase.backendVersion, testCase.kind, testCase.name))
		if err != nil {
			t.Errorf("case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
		}
		fakeDirectCSIDriveAdapter := &directCSIDriveInterface{*fakeDirecCSIClient}
		updatedDrive, err := fakeDirectCSIDriveAdapter.UpdateStatus(ctx, testCase.presentDrive, metav1.UpdateOptions{})
		if err != nil {
			t.Errorf("case %v: Error in Get, Expected  err to nil, got %v", i+1, err)
		}
		if !reflect.DeepEqual(updatedDrive, testCase.expectedDrive) {
			t.Fatalf("case %v: result: expected: %v, got: %v", i+1, testCase.expectedDrive, updatedDrive)
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

	fakeDriveClient, err := getFakeDirectCSIAdapter("v1beta2", "directcsidrives", newUnstructured("v1beta2", "DirectCSIDrive", "drive-1"))
	if err != nil {
		t.Errorf("Error in creating fake drive interface: %v", err)
	}
	fakeWatchInterface, err := fakeDriveClient.Watch(ctx, metav1.ListOptions{})
	if err != nil {
		t.Errorf("Error in creating watch interface: %v", err)
	}
	resultCh := watchInterfaceWrapper{fakeWatchInterface}.ResultChan()
	fakeDirectCSIDriveAdapter := &directCSIDriveInterface{*fakeDriveClient}

	// Test create event
	testCreateObject := &directcsi.DirectCSIDrive{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "drive-2",
		},
	}
	_, err = fakeDirectCSIDriveAdapter.Create(ctx, testCreateObject, metav1.CreateOptions{})
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
	_, err = fakeDirectCSIDriveAdapter.Update(ctx, testUpdateObject, metav1.UpdateOptions{})
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
	err = fakeDirectCSIDriveAdapter.Delete(ctx, "drive-1", metav1.DeleteOptions{})
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

func TestGetVolume(t *testing.T) {
	testCases := []struct {
		backendVersion string
		kind           string
		name           string
		expectedDrive  *directcsi.DirectCSIVolume
	}{
		{
			backendVersion: "v1beta1",
			kind:           "DirectCSIVolume",
			name:           "volume-2",
			expectedDrive: &directcsi.DirectCSIVolume{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{Name: "volume-2", Labels: map[string]string{
					"direct.csi.min.io/created-by": "directcsi-controller",
					"direct.csi.min.io/node":       "",
					"direct.csi.min.io/version":    "v1beta1",
				}},
			},
		},
		{
			backendVersion: "v1beta2",
			kind:           "DirectCSIVolume",
			name:           "volume-3",
			expectedDrive: &directcsi.DirectCSIVolume{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{Name: "volume-3", Labels: map[string]string{
					"direct.csi.min.io/created-by": "directcsi-controller",
					"direct.csi.min.io/node":       "",
					"direct.csi.min.io/version":    "v1beta2",
				}},
			},
		},
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		fakeDirecCSIClient, err := getFakeDirectCSIAdapter(testCase.backendVersion, "directcsivolumes", newUnstructured(testCase.backendVersion, testCase.kind, testCase.name))
		if err != nil {
			t.Errorf("case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
		}
		fakeDirectCSIVolumeAdapter := &directCSIVolumeInterface{*fakeDirecCSIClient}
		directCSIVolume, err := fakeDirectCSIVolumeAdapter.Get(ctx, testCase.name, metav1.GetOptions{})
		if err != nil {
			t.Errorf("case %v: Error in Get, Expected  err to nil, got %v", i+1, err)
		}

		if !reflect.DeepEqual(directCSIVolume, testCase.expectedDrive) {
			t.Fatalf("case %v: result: expected: %v, got: %v", i+1, testCase.expectedDrive, directCSIVolume)
		}
	}
}

func TestDeleteVolume(t *testing.T) {
	testCases := []struct {
		backendVersion string
		kind           string
		name           string
	}{
		{
			backendVersion: "v1alpha1",
			kind:           "DirectCSIVolume",
			name:           "volume-1",
		},
		{
			backendVersion: "v1beta1",
			kind:           "DirectCSIVolume",
			name:           "volume-2",
		},
		{
			backendVersion: "v1beta2",
			kind:           "DirectCSIVolume",
			name:           "volume-3",
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {

		fakeDirecCSIClient, err := getFakeDirectCSIAdapter(testCase.backendVersion, "directcsivolumes", newUnstructured(testCase.backendVersion, testCase.kind, testCase.name))
		if err != nil {
			t.Errorf("case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
		}
		fakeDirectCSIVolumeAdapter := &directCSIVolumeInterface{*fakeDirecCSIClient}

		err = fakeDirectCSIVolumeAdapter.Delete(ctx, testCase.name, metav1.DeleteOptions{})
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
		kind           string
		name           string
	}{
		{
			backendVersion: "v1alpha1",
			kind:           "DirectCSIVolume",
			name:           "volume-1",
		},
		{
			backendVersion: "v1beta1",
			kind:           "DirectCSIVolume",
			name:           "volume-2",
		},
		{
			backendVersion: "v1beta2",
			kind:           "DirectCSIVolume",
			name:           "volume-3",
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		fakeDirecCSIClient, err := getFakeDirectCSIAdapter(testCase.backendVersion, "directcsivolumes", newUnstructured(testCase.backendVersion, testCase.kind, testCase.name))
		if err != nil {
			t.Errorf("case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
		}
		fakeDirectCSIDriveAdapter := &directCSIDriveInterface{*fakeDirecCSIClient}
		err = fakeDirectCSIDriveAdapter.DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{})
		if err != nil {
			t.Errorf("case %v: Error in Delete, Expected  err to nil, got %v", i+1, err)
		}
		_, err = fakeDirectCSIDriveAdapter.Get(ctx, testCase.name, metav1.GetOptions{})
		if err != nil && !k8serror.IsNotFound(err) {
			t.Errorf("case %v: Error in Get after delete, Expected  err to nil, got %v", i+1, err)
		}
	}
}

func TestUpdateVolume(t *testing.T) {
	testCases := []struct {
		backendVersion string
		kind           string
		name           string
		presentVolume  *directcsi.DirectCSIVolume
		expectedVolume *directcsi.DirectCSIVolume
	}{
		{
			backendVersion: "v1alpha1",
			kind:           "DirectCSIVolume",
			name:           "volume-1",
			presentVolume: &directcsi.DirectCSIVolume{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "volume-1",
				},
				Status: directcsi.DirectCSIVolumeStatus{AvailableCapacity: 12899},
			},
			expectedVolume: &directcsi.DirectCSIVolume{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "volume-1",
				},
				Status: directcsi.DirectCSIVolumeStatus{AvailableCapacity: 12899},
			},
		},
		{
			backendVersion: "v1beta1",
			kind:           "DirectCSIVolume",
			name:           "volume-2",
			presentVolume: &directcsi.DirectCSIVolume{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "volume-2",
				},
				Status: directcsi.DirectCSIVolumeStatus{AvailableCapacity: 11111},
			},
			expectedVolume: &directcsi.DirectCSIVolume{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "volume-2",
				},
				Status: directcsi.DirectCSIVolumeStatus{AvailableCapacity: 11111},
			},
		},
		{
			backendVersion: "v1beta2",
			kind:           "DirectCSIVolume",
			name:           "volume-3",
			presentVolume: &directcsi.DirectCSIVolume{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "volume-3",
				},
				Status: directcsi.DirectCSIVolumeStatus{AvailableCapacity: 22222},
			},
			expectedVolume: &directcsi.DirectCSIVolume{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "volume-3",
				},
				Status: directcsi.DirectCSIVolumeStatus{AvailableCapacity: 22222},
			},
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		fakeDirecCSIClient, err := getFakeDirectCSIAdapter(testCase.backendVersion, "directcsivolumes", newUnstructured(testCase.backendVersion, testCase.kind, testCase.name))
		if err != nil {
			t.Errorf("case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
		}
		fakeDirectCSIVolumeAdapter := &directCSIVolumeInterface{*fakeDirecCSIClient}
		updatedVolume, err := fakeDirectCSIVolumeAdapter.Update(ctx, testCase.presentVolume, metav1.UpdateOptions{})
		if err != nil {
			t.Errorf("case %v: Error in Update, Expected  err to nil, got %v", i+1, err)
		}

		if !reflect.DeepEqual(updatedVolume, testCase.expectedVolume) {
			t.Fatalf("case %v: result: expected: %v, got: %v", i+1, testCase.expectedVolume, updatedVolume)
		}

	}
}

func TestUpdateStatusVolume(t *testing.T) {
	testCases := []struct {
		backendVersion string
		kind           string
		name           string
		presentVolume  *directcsi.DirectCSIVolume
		expectedVolume *directcsi.DirectCSIVolume
	}{
		{
			backendVersion: "v1alpha1",
			kind:           "DirectCSIVolume",
			name:           "volume-1",
			presentVolume: &directcsi.DirectCSIVolume{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "volume-1",
				},
				Status: directcsi.DirectCSIVolumeStatus{AvailableCapacity: 12899},
			},
			expectedVolume: &directcsi.DirectCSIVolume{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "volume-1",
				},
				Status: directcsi.DirectCSIVolumeStatus{AvailableCapacity: 12899},
			},
		},
		{
			backendVersion: "v1beta1",
			kind:           "DirectCSIVolume",
			name:           "volume-2",
			presentVolume: &directcsi.DirectCSIVolume{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "volume-2",
				},
				Status: directcsi.DirectCSIVolumeStatus{AvailableCapacity: 11111},
			},
			expectedVolume: &directcsi.DirectCSIVolume{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "volume-2",
				},
				Status: directcsi.DirectCSIVolumeStatus{AvailableCapacity: 11111},
			},
		},
		{
			backendVersion: "v1beta2",
			kind:           "DirectCSIVolume",
			name:           "volume-3",
			presentVolume: &directcsi.DirectCSIVolume{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "volume-3",
				},
				Status: directcsi.DirectCSIVolumeStatus{AvailableCapacity: 22222},
			},
			expectedVolume: &directcsi.DirectCSIVolume{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "volume-3",
				},
				Status: directcsi.DirectCSIVolumeStatus{AvailableCapacity: 22222},
			},
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		fakeDirecCSIClient, err := getFakeDirectCSIAdapter(testCase.backendVersion, "directcsivolumes", newUnstructured(testCase.backendVersion, testCase.kind, testCase.name))
		if err != nil {
			t.Errorf("case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
		}
		fakeDirectCSIVolumeAdapter := &directCSIVolumeInterface{*fakeDirecCSIClient}
		updatedVolume, err := fakeDirectCSIVolumeAdapter.UpdateStatus(ctx, testCase.presentVolume, metav1.UpdateOptions{})
		if err != nil {
			t.Errorf("case %v: Error in Update, Expected  err to nil, got %v", i+1, err)
		}

		if !reflect.DeepEqual(updatedVolume, testCase.expectedVolume) {
			t.Fatalf("case %v: result: expected: %v, got: %v", i+1, testCase.expectedVolume, updatedVolume)
		}

	}
}
func TestCreateVolume(t *testing.T) {
	testCases := []struct {
		backendVersion       string
		kind                 string
		name                 string
		newVolumeName        string
		volumeRepresentation *directcsi.DirectCSIVolume
	}{
		{
			backendVersion: "v1alpha1",
			kind:           "DirectCSIVolume",
			name:           "volume-1",
			newVolumeName:  "newvolume-1",
			volumeRepresentation: &directcsi.DirectCSIVolume{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "newvolume-1",
				},
			},
		},
		{
			backendVersion: "vbeta1",
			kind:           "DirectCSIVolume",
			name:           "volume-2",
			newVolumeName:  "newvolume-2",
			volumeRepresentation: &directcsi.DirectCSIVolume{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "newvolume-2",
				},
			},
		},
		{
			backendVersion: "vbeta3",
			kind:           "DirectCSIVolume",
			name:           "volume-3",
			newVolumeName:  "newvolume-3",
			volumeRepresentation: &directcsi.DirectCSIVolume{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "newvolume-3",
				},
			},
		},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for i, testCase := range testCases {
		fakeDirecCSIClient, err := getFakeDirectCSIAdapter(testCase.backendVersion, "directcsivolumes", newUnstructured(testCase.backendVersion, testCase.kind, testCase.name))
		if err != nil {
			t.Errorf("case %v: Error in creating fake client, Expected err to nil, got %v", i+1, err)
		}
		fakeDirectCSIVolumeAdapter := &directCSIVolumeInterface{*fakeDirecCSIClient}
		_, err = fakeDirectCSIVolumeAdapter.Create(ctx, testCase.volumeRepresentation, metav1.CreateOptions{})
		if err != nil {
			t.Errorf("case %v: Error in Create, Expected  err to nil, got %v", i+1, err)
		}

		_, err = fakeDirectCSIVolumeAdapter.Get(ctx, testCase.newVolumeName, metav1.GetOptions{})
		if err != nil {
			t.Errorf("case %v: Error while fetching newly created drive, Expected  err to nil, got %v", i+1, err)
		}
	}
}
