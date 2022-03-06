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

package converter

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type migrateFunc func(object *unstructured.Unstructured, toVersion string) error

var (
	errUnsupportedCRDKind = errors.New("unsupported CRD Kind")
)

func convertDriveCRD(Object *unstructured.Unstructured, toVersion string) (*unstructured.Unstructured, metav1.Status) {
	convertedObject := Object.DeepCopy()
	if err := migrate(convertedObject, toVersion); err != nil {
		return nil, statusErrorWithMessage(err.Error())
	}
	return convertedObject, statusSucceed()
}

func convertVolumeCRD(Object *unstructured.Unstructured, toVersion string) (*unstructured.Unstructured, metav1.Status) {
	convertedObject := Object.DeepCopy()
	if err := migrate(convertedObject, toVersion); err != nil {
		return nil, statusErrorWithMessage(err.Error())
	}
	return convertedObject, statusSucceed()
}

func MigrateList(fromList, toList *unstructured.UnstructuredList, groupVersion schema.GroupVersion) error {
	fromList.DeepCopyInto(toList)
	fn := func(obj runtime.Object) error {
		cpObj := obj.DeepCopyObject()
		if err := Migrate(cpObj.(*unstructured.Unstructured), obj.(*unstructured.Unstructured), groupVersion); err != nil {
			return err
		}
		return nil
	}
	toList.SetAPIVersion(fmt.Sprintf("%s/%s", groupVersion.Group, groupVersion.Version))
	return toList.EachListItem(fn)
}

// Migrate function migrates unstructured object from one to another
func Migrate(from, to *unstructured.Unstructured, groupVersion schema.GroupVersion) error {
	obj := from.DeepCopy()
	if from.GetAPIVersion() == groupVersion.String() {
		from.DeepCopyInto(to)
		return nil
	}
	toVersion := fmt.Sprintf("%s/%s", groupVersion.Group, groupVersion.Version)
	if err := migrate(obj, toVersion); err != nil {
		return err
	}
	return runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, to)
}

func migrate(object *unstructured.Unstructured, toVersion string) error {
	fromVersion := object.GetAPIVersion()
	migrateFn, err := getMigrateFunc(fromVersion, toVersion)
	if err != nil {
		return err
	}

	// migrate the CRDs
	if err := migrateFn(object, toVersion); err != nil {
		return err
	}
	object.SetAPIVersion(toVersion)

	labels := object.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	if _, ok := labels[directcsi.Group+"/version"]; !ok {
		labels[directcsi.Group+"/version"] = filepath.Base(fromVersion)
	}
	object.SetLabels(labels)

	return nil
}

func getMigrateFunc(fromVersion, toVersion string) (migrateFunc, error) {
	getIndex := func(version string) int {
		for i := range supportedVersions {
			if supportedVersions[i] == version {
				return i
			}
		}
		return -1
	}

	fromIndex := getIndex(fromVersion)
	if fromIndex == -1 {
		return nil, fmt.Errorf("invalid fromVersion: %s", fromVersion)
	}

	toIndex := getIndex(toVersion)
	if toIndex == -1 {
		return nil, fmt.Errorf("invalid toVersion: %s", toVersion)
	}

	if fromIndex == toIndex {
		return nil, fmt.Errorf("conversion from a version to itself should not call the webhook: %s", toVersion)
	}

	if fromIndex > toIndex {
		return downgradeObject, nil
	}

	return upgradeObject, nil
}

func getCRDKind(object *unstructured.Unstructured) crdKind {
	return crdKind(strings.ReplaceAll(object.GetKind(), " ", ""))
}

func upgradeObject(object *unstructured.Unstructured, toVersion string) error {
	switch getCRDKind(object) {
	case driveCRDKind:
		return upgradeDriveObject(object, toVersion)
	case volumeCRDKind:
		return upgradeVolumeObject(object, toVersion)
	}
	return errUnsupportedCRDKind
}

func downgradeObject(object *unstructured.Unstructured, toVersion string) error {
	switch getCRDKind(object) {
	case driveCRDKind:
		return downgradeDriveObject(object, toVersion)
	case volumeCRDKind:
		return downgradeVolumeObject(object, toVersion)
	}
	return errUnsupportedCRDKind
}
