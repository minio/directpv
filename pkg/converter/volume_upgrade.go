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
	"k8s.io/klog/v2"

	directv1alpha1 "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1alpha1"
	directv1beta1 "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta1"
	directv1beta2 "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta2"
	directv1beta3 "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	directv1beta4 "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func upgradeVolumeObject(object *unstructured.Unstructured, toVersion string) error {
	switch object.GetAPIVersion() {
	case versionV1Alpha1:
		if toVersion == versionV1Alpha1 {
			klog.V(10).Info("Successfully migrated")
			break
		}
		if err := volumeUpgradeV1alpha1ToV1Beta1(object); err != nil {
			return err
		}
		fallthrough
	case versionV1Beta1:
		if toVersion == versionV1Beta1 {
			klog.V(10).Info("Successfully migrated")
			break
		}
		if err := volumeUpgradeV1Beta1ToV1Beta2(object); err != nil {
			return err
		}
		fallthrough
	case versionV1Beta2:
		if toVersion == versionV1Beta2 {
			klog.V(10).Info("Successfully migrated")
			break
		}
		if err := volumeUpgradeV1Beta2ToV1Beta3(object); err != nil {
			return err
		}
		fallthrough
	case versionV1Beta3:
		if toVersion == versionV1Beta3 {
			klog.V(10).Info("Successfully migrated")
			break
		}
		if err := volumeUpgradeV1Beta3ToV1Beta4(object); err != nil {
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

func volumeUpgradeV1alpha1ToV1Beta1(unstructured *unstructured.Unstructured) error {

	unstructuredObject := unstructured.Object

	var v1alpha1DirectCSIVolume directv1alpha1.DirectCSIVolume
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject, &v1alpha1DirectCSIVolume)
	if err != nil {
		return err
	}

	klog.V(10).Infof("Converting directpvvolume:%v volume to v1beta1", v1alpha1DirectCSIVolume.Name)

	var v1beta1DirectCSIVolume directv1beta1.DirectCSIVolume
	if err := directv1beta1.Convert_v1alpha1_DirectCSIVolume_To_v1beta1_DirectCSIVolume(&v1alpha1DirectCSIVolume, &v1beta1DirectCSIVolume, nil); err != nil {
		return err
	}

	getReadyCondition := func() metav1.Condition {
		isVolumeStaged := func() bool {
			return v1beta1DirectCSIVolume.Status.StagingPath != ""
		}()

		return metav1.Condition{
			Type: string(directv1beta1.DirectCSIVolumeConditionReady),
			Status: func() metav1.ConditionStatus {
				if isVolumeStaged {
					return metav1.ConditionTrue
				}
				return metav1.ConditionFalse
			}(),
			Reason: func() string {
				if isVolumeStaged {
					return string(directv1beta1.DirectCSIVolumeReasonReady)
				}
				return string(directv1beta1.DirectCSIVolumeReasonNotReady)
			}(),
			Message:            "",
			LastTransitionTime: metav1.Now(),
		}
	}

	v1beta1DirectCSIVolume.Status.Conditions = append(v1beta1DirectCSIVolume.Status.Conditions, getReadyCondition())
	v1beta1DirectCSIVolume.TypeMeta = v1alpha1DirectCSIVolume.TypeMeta

	convertedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&v1beta1DirectCSIVolume)
	if err != nil {
		return err
	}

	unstructured.Object = convertedObj
	return nil
}

func volumeUpgradeV1Beta1ToV1Beta2(unstructured *unstructured.Unstructured) error {

	unstructuredObject := unstructured.Object

	var v1beta1DirectCSIVolume directv1beta1.DirectCSIVolume
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject, &v1beta1DirectCSIVolume)
	if err != nil {
		return err
	}

	klog.V(10).Infof("Converting directpvvolume:%v to v1beta2", v1beta1DirectCSIVolume.Name)

	var v1beta2DirectCSIVolume directv1beta2.DirectCSIVolume
	if err := directv1beta2.Convert_v1beta1_DirectCSIVolume_To_v1beta2_DirectCSIVolume(&v1beta1DirectCSIVolume, &v1beta2DirectCSIVolume, nil); err != nil {
		return err
	}

	v1beta2DirectCSIVolume.TypeMeta = v1beta1DirectCSIVolume.TypeMeta
	utils.UpdateLabels(&v1beta2DirectCSIVolume, map[utils.LabelKey]utils.LabelValue{
		utils.NodeLabelKey:      utils.NewLabelValue(v1beta2DirectCSIVolume.Status.NodeName),
		utils.CreatedByLabelKey: utils.DirectCSIControllerName,
	})

	convertedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&v1beta2DirectCSIVolume)
	if err != nil {
		return err
	}

	unstructured.Object = convertedObj
	return nil
}

func volumeUpgradeV1Beta2ToV1Beta3(unstructured *unstructured.Unstructured) error {

	unstructuredObject := unstructured.Object

	var v1beta2DirectCSIVolume directv1beta2.DirectCSIVolume
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject, &v1beta2DirectCSIVolume)
	if err != nil {
		return err
	}

	klog.V(10).Infof("Converting directpvvolume:%v to v1beta3", v1beta2DirectCSIVolume.Name)

	var v1beta3DirectCSIVolume directv1beta3.DirectCSIVolume
	if err := directv1beta3.Convert_v1beta2_DirectCSIVolume_To_v1beta3_DirectCSIVolume(&v1beta2DirectCSIVolume, &v1beta3DirectCSIVolume, nil); err != nil {
		return err
	}

	v1beta3DirectCSIVolume.TypeMeta = v1beta2DirectCSIVolume.TypeMeta
	utils.UpdateLabels(&v1beta3DirectCSIVolume, map[utils.LabelKey]utils.LabelValue{
		utils.NodeLabelKey:      utils.NewLabelValue(v1beta3DirectCSIVolume.Status.NodeName),
		utils.CreatedByLabelKey: utils.DirectCSIControllerName,
	})

	convertedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&v1beta3DirectCSIVolume)
	if err != nil {
		return err
	}

	unstructured.Object = convertedObj
	return nil
}

func volumeUpgradeV1Beta3ToV1Beta4(unstructured *unstructured.Unstructured) error {

	unstructuredObject := unstructured.Object

	var v1beta3DirectCSIVolume directv1beta3.DirectCSIVolume
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject, &v1beta3DirectCSIVolume)
	if err != nil {
		return err
	}

	klog.V(10).Infof("Converting directpvvolume:%v to v1beta4", v1beta3DirectCSIVolume.Name)

	var v1beta4DirectCSIVolume directv1beta4.DirectCSIVolume
	if err := directv1beta4.Convert_v1beta3_DirectCSIVolume_To_v1beta4_DirectCSIVolume(&v1beta3DirectCSIVolume, &v1beta4DirectCSIVolume, nil); err != nil {
		return err
	}

	v1beta4DirectCSIVolume.TypeMeta = v1beta3DirectCSIVolume.TypeMeta
	utils.UpdateLabels(&v1beta3DirectCSIVolume, map[utils.LabelKey]utils.LabelValue{
		utils.NodeLabelKey:      utils.NewLabelValue(v1beta3DirectCSIVolume.Status.NodeName),
		utils.CreatedByLabelKey: utils.DirectCSIControllerName,
	})

	convertedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&v1beta4DirectCSIVolume)
	if err != nil {
		return err
	}

	unstructured.Object = convertedObj
	return nil
}
