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

package converter

import (
	"fmt"
	"github.com/golang/glog"

	directv1alpha1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	versionV1Alpha1 = "direct.csi.min.io/v1alpha1"
	versionV1Test   = "direct.csi.min.io/v1test"
)

var (
	supportedVersions = []string{versionV1Alpha1,
		versionV1Test} //ordered
)

type migrateFunc func(fromVersion, toVersion string, Object *unstructured.Unstructured) error

func convertDriveCRD(Object *unstructured.Unstructured, toVersion string) (*unstructured.Unstructured, metav1.Status) {
	glog.V(2).Info("converting crd")

	convertedObject := Object.DeepCopy()
	fromVersion := Object.GetAPIVersion()

	migrateFn, err := getMigrateFunc(fromVersion, toVersion)
	if err != nil {
		return nil, statusErrorWithMessage(err.Error())
	}

	// migrate the CRDs
	migrateFn(fromVersion, toVersion, convertedObject)

	return convertedObject, statusSucceed()
}

func getMigrateFunc(fromVersion, toVersion string) (migrateFunc, error) {
	var migrateFn migrateFunc
	getIndex := func(version string) int {
		for i := range supportedVersions {
			if supportedVersions[i] == version {
				return i
			}
		}
		return -1
	}

	shouldUpgrade := func() (bool, error) {
		fromIndex := getIndex(fromVersion)
		if fromIndex == -1 {
			return false, fmt.Errorf("Invalid fromVersion: %s", fromVersion)
		}

		toIndex := getIndex(toVersion)
		if toIndex == -1 {
			return false, fmt.Errorf("Invalid toVersion: %s", toVersion)
		}

		if fromIndex == toIndex {
			return false, fmt.Errorf("conversion from a version to itself should not call the webhook: %s", toVersion)
		}

		if fromIndex > toIndex {
			return false, nil
		}
		return true, nil
	}

	upgrade, err := shouldUpgrade()
	if err != nil {
		return migrateFn, err
	}

	migrateFn = func() migrateFunc {
		if upgrade {
			return upgradeObject
		}
		return downgradeObject
	}()

	return migrateFn, nil
}

func upgradeObject(fromVersion, toVersion string, convertedObject *unstructured.Unstructured) error {
	switch fromVersion {
	case versionV1Alpha1:
		if err := upgradeV1alpha1ToV1test(convertedObject); err != nil {
			return err
		}
		fallthrough
	case versionV1Test:
		if toVersion == versionV1Test {
			glog.V(2).Info("Successfully migrated")
			break
		}
	}
	return nil
}

func upgradeV1alpha1ToV1test(unstructured *unstructured.Unstructured) error {

	unstructuredObject := unstructured.Object

	var directCSIDrive directv1alpha1.DirectCSIDrive
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject, &directCSIDrive)
	if err != nil {
		return err
	}

	directCSIDrive.Status.DriveStatus = directv1alpha1.DriveStatusUnavailable
	convertedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&directCSIDrive)
	if err != nil {
		return err
	}

	unstructured.Object = convertedObj
	return nil
}

func downgradeObject(fromVersion, toVersion string, convertedObject *unstructured.Unstructured) error {
	switch fromVersion {
	case versionV1Test:
		if err := downgradeV1testToV1alpha1(convertedObject); err != nil {
			return err
		}
		fallthrough
	case versionV1Alpha1:
		if toVersion == versionV1Alpha1 {
			glog.V(2).Info("Successfully migrated")
			break
		}
	}
	return nil
}

func downgradeV1testToV1alpha1(unstructured *unstructured.Unstructured) error {

	unstructuredObject := unstructured.Object

	var directCSIDrive directv1alpha1.DirectCSIDrive
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject, &directCSIDrive); err != nil {
		return err
	}

	directCSIDrive.Status.DriveStatus = directv1alpha1.DriveStatusAvailable
	convertedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(directCSIDrive)
	if err != nil {
		return err
	}

	unstructured.Object = convertedObj
	return nil
}
