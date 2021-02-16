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
	directv1beta1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	versionV1Alpha1 = "direct.csi.min.io/v1alpha1"
	versionV1Beta1  = "direct.csi.min.io/v1beta1"
)

var (
	supportedVersions = []string{versionV1Alpha1,
		versionV1Beta1} //ordered
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
		if err := upgradeV1alpha1ToV1Beta1(convertedObject); err != nil {
			return err
		}
		fallthrough
	case versionV1Beta1:
		if toVersion == versionV1Beta1 {
			glog.V(2).Info("Successfully migrated")
			break
		}
	}
	return nil
}

func upgradeV1alpha1ToV1Beta1(unstructured *unstructured.Unstructured) error {

	unstructuredObject := unstructured.Object

	var v1alpha1DirectCSIDrive directv1alpha1.DirectCSIDrive
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject, &v1alpha1DirectCSIDrive)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("Converting %v to v1beta1", v1alpha1DirectCSIDrive.Name)

	var v1beta1DirectCSIDrive directv1beta1.DirectCSIDrive
	if err := directv1beta1.Convert_v1alpha1_DirectCSIDrive_To_v1beta1_DirectCSIDrive(&v1alpha1DirectCSIDrive, &v1beta1DirectCSIDrive, nil); err != nil {
		return err
	}

	v1beta1DirectCSIDrive.TypeMeta = v1alpha1DirectCSIDrive.TypeMeta
	v1beta1DirectCSIDrive.Status.AccessTier = directv1beta1.AccessTierUnknown
	convertedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&v1beta1DirectCSIDrive)
	if err != nil {
		return err
	}

	unstructured.Object = convertedObj
	return nil
}

func downgradeObject(fromVersion, toVersion string, convertedObject *unstructured.Unstructured) error {
	switch fromVersion {
	case versionV1Beta1:
		if err := downgradeV1Beta1ToV1alpha1(convertedObject); err != nil {
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

func downgradeV1Beta1ToV1alpha1(unstructured *unstructured.Unstructured) error {

	unstructuredObject := unstructured.Object

	var v1beta1DirectCSIDrive directv1beta1.DirectCSIDrive
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject, &v1beta1DirectCSIDrive); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("Converting %v to v1alpha1", v1beta1DirectCSIDrive.Name)

	var v1alpha1DirectCSIDrive directv1alpha1.DirectCSIDrive
	if err := directv1beta1.Convert_v1beta1_DirectCSIDrive_To_v1alpha1_DirectCSIDrive(&v1beta1DirectCSIDrive, &v1alpha1DirectCSIDrive, nil); err != nil {
		return err
	}

	v1alpha1DirectCSIDrive.TypeMeta = v1beta1DirectCSIDrive.TypeMeta
	convertedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(v1alpha1DirectCSIDrive)
	if err != nil {
		return err
	}

	unstructured.Object = convertedObj
	return nil
}
