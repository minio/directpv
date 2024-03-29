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

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	versionV1Beta1 = consts.GroupName + "/v1beta1"
)

var supportedVersions = []string{
	versionV1Beta1,
} // ordered

type crdKind string

const (
	driveCRDKind       crdKind = consts.DriveKind
	volumeCRDKind      crdKind = consts.VolumeKind
	nodeCRDKind        crdKind = consts.NodeKind
	initRequestCRDKind crdKind = consts.InitRequestKind
)

type migrateFunc func(object *unstructured.Unstructured, toVersion string) error

var errUnsupportedCRDKind = errors.New("unsupported CRD Kind")

// MigrateList migrate the list to the provided group version
func MigrateList(fromList, toList *unstructured.UnstructuredList, groupVersion schema.GroupVersion) error {
	fromList.DeepCopyInto(toList)
	fn := func(obj runtime.Object) error {
		cpObj := obj.DeepCopyObject()
		return Migrate(cpObj.(*unstructured.Unstructured), obj.(*unstructured.Unstructured), groupVersion)
	}
	toList.SetAPIVersion(groupVersion.Group + "/" + groupVersion.Version)
	return toList.EachListItem(fn)
}

// Migrate function migrates unstructured object from one to another
func Migrate(from, to *unstructured.Unstructured, groupVersion schema.GroupVersion) error {
	obj := from.DeepCopy()
	if from.GetAPIVersion() == groupVersion.String() {
		from.DeepCopyInto(to)
		return nil
	}
	toVersion := groupVersion.Group + "/" + groupVersion.Version
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
	if _, ok := labels[string(directpvtypes.VersionLabelKey)]; !ok {
		labels[string(directpvtypes.VersionLabelKey)] = filepath.Base(fromVersion)
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
	case nodeCRDKind:
		return upgradeNodeObject(object, toVersion)
	case initRequestCRDKind:
		return upgradeInitRequestObject(object, toVersion)
	}
	return errUnsupportedCRDKind
}

func downgradeObject(object *unstructured.Unstructured, toVersion string) error {
	switch getCRDKind(object) {
	case driveCRDKind:
		return downgradeDriveObject(object, toVersion)
	case volumeCRDKind:
		return downgradeVolumeObject(object, toVersion)
	case nodeCRDKind:
		return downgradeNodeObject(object, toVersion)
	case initRequestCRDKind:
		return downgradeInitRequestObject(object, toVersion)
	}
	return errUnsupportedCRDKind
}
