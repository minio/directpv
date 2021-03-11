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
	"errors"
	"fmt"
	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type migrateFunc func(fromVersion, toVersion string, Object *unstructured.Unstructured, crdKind CRDKind) error

var (
	ErrCRDKindNotSupported = errors.New("Unsupported CRD Kind")
)

func convertDriveCRD(Object *unstructured.Unstructured, toVersion string) (*unstructured.Unstructured, metav1.Status) {
	glog.V(2).Info("converting drive crd")

	fmt.Println()
	fmt.Printf("Converting drive crd kind: %s", Object.GetKind()) 

	convertedObject := Object.DeepCopy()
	fromVersion := Object.GetAPIVersion()

	migrateFn, err := getMigrateFunc(fromVersion, toVersion)
	if err != nil {
		return nil, statusErrorWithMessage(err.Error())
	}

	// migrate the CRDs
	if err := migrateFn(fromVersion, toVersion, convertedObject, DriveCRDKind); err != nil {
		return nil, statusErrorWithMessage(err.Error())
	}

	return convertedObject, statusSucceed()
}

func convertVolumeCRD(Object *unstructured.Unstructured, toVersion string) (*unstructured.Unstructured, metav1.Status) {
	glog.V(2).Info("converting volume crd")

	fmt.Println()
	fmt.Printf("Converting volume crd kind: %s", Object.GetKind()) 

	convertedObject := Object.DeepCopy()
	fromVersion := Object.GetAPIVersion()

	migrateFn, err := getMigrateFunc(fromVersion, toVersion)
	if err != nil {
		return nil, statusErrorWithMessage(err.Error())
	}

	// migrate the CRDs
	if err := migrateFn(fromVersion, toVersion, convertedObject, VolumeCRDKind); err != nil {
		return nil, statusErrorWithMessage(err.Error())
	}

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

func upgradeObject(fromVersion, toVersion string, convertedObject *unstructured.Unstructured, crdKind CRDKind) error {
	switch crdKind {
	case DriveCRDKind:
		if err := upgradeDriveObject(fromVersion, toVersion, convertedObject); err != nil {
			return err
		}
	case VolumeCRDKind:
		if err := upgradeVolumeObject(fromVersion, toVersion, convertedObject); err != nil {
			return err
		}
	default:
		return ErrCRDKindNotSupported
	}
	return nil
}

func downgradeObject(fromVersion, toVersion string, convertedObject *unstructured.Unstructured, crdKind CRDKind) error {
	switch crdKind {
	case DriveCRDKind:
		if err := downgradeDriveObject(fromVersion, toVersion, convertedObject); err != nil {
			return err
		}
	case VolumeCRDKind:
		if err := downgradeVolumeObject(fromVersion, toVersion, convertedObject); err != nil {
			return err
		}
	default:
		return ErrCRDKindNotSupported
	}
	return nil
}
