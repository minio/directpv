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
	"path/filepath"
	"strings"

	directv1alpha1 "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1alpha1"
	directv1beta1 "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta1"
	directv1beta2 "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta2"
	directv1beta3 "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	directv1beta4 "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
)

func convertV1Beta1V1Beta2DeviceName(devName string) string {
	switch {
	case strings.HasPrefix(devName, sys.HostDevRoot):
		return devName
	case strings.Contains(devName, sys.DirectCSIDevRoot):
		return convertV1Beta1V1Beta2DeviceName(filepath.Base(devName))
	default:
		name := strings.ReplaceAll(
			strings.Replace(devName, sys.DirectCSIPartitionInfix, "", 1),
			sys.DirectCSIPartitionInfix,
			sys.HostPartitionInfix,
		)
		return filepath.Join(sys.HostDevRoot, name)
	}
}

func upgradeDriveObject(object *unstructured.Unstructured, toVersion string) error {
	switch object.GetAPIVersion() {
	case versionV1Alpha1:
		if toVersion == versionV1Alpha1 {
			klog.V(10).Info("Successfully migrated")
			break
		}
		if err := driveUpgradeV1alpha1ToV1Beta1(object); err != nil {
			return err
		}
		fallthrough
	case versionV1Beta1:
		if toVersion == versionV1Beta1 {
			klog.V(10).Info("Successfully migrated")
			break
		}
		if err := driveUpgradeV1Beta1ToV1Beta2(object); err != nil {
			return err
		}
		fallthrough
	case versionV1Beta2:
		if toVersion == versionV1Beta2 {
			klog.V(10).Info("Successfully migrated")
			break
		}
		if err := driveUpgradeV1Beta2ToV1Beta3(object); err != nil {
			return err
		}
		fallthrough
	case versionV1Beta3:
		if toVersion == versionV1Beta3 {
			klog.V(10).Info("Successfully migrated")
			break
		}
		if err := driveUpgradeV1Beta3ToV1Beta4(object); err != nil {
			return err
		}
		fallthrough
	case versionV1Beta4:
		if toVersion == versionV1Beta4 {
			klog.V(10).Info("Successfully migrated")
			break
		}
	}
	return nil
}

func driveUpgradeV1alpha1ToV1Beta1(unstructured *unstructured.Unstructured) error {

	unstructuredObject := unstructured.Object

	var v1alpha1DirectCSIDrive directv1alpha1.DirectCSIDrive
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject, &v1alpha1DirectCSIDrive)
	if err != nil {
		return err
	}

	klog.V(10).Infof("Converting directpvdrive: %v to v1beta1", v1alpha1DirectCSIDrive.Name)

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

func driveUpgradeV1Beta1ToV1Beta2(unstructured *unstructured.Unstructured) error {

	unstructuredObject := unstructured.Object

	var v1beta1DirectCSIDrive directv1beta1.DirectCSIDrive
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject, &v1beta1DirectCSIDrive)
	if err != nil {
		return err
	}

	klog.V(10).Infof("Converting directpvdrive: %v to v1beta2", v1beta1DirectCSIDrive.Name)

	var v1beta2DirectCSIDrive directv1beta2.DirectCSIDrive
	if err := directv1beta2.Convert_v1beta1_DirectCSIDrive_To_v1beta2_DirectCSIDrive(&v1beta1DirectCSIDrive, &v1beta2DirectCSIDrive, nil); err != nil {
		return err
	}

	v1beta2DirectCSIDrive.TypeMeta = v1beta1DirectCSIDrive.TypeMeta
	v1beta2DirectCSIDrive.Status.Path = convertV1Beta1V1Beta2DeviceName(v1beta1DirectCSIDrive.Status.Path)
	utils.UpdateLabels(&v1beta2DirectCSIDrive, map[utils.LabelKey]utils.LabelValue{
		utils.NodeLabelKey:       utils.NewLabelValue(v1beta1DirectCSIDrive.Status.NodeName),
		utils.PathLabelKey:       utils.NewLabelValue(filepath.Base(v1beta2DirectCSIDrive.Status.Path)),
		utils.CreatedByLabelKey:  utils.DirectCSIDriverName,
		utils.AccessTierLabelKey: utils.NewLabelValue(string(v1beta1DirectCSIDrive.Status.AccessTier)),
	})

	convertedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&v1beta2DirectCSIDrive)
	if err != nil {
		return err
	}

	unstructured.Object = convertedObj
	return nil
}

func driveUpgradeV1Beta2ToV1Beta3(unstructured *unstructured.Unstructured) error {

	unstructuredObject := unstructured.Object

	var v1beta2DirectCSIDrive directv1beta2.DirectCSIDrive
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject, &v1beta2DirectCSIDrive)
	if err != nil {
		return err
	}

	klog.V(10).Infof("Converting directpvdrive: %v to v1beta3", v1beta2DirectCSIDrive.Name)

	var v1beta3DirectCSIDrive directv1beta3.DirectCSIDrive
	if err := directv1beta3.Convert_v1beta2_DirectCSIDrive_To_v1beta3_DirectCSIDrive(&v1beta2DirectCSIDrive, &v1beta3DirectCSIDrive, nil); err != nil {
		return err
	}

	v1beta3DirectCSIDrive.TypeMeta = v1beta2DirectCSIDrive.TypeMeta
	v1beta3DirectCSIDrive.Status.Path = convertV1Beta1V1Beta2DeviceName(v1beta2DirectCSIDrive.Status.Path)
	utils.UpdateLabels(&v1beta3DirectCSIDrive, map[utils.LabelKey]utils.LabelValue{
		utils.NodeLabelKey:       utils.NewLabelValue(v1beta2DirectCSIDrive.Status.NodeName),
		utils.PathLabelKey:       utils.NewLabelValue(filepath.Base(v1beta3DirectCSIDrive.Status.Path)),
		utils.CreatedByLabelKey:  utils.DirectCSIDriverName,
		utils.AccessTierLabelKey: utils.NewLabelValue(string(v1beta2DirectCSIDrive.Status.AccessTier)),
	})

	convertedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&v1beta3DirectCSIDrive)
	if err != nil {
		return err
	}

	unstructured.Object = convertedObj
	return nil
}

func driveUpgradeV1Beta3ToV1Beta4(unstructured *unstructured.Unstructured) error {

	unstructuredObject := unstructured.Object

	var v1beta3DirectCSIDrive directv1beta3.DirectCSIDrive
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject, &v1beta3DirectCSIDrive)
	if err != nil {
		return err
	}

	klog.V(10).Infof("Converting directpvdrive: %v to v1beta4", v1beta3DirectCSIDrive.Name)

	var v1beta4DirectCSIDrive directv1beta4.DirectCSIDrive
	if err := directv1beta4.Convert_v1beta3_DirectCSIDrive_To_v1beta4_DirectCSIDrive(&v1beta3DirectCSIDrive, &v1beta4DirectCSIDrive, nil); err != nil {
		return err
	}

	v1beta4DirectCSIDrive.TypeMeta = v1beta3DirectCSIDrive.TypeMeta
	v1beta4DirectCSIDrive.Status.Path = convertV1Beta1V1Beta2DeviceName(v1beta3DirectCSIDrive.Status.Path)
	utils.UpdateLabels(&v1beta4DirectCSIDrive, map[utils.LabelKey]utils.LabelValue{
		utils.NodeLabelKey:       utils.NewLabelValue(v1beta3DirectCSIDrive.Status.NodeName),
		utils.PathLabelKey:       utils.NewLabelValue(filepath.Base(v1beta4DirectCSIDrive.Status.Path)),
		utils.CreatedByLabelKey:  utils.DirectCSIDriverName,
		utils.AccessTierLabelKey: utils.NewLabelValue(string(v1beta3DirectCSIDrive.Status.AccessTier)),
	})

	convertedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&v1beta4DirectCSIDrive)
	if err != nil {
		return err
	}

	unstructured.Object = convertedObj
	return nil
}
