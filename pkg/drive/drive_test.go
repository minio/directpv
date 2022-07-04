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

package drive

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/client"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	testNodeID = "test-node"
)

func createFakeDriveEventListener() *driveEventHandler {
	return &driveEventHandler{
		nodeID:                  testNodeID,
		getDevice:               func(major, minor uint32) (string, error) { return "", nil },
		stat:                    func(name string) (os.FileInfo, error) { return nil, nil },
		mountDevice:             func(device, target string, flags []string) error { return nil },
		unmountDevice:           func(device string) error { return nil },
		makeFS:                  func(ctx context.Context, device, uuid string, force, reflink bool) error { return nil },
		getFreeCapacity:         func(path string) (uint64, error) { return 0, nil },
		verifyHostStateForDrive: func(drive *directcsi.DirectCSIDrive) error { return nil },
		isMounted: func(target string) (bool, error) {
			return false, nil
		},
		safeUnmount: func(target string, force, detach, expire bool) error {
			return nil
		},
	}
}

func TestFormatHander(t *testing.T) {
	testDrive := &directcsi.DirectCSIDrive{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "2bf15006-a710-4c5f-8678-e3c996baaf2f",
		},
		Spec: directcsi.DirectCSIDriveSpec{
			DirectCSIOwned: true,
			RequestedFormat: &directcsi.RequestedFormat{
				Force:      true,
				Filesystem: "xfs",
			},
		},
		Status: directcsi.DirectCSIDriveStatus{
			NodeName:       "test_node",
			DriveStatus:    directcsi.DriveStatusAvailable,
			Path:           "/dev/sda",
			FilesystemUUID: "e5850478-95d8-4c64-a160-cc838eae50bd",
			Filesystem:     "xfs",
			MajorNumber:    202,
			MinorNumber:    1,
			Conditions: []metav1.Condition{
				{
					Type:               string(directcsi.DirectCSIDriveConditionOwned),
					Status:             metav1.ConditionFalse,
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionMounted),
					Status:             metav1.ConditionFalse,
					Message:            "not mounted",
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionFormatted),
					Status:             metav1.ConditionFalse,
					Message:            "xfs",
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionInitialized),
					Status:             metav1.ConditionTrue,
					Message:            "initialized",
					Reason:             string(directcsi.DirectCSIDriveReasonInitialized),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionReady),
					Status:             metav1.ConditionTrue,
					Message:            "",
					Reason:             string(directcsi.DirectCSIDriveReasonReady),
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}
	ctx := context.TODO()
	dl := createFakeDriveEventListener()

	var getDeviceCalled, isMountedCalled, statCalled, makeFSCalled, mountDeviceCalled bool
	dl.isMounted = func(target string) (bool, error) {
		isMountedCalled = true
		if target != filepath.Join(sys.MountRoot, testDrive.Name) {
			return false, fmt.Errorf(
				"expected target %s but got %s",
				filepath.Join(sys.MountRoot, testDrive.Name),
				target,
			)
		}
		return false, nil
	}
	dl.getDevice = func(major, minor uint32) (string, error) {
		getDeviceCalled = true
		if major != uint32(testDrive.Status.MajorNumber) {
			return "", fmt.Errorf("expected major: %v but got %v", testDrive.Status.MajorNumber, major)
		}
		if minor != uint32(testDrive.Status.MinorNumber) {
			return "", fmt.Errorf("expected minor: %v but got %v", testDrive.Status.MinorNumber, minor)
		}
		return "/dev/" + filepath.Base(testDrive.Status.Path), nil
	}
	dl.stat = func(name string) (os.FileInfo, error) {
		statCalled = true
		if name != testDrive.Status.Path {
			return nil, fmt.Errorf("expected name %s but got %s", testDrive.Status.Path, name)
		}
		return nil, nil
	}
	dl.makeFS = func(ctx context.Context, device, uuid string, force, reflink bool) error {
		makeFSCalled = true
		if device != testDrive.Status.Path {
			return fmt.Errorf("expected device %s but got %s", testDrive.Status.Path, device)
		}
		if uuid != testDrive.Name {
			return fmt.Errorf("expected fssuid for formatting %s but got %s", testDrive.Name, uuid)
		}
		if testDrive.Status.Filesystem != "" && !force {
			return fmt.Errorf("force is not enabled for formatting the drive with FS %s", testDrive.Status.Filesystem)
		}
		if reflink != dl.reflinkSupport {
			return fmt.Errorf("expected reflink %v but got %v", dl.reflinkSupport, reflink)
		}
		return nil
	}
	dl.mountDevice = func(device, target string, flags []string) error {
		mountDeviceCalled = true
		if device != testDrive.Status.Path {
			return fmt.Errorf("expected device %s but got %s", testDrive.Status.Path, device)
		}
		if target != filepath.Join(sys.MountRoot, testDrive.Name) {
			return fmt.Errorf("expected target %s but got %s", filepath.Join(sys.MountRoot, testDrive.Name), target)
		}
		return nil
	}
	if err := dl.handleUpdate(ctx, testDrive); err != nil {
		t.Fatalf("error while handling update %v", err)
	}
	if !getDeviceCalled {
		t.Error("getDevice function is not called")
	}
	if !isMountedCalled {
		t.Error("isMounted function is not called")
	}
	if !statCalled {
		t.Error("stat function is not called")
	}
	if !makeFSCalled {
		t.Error("makeFS function is not called")
	}
	if !mountDeviceCalled {
		t.Error("mountDevice function is not called")
	}
}

func TestFormatHanderWithError(t *testing.T) {
	testDrive := &directcsi.DirectCSIDrive{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "2bf15006-a710-4c5f-8678-e3c996baaf2f",
		},
		Spec: directcsi.DirectCSIDriveSpec{
			DirectCSIOwned: true,
			RequestedFormat: &directcsi.RequestedFormat{
				Force:      true,
				Filesystem: "xfs",
			},
		},
		Status: directcsi.DirectCSIDriveStatus{
			NodeName:       "test_node",
			DriveStatus:    directcsi.DriveStatusAvailable,
			Path:           "/dev/sda",
			FilesystemUUID: "e5850478-95d8-4c64-a160-cc838eae50bd",
			Filesystem:     "xfs",
			MajorNumber:    202,
			MinorNumber:    1,
			Conditions: []metav1.Condition{
				{
					Type:               string(directcsi.DirectCSIDriveConditionOwned),
					Status:             metav1.ConditionTrue,
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionMounted),
					Status:             metav1.ConditionFalse,
					Message:            "not mounted",
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionFormatted),
					Status:             metav1.ConditionFalse,
					Message:            "xfs",
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionInitialized),
					Status:             metav1.ConditionTrue,
					Message:            "initialized",
					Reason:             string(directcsi.DirectCSIDriveReasonInitialized),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionReady),
					Status:             metav1.ConditionTrue,
					Message:            "",
					Reason:             string(directcsi.DirectCSIDriveReasonReady),
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	ctx := context.TODO()
	dl := createFakeDriveEventListener()
	client.SetLatestDirectCSIDriveInterface(clientsetfake.NewSimpleClientset(testDrive).DirectV1beta4().DirectCSIDrives())

	var getDeviceCalled, isMountedCalled, statCalled, makeFSCalled bool
	dl.isMounted = func(target string) (bool, error) {
		isMountedCalled = true
		if target != filepath.Join(sys.MountRoot, testDrive.Name) {
			return false, fmt.Errorf(
				"expected target %s but got %s",
				filepath.Join(sys.MountRoot, testDrive.Name),
				target,
			)
		}
		return false, nil
	}
	dl.getDevice = func(major, minor uint32) (string, error) {
		getDeviceCalled = true
		if major != uint32(testDrive.Status.MajorNumber) {
			return "", fmt.Errorf("expected major: %v but got %v", testDrive.Status.MajorNumber, major)
		}
		if minor != uint32(testDrive.Status.MinorNumber) {
			return "", fmt.Errorf("expected minor: %v but got %v", testDrive.Status.MinorNumber, minor)
		}
		return "/dev/" + filepath.Base(testDrive.Status.Path), nil
	}
	dl.stat = func(name string) (os.FileInfo, error) {
		statCalled = true
		if name != testDrive.Status.Path {
			return nil, fmt.Errorf("expected name %s but got %s", testDrive.Status.Path, name)
		}
		return nil, nil
	}
	dl.makeFS = func(ctx context.Context, device, uuid string, force, reflink bool) error {
		makeFSCalled = true
		return errors.New("returning an error to test the failure scenario")
	}
	dl.mountDevice = func(device, target string, flags []string) error {
		return errors.New("mountDevice function shouldn't be called here")
	}
	if err := dl.handleUpdate(ctx, testDrive); err == nil {
		t.Error("expected handleUpdate to fail but returned with no error")
	}
	if !getDeviceCalled {
		t.Error("getDevice function is not called")
	}
	if !isMountedCalled {
		t.Error("isMounted function is not called")
	}
	if !statCalled {
		t.Error("stat function is not called")
	}
	if !makeFSCalled {
		t.Error("makeFS function is not called")
	}
	updatedDrive, dErr := client.GetLatestDirectCSIDriveInterface().Get(ctx, testDrive.Name, metav1.GetOptions{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
	})
	if dErr != nil {
		t.Errorf("Error while fetching the drive object: %+v", dErr)
	}
	if utils.IsConditionStatus(
		updatedDrive.Status.Conditions,
		string(directcsi.DirectCSIDriveConditionOwned),
		metav1.ConditionTrue) {
		t.Error("drive condition is not set to false after error")
	}
}

func TestReleaseHander(t *testing.T) {
	testDrive := &directcsi.DirectCSIDrive{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "2bf15006-a710-4c5f-8678-e3c996baaf2f",
		},
		Status: directcsi.DirectCSIDriveStatus{
			NodeName:       "test_node",
			DriveStatus:    directcsi.DriveStatusReleased,
			Path:           "/dev/sda",
			FilesystemUUID: "e5850478-95d8-4c64-a160-cc838eae50bd",
			Filesystem:     "xfs",
			MajorNumber:    202,
			MinorNumber:    1,
			Mountpoint:     "/var/lib/direct-csi/mnt/2bf15006-a710-4c5f-8678-e3c996baaf2f",
			Conditions: []metav1.Condition{
				{
					Type:               string(directcsi.DirectCSIDriveConditionOwned),
					Status:             metav1.ConditionFalse,
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionMounted),
					Status:             metav1.ConditionFalse,
					Message:            "not mounted",
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionFormatted),
					Status:             metav1.ConditionFalse,
					Message:            "xfs",
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionInitialized),
					Status:             metav1.ConditionTrue,
					Message:            "initialized",
					Reason:             string(directcsi.DirectCSIDriveReasonInitialized),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionReady),
					Status:             metav1.ConditionTrue,
					Message:            "",
					Reason:             string(directcsi.DirectCSIDriveReasonReady),
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}
	ctx := context.TODO()
	dl := createFakeDriveEventListener()

	var getDeviceCalled, unmountDeviceCalled bool
	dl.getDevice = func(major, minor uint32) (string, error) {
		getDeviceCalled = true
		if major != uint32(testDrive.Status.MajorNumber) {
			return "", fmt.Errorf("expected major: %v but got %v", testDrive.Status.MajorNumber, major)
		}
		if minor != uint32(testDrive.Status.MinorNumber) {
			return "", fmt.Errorf("expected minor: %v but got %v", testDrive.Status.MinorNumber, minor)
		}
		return "/dev/" + filepath.Base(testDrive.Status.Path), nil
	}
	dl.unmountDevice = func(device string) error {
		unmountDeviceCalled = true
		if device != testDrive.Status.Path {
			return fmt.Errorf("expected device %s but got %s", testDrive.Status.Path, device)
		}
		return nil
	}
	if err := dl.handleUpdate(ctx, testDrive); err != nil {
		t.Fatalf("error while handling update %v", err)
	}
	if !getDeviceCalled {
		t.Error("getDevice function is not called")
	}
	if !unmountDeviceCalled {
		t.Error("unmountDeviceCalled function is not called")
	}
}

func TestReleaseHanderWithError(t *testing.T) {
	testDrive := &directcsi.DirectCSIDrive{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "2bf15006-a710-4c5f-8678-e3c996baaf2f",
		},
		Status: directcsi.DirectCSIDriveStatus{
			NodeName:       "test_node",
			DriveStatus:    directcsi.DriveStatusReleased,
			Path:           "/dev/sda",
			FilesystemUUID: "e5850478-95d8-4c64-a160-cc838eae50bd",
			Filesystem:     "xfs",
			MajorNumber:    202,
			MinorNumber:    1,
			Mountpoint:     "/var/lib/direct-csi/mnt/2bf15006-a710-4c5f-8678-e3c996baaf2f",
			Conditions: []metav1.Condition{
				{
					Type:               string(directcsi.DirectCSIDriveConditionOwned),
					Status:             metav1.ConditionTrue,
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionMounted),
					Status:             metav1.ConditionFalse,
					Message:            "not mounted",
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionFormatted),
					Status:             metav1.ConditionFalse,
					Message:            "xfs",
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionInitialized),
					Status:             metav1.ConditionTrue,
					Message:            "initialized",
					Reason:             string(directcsi.DirectCSIDriveReasonInitialized),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionReady),
					Status:             metav1.ConditionTrue,
					Message:            "",
					Reason:             string(directcsi.DirectCSIDriveReasonReady),
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}
	ctx := context.TODO()
	dl := createFakeDriveEventListener()
	client.SetLatestDirectCSIDriveInterface(clientsetfake.NewSimpleClientset(testDrive).DirectV1beta4().DirectCSIDrives())

	var getDeviceCalled, unmountDeviceCalled bool
	dl.getDevice = func(major, minor uint32) (string, error) {
		getDeviceCalled = true
		if major != uint32(testDrive.Status.MajorNumber) {
			return "", fmt.Errorf("expected major: %v but got %v", testDrive.Status.MajorNumber, major)
		}
		if minor != uint32(testDrive.Status.MinorNumber) {
			return "", fmt.Errorf("expected minor: %v but got %v", testDrive.Status.MinorNumber, minor)
		}
		return "/dev/" + filepath.Base(testDrive.Status.Path), nil
	}
	dl.unmountDevice = func(device string) error {
		unmountDeviceCalled = true
		if device != testDrive.Status.Path {
			return fmt.Errorf("expected device %s but got %s", testDrive.Status.Path, device)
		}
		return errors.New("returning error here to test error scenario")
	}
	if err := dl.handleUpdate(ctx, testDrive); err == nil {
		t.Error("expected handleUpdate to fail but returned with no error")
	}
	if !getDeviceCalled {
		t.Error("getDevice function is not called")
	}
	if !unmountDeviceCalled {
		t.Error("unmountDeviceCalled function is not called")
	}
	updatedDrive, dErr := client.GetLatestDirectCSIDriveInterface().Get(ctx, testDrive.Name, metav1.GetOptions{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
	})
	if dErr != nil {
		t.Errorf("Error while fetching the drive object: %+v", dErr)
	}
	if utils.IsConditionStatus(
		updatedDrive.Status.Conditions,
		string(directcsi.DirectCSIDriveConditionOwned),
		metav1.ConditionTrue) {
		t.Error("drive condition is not set to false after error")
	}
}

func TestMountDriveHander(t *testing.T) {
	testDrive := &directcsi.DirectCSIDrive{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "2bf15006-a710-4c5f-8678-e3c996baaf2f",
		},
		Status: directcsi.DirectCSIDriveStatus{
			NodeName:       "test_node",
			DriveStatus:    directcsi.DriveStatusReleased,
			Path:           "/dev/sda",
			FilesystemUUID: "e5850478-95d8-4c64-a160-cc838eae50bd",
			Filesystem:     "xfs",
			MajorNumber:    202,
			MinorNumber:    1,
			Mountpoint:     "/var/lib/direct-csi/mnt/2bf15006-a710-4c5f-8678-e3c996baaf2f",
			Conditions: []metav1.Condition{
				{
					Type:               string(directcsi.DirectCSIDriveConditionOwned),
					Status:             metav1.ConditionTrue,
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionMounted),
					Status:             metav1.ConditionFalse,
					Message:            "not mounted",
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionFormatted),
					Status:             metav1.ConditionFalse,
					Message:            "xfs",
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionInitialized),
					Status:             metav1.ConditionTrue,
					Message:            "initialized",
					Reason:             string(directcsi.DirectCSIDriveReasonInitialized),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionReady),
					Status:             metav1.ConditionTrue,
					Message:            "",
					Reason:             string(directcsi.DirectCSIDriveReasonReady),
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}
	ctx := context.TODO()
	dl := createFakeDriveEventListener()

	// MountDrive test
	var getDeviceCalled, mountDeviceCalled, safeUnmountCalled bool
	dl.getDevice = func(major, minor uint32) (string, error) {
		getDeviceCalled = true
		if major != uint32(testDrive.Status.MajorNumber) {
			return "", fmt.Errorf("expected major: %v but got %v", testDrive.Status.MajorNumber, major)
		}
		if minor != uint32(testDrive.Status.MinorNumber) {
			return "", fmt.Errorf("expected minor: %v but got %v", testDrive.Status.MinorNumber, minor)
		}
		return "/dev/" + filepath.Base(testDrive.Status.Path), nil
	}
	dl.mountDevice = func(device, target string, flags []string) error {
		mountDeviceCalled = true
		if device != testDrive.Status.Path {
			return fmt.Errorf("expected device %s but got %s", testDrive.Status.Path, device)
		}
		if target != filepath.Join(sys.MountRoot, testDrive.Name) {
			return fmt.Errorf("expected target %s but got %s", filepath.Join(sys.MountRoot, testDrive.Name), target)
		}
		return nil
	}
	dl.safeUnmount = func(target string, force, detach, expire bool) error {
		safeUnmountCalled = true
		return nil
	}
	dl.verifyHostStateForDrive = func(drive *directcsi.DirectCSIDrive) error {
		return errNotMounted
	}
	if err := dl.handleUpdate(ctx, testDrive); err != nil {
		t.Fatalf("error while handling update %v", err)
	}
	if !getDeviceCalled {
		t.Error("getDevice function is not called")
	}
	if safeUnmountCalled {
		t.Error("safeUnmount function is not expected to be called")
	}
	if !mountDeviceCalled {
		t.Error("mountDeviceCalled function is not called")
	}

	// MountDrive with error
	client.SetLatestDirectCSIDriveInterface(clientsetfake.NewSimpleClientset(testDrive).DirectV1beta4().DirectCSIDrives())
	getDeviceCalled = false
	mountDeviceCalled = false
	dl.mountDevice = func(device, target string, flags []string) error {
		mountDeviceCalled = true
		return errors.New("returning an error to test the failure scenario")
	}
	dl.safeUnmount = func(target string, force, detach, expire bool) error {
		safeUnmountCalled = true
		return nil
	}
	if err := dl.handleUpdate(ctx, testDrive); err == nil {
		t.Error("expected handleUpdate to fail but returned with no error")
	}
	if !getDeviceCalled {
		t.Error("getDevice function is not called")
	}
	if safeUnmountCalled {
		t.Error("safeUnmount function is not expected to be called")
	}
	if !mountDeviceCalled {
		t.Error("mountDeviceCalled function is not called")
	}
	updatedDrive, dErr := client.GetLatestDirectCSIDriveInterface().Get(ctx, testDrive.Name, metav1.GetOptions{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
	})
	if dErr != nil {
		t.Errorf("Error while fetching the drive object: %+v", dErr)
	}
	if utils.IsConditionStatus(
		updatedDrive.Status.Conditions,
		string(directcsi.DirectCSIDriveConditionOwned),
		metav1.ConditionTrue) {
		t.Error("drive condition is not set to false after error")
	}
}

func TestRemountDriveHander(t *testing.T) {
	testDrive := &directcsi.DirectCSIDrive{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "2bf15006-a710-4c5f-8678-e3c996baaf2f",
		},
		Status: directcsi.DirectCSIDriveStatus{
			NodeName:       "test_node",
			DriveStatus:    directcsi.DriveStatusReleased,
			Path:           "/dev/sda",
			FilesystemUUID: "e5850478-95d8-4c64-a160-cc838eae50bd",
			Filesystem:     "xfs",
			MajorNumber:    202,
			MinorNumber:    1,
			Mountpoint:     "/var/lib/direct-csi/mnt/2bf15006-a710-4c5f-8678-e3c996baaf2f",
			Conditions: []metav1.Condition{
				{
					Type:               string(directcsi.DirectCSIDriveConditionOwned),
					Status:             metav1.ConditionTrue,
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionMounted),
					Status:             metav1.ConditionFalse,
					Message:            "not mounted",
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionFormatted),
					Status:             metav1.ConditionFalse,
					Message:            "xfs",
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionInitialized),
					Status:             metav1.ConditionTrue,
					Message:            "initialized",
					Reason:             string(directcsi.DirectCSIDriveReasonInitialized),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionReady),
					Status:             metav1.ConditionTrue,
					Message:            "",
					Reason:             string(directcsi.DirectCSIDriveReasonReady),
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}
	ctx := context.TODO()
	dl := createFakeDriveEventListener()

	var getDeviceCalled, mountDeviceCalled, safeUnmountCalled bool
	dl.getDevice = func(major, minor uint32) (string, error) {
		getDeviceCalled = true
		if major != uint32(testDrive.Status.MajorNumber) {
			return "", fmt.Errorf("expected major: %v but got %v", testDrive.Status.MajorNumber, major)
		}
		if minor != uint32(testDrive.Status.MinorNumber) {
			return "", fmt.Errorf("expected minor: %v but got %v", testDrive.Status.MinorNumber, minor)
		}
		return "/dev/" + filepath.Base(testDrive.Status.Path), nil
	}
	dl.mountDevice = func(device, target string, flags []string) error {
		mountDeviceCalled = true
		if device != testDrive.Status.Path {
			return fmt.Errorf("expected device %s but got %s", testDrive.Status.Path, device)
		}
		if target != filepath.Join(sys.MountRoot, testDrive.Name) {
			return fmt.Errorf("expected target %s but got %s", filepath.Join(sys.MountRoot, testDrive.Name), target)
		}
		return nil
	}
	dl.safeUnmount = func(target string, force, detach, expire bool) error {
		safeUnmountCalled = true
		if target != testDrive.Status.Mountpoint {
			return fmt.Errorf("expected target %s but got %s", testDrive.Status.Mountpoint, target)
		}
		return nil
	}

	dl.verifyHostStateForDrive = func(drive *directcsi.DirectCSIDrive) error {
		return errInvalidMountOptions
	}
	if err := dl.handleUpdate(ctx, testDrive); err != nil {
		t.Fatalf("error while handling update %v", err)
	}
	if !getDeviceCalled {
		t.Error("getDevice function is not called")
	}
	if !mountDeviceCalled {
		t.Error("mountDevice function is not called")
	}
	if !safeUnmountCalled {
		t.Error("safeUnmount function is not called")
	}

	// remountDrive with error
	client.SetLatestDirectCSIDriveInterface(clientsetfake.NewSimpleClientset(testDrive).DirectV1beta4().DirectCSIDrives())
	getDeviceCalled = false
	mountDeviceCalled = false
	safeUnmountCalled = false
	dl.safeUnmount = func(target string, force, detach, expire bool) error {
		safeUnmountCalled = true
		return errors.New("returning an error to test the failure scenario")
	}
	dl.mountDevice = func(device, target string, flags []string) error {
		mountDeviceCalled = true
		return errors.New("shouldn't be called if umount is failing")
	}
	if err := dl.handleUpdate(ctx, testDrive); err == nil {
		t.Error("expected handleUpdate to fail but returned with no error")
	}
	if !getDeviceCalled {
		t.Error("getDevice function is not called")
	}
	if !safeUnmountCalled {
		t.Error("safeUnmount function is not called")
	}
	if mountDeviceCalled {
		t.Error("mountDevice function shouldn't be called if umount is failing")
	}
	updatedDrive, dErr := client.GetLatestDirectCSIDriveInterface().Get(ctx, testDrive.Name, metav1.GetOptions{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
	})
	if dErr != nil {
		t.Errorf("Error while fetching the drive object: %+v", dErr)
	}
	if utils.IsConditionStatus(
		updatedDrive.Status.Conditions,
		string(directcsi.DirectCSIDriveConditionOwned),
		metav1.ConditionTrue) {
		t.Error("drive condition is not set to false after error")
	}
}

func TestDriveLostHandler(t *testing.T) {
	testDriveObject := &directcsi.DirectCSIDrive{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "2bf15006-a710-4c5f-8678-e3c996baaf2f",
			Finalizers: []string{
				string(directcsi.DirectCSIDriveFinalizerDataProtection),
				directcsi.DirectCSIDriveFinalizerPrefix + "test_volume_1",
				directcsi.DirectCSIDriveFinalizerPrefix + "test_volume_2",
			},
		},
		Status: directcsi.DirectCSIDriveStatus{
			NodeName:    "test_node",
			DriveStatus: directcsi.DriveStatusInUse,
			Conditions: []metav1.Condition{
				{
					Type:               string(directcsi.DirectCSIDriveConditionOwned),
					Status:             metav1.ConditionTrue,
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionMounted),
					Status:             metav1.ConditionTrue,
					Message:            "mounted",
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionFormatted),
					Status:             metav1.ConditionTrue,
					Message:            "xfs",
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionInitialized),
					Status:             metav1.ConditionTrue,
					Message:            "initialized",
					Reason:             string(directcsi.DirectCSIDriveReasonInitialized),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionReady),
					Status:             metav1.ConditionTrue,
					Message:            "",
					Reason:             string(directcsi.DirectCSIDriveReasonReady),
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}
	testVolumeObjects := []runtime.Object{
		&directcsi.DirectCSIVolume{
			TypeMeta: utils.DirectCSIVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test_volume_1",
				Finalizers: []string{
					string(directcsi.DirectCSIVolumeFinalizerPurgeProtection),
				},
			},
			Status: directcsi.DirectCSIVolumeStatus{
				NodeName: "test_node",
				HostPath: "hostpath",
				Drive:    "2bf15006-a710-4c5f-8678-e3c996baaf2f",
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIVolumeConditionStaged),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIVolumeReasonInUse),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directcsi.DirectCSIVolumeConditionPublished),
						Status:             metav1.ConditionFalse,
						Message:            "",
						Reason:             string(directcsi.DirectCSIVolumeReasonNotInUse),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directcsi.DirectCSIVolumeConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIVolumeReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
		&directcsi.DirectCSIVolume{
			TypeMeta: utils.DirectCSIVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test_volume_2",
				Finalizers: []string{
					string(directcsi.DirectCSIVolumeFinalizerPurgeProtection),
				},
			},
			Status: directcsi.DirectCSIVolumeStatus{
				NodeName: "test_node",
				HostPath: "hostpath",
				Drive:    "2bf15006-a710-4c5f-8678-e3c996baaf2f",
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIVolumeConditionStaged),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIVolumeReasonInUse),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directcsi.DirectCSIVolumeConditionPublished),
						Status:             metav1.ConditionFalse,
						Message:            "",
						Reason:             string(directcsi.DirectCSIVolumeReasonNotInUse),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directcsi.DirectCSIVolumeConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIVolumeReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
	}

	ctx := context.TODO()
	dl := createFakeDriveEventListener()
	client.SetLatestDirectCSIDriveInterface(clientsetfake.NewSimpleClientset(testDriveObject).DirectV1beta4().DirectCSIDrives())
	client.SetLatestDirectCSIVolumeInterface(clientsetfake.NewSimpleClientset(testVolumeObjects...).DirectV1beta4().DirectCSIVolumes())
	dl.verifyHostStateForDrive = func(drive *directcsi.DirectCSIDrive) error {
		return os.ErrNotExist
	}
	if err := dl.handleUpdate(ctx, testDriveObject); err != nil {
		t.Fatalf("error while handling lost drive: %v", err)
	}

	updatedDrive, dErr := client.GetLatestDirectCSIDriveInterface().Get(ctx, testDriveObject.Name, metav1.GetOptions{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
	})
	if dErr != nil {
		t.Errorf("Error while fetching the drive object: %+v", dErr)
	}

	if !utils.IsCondition(updatedDrive.Status.Conditions,
		string(directcsi.DirectCSIDriveConditionReady),
		metav1.ConditionFalse,
		string(directcsi.DirectCSIDriveReasonLost),
		string(directcsi.DirectCSIDriveMessageLost),
	) {
		t.Error("invalid drive ready condition")
	}

	result, err := client.GetLatestDirectCSIVolumeInterface().List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("could not list drives: %v", err)
	}
	if len(result.Items) != 2 {
		t.Error("unexpected volume count")
	}
	for _, volume := range result.Items {
		if !utils.IsCondition(volume.Status.Conditions,
			string(directcsi.DirectCSIVolumeConditionReady),
			metav1.ConditionFalse,
			string(directcsi.DirectCSIVolumeReasonDriveLost),
			string(directcsi.DirectCSIVolumeMessageDriveLost),
		) {
			t.Fatalf("invalid ready status condition for volume: %s", volume.Name)
		}
	}
}

func TestInvalidatedDrive(t *testing.T) {
	testDriveObject := &directcsi.DirectCSIDrive{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "2bf15006-a710-4c5f-8678-e3c996baaf2f",
			Finalizers: []string{
				string(directcsi.DirectCSIDriveFinalizerDataProtection),
				directcsi.DirectCSIDriveFinalizerPrefix + "test_volume_1",
				directcsi.DirectCSIDriveFinalizerPrefix + "test_volume_2",
			},
		},
		Status: directcsi.DirectCSIDriveStatus{
			NodeName:    "test_node",
			DriveStatus: directcsi.DriveStatusReady,
			Conditions: []metav1.Condition{
				{
					Type:               string(directcsi.DirectCSIDriveConditionOwned),
					Status:             metav1.ConditionTrue,
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionMounted),
					Status:             metav1.ConditionTrue,
					Message:            "mounted",
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionFormatted),
					Status:             metav1.ConditionTrue,
					Message:            "xfs",
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionInitialized),
					Status:             metav1.ConditionTrue,
					Message:            "initialized",
					Reason:             string(directcsi.DirectCSIDriveReasonInitialized),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionReady),
					Status:             metav1.ConditionTrue,
					Message:            "",
					Reason:             string(directcsi.DirectCSIDriveReasonReady),
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	ctx := context.TODO()
	dl := createFakeDriveEventListener()
	client.SetLatestDirectCSIDriveInterface(clientsetfake.NewSimpleClientset(testDriveObject).DirectV1beta4().DirectCSIDrives())
	dl.verifyHostStateForDrive = func(drive *directcsi.DirectCSIDrive) error {
		return errors.New("mismatch error")
	}
	if err := dl.handleUpdate(ctx, testDriveObject); err == nil {
		t.Fatal("expected error but got nil")
	}

	updatedDrive, dErr := client.GetLatestDirectCSIDriveInterface().Get(ctx, testDriveObject.Name, metav1.GetOptions{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
	})
	if dErr != nil {
		t.Errorf("Error while fetching the drive object: %+v", dErr)
	}

	if !utils.IsCondition(updatedDrive.Status.Conditions,
		string(directcsi.DirectCSIDriveConditionReady),
		metav1.ConditionFalse,
		string(directcsi.DirectCSIDriveReasonNotReady),
		string(directcsi.DirectCSIDriveMessageNotReady),
	) {
		t.Error("wrong drive ready condition")
	}
}
