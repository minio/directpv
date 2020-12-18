// This file is part of MinIO Direct CSI
// Copyright (c) 2020 MinIO, Inc.
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

package utils

import (
	"context"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Label to denote the creator
	CreatedByLabel = "created-by"
	// Denotes that it was created by direct-csi-controller
	DirectCSIControllerName = "controller.direct.csi.min.io"

	CSIDriver = "CSIDriver"
	DirectCSI = "direct.csi.min.io"

	clusterRoleVerbList   = "list"
	clusterRoleVerbGet    = "get"
	clusterRoleVerbWatch  = "watch"
	clusterRoleVerbCreate = "create"
	clusterRoleVerbDelete = "delete"
	clusterRoleVerbUpdate = "update"
	clusterRoleVerbPatch  = "patch"

	volumeNameSocketDir       = "socket-dir"
	volumeNameDevDir          = "dev-dir"
	volumePathDevDir          = "/dev"
	volumeNameSysDir          = "sys-fs"
	volumePathSysDir          = "/sys"
	volumeNameCSIRootDir      = "direct-csi-common-root"
	volumeNameMountpointDir   = "mountpoint-dir"
	volumeNameRegistrationDir = "registration-dir"
	volumeNamePluginDir       = "plugins-dir"

	directCSISelector = "selector.direct.csi.min.io"

	csiProvisionerContainerName  = "csi-provisioner"
	csiProvisionerContainerImage = "quay.io/k8scsi/csi-provisioner:v1.2.1"

	directCSIContainerName  = "direct-csi"
	directCSIContainerImage = "minio/direct-csi:v0.2.1"

	livenessProbeContainerName  = "liveness-probe"
	livenessProbeContainerImage = "quay.io/k8scsi/livenessprobe:v1.1.0"

	nodeDriverRegistrarContainerName  = "node-driver-registrar"
	nodeDriverRegistrarContainerImage = "quay.io/k8scsi/csi-node-driver-registrar:v2.0.0"

	healthZContainerPort         = 9898
	healthZContainerPortName     = "healthz"
	healthZContainerPortProtocol = "TCP"
	healthZContainerPortPath     = "/healthz"

	kubeNodeNameEnvVar = "KUBE_NODE_NAME"
	endpointEnvVarCSI  = "CSI_ENDPOINT"
)

func CreateNamespace(ctx context.Context, identity string) error {
	ns := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: sanitizeName(identity),
			Labels: map[string]string{
				CreatedByLabel: DirectCSIControllerName,
			},
			Annotations: map[string]string{
				"app":  DirectCSI,
				"type": CSIDriver,
			},
		},
		Spec: corev1.NamespaceSpec{
			Finalizers: []corev1.FinalizerName{},
		},
		Status: corev1.NamespaceStatus{},
	}

	// Create Namespace Obj
	if _, err := kubeClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func CreateCSIDriver(ctx context.Context, identity string) error {
	podInfoOnMount := false
	attachRequired := false

	gvk, err := GetGroupKindVersions("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return err
	}

	switch gvk.Version {
	case "v1":
		csiDriver := &storagev1.CSIDriver{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CSIDriver",
				APIVersion: "storage.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: sanitizeName(identity),
				Labels: map[string]string{
					CreatedByLabel: DirectCSIControllerName,
				},
				Annotations: map[string]string{
					"app":  DirectCSI,
					"type": CSIDriver,
				},
			},
			Spec: storagev1.CSIDriverSpec{
				PodInfoOnMount: &podInfoOnMount,
				AttachRequired: &attachRequired,
				VolumeLifecycleModes: []storagev1.VolumeLifecycleMode{
					storagev1.VolumeLifecyclePersistent,
					storagev1.VolumeLifecycleEphemeral,
				},
			},
		}

		// Create CSIDriver Obj
		if _, err := kubeClient.StorageV1().CSIDrivers().Create(ctx, csiDriver, metav1.CreateOptions{}); err != nil {
			return err
		}
	case "v1beta1":
		csiDriver := &storagev1beta1.CSIDriver{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CSIDriver",
				APIVersion: "storage.k8s.io/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: sanitizeName(identity),
				Labels: map[string]string{
					CreatedByLabel: DirectCSIControllerName,
				},
				Annotations: map[string]string{
					"app":  DirectCSI,
					"type": CSIDriver,
				},
			},
			Spec: storagev1beta1.CSIDriverSpec{
				PodInfoOnMount: &podInfoOnMount,
				AttachRequired: &attachRequired,
				VolumeLifecycleModes: []storagev1beta1.VolumeLifecycleMode{
					storagev1beta1.VolumeLifecyclePersistent,
					storagev1beta1.VolumeLifecycleEphemeral,
				},
			},
		}

		// Create CSIDriver Obj
		if _, err := kubeClient.StorageV1beta1().CSIDrivers().Create(ctx, csiDriver, metav1.CreateOptions{}); err != nil {
			return err
		}
	default:
		return ErrKubeVersionNotSupported
	}
	return nil
}

func sanitizeName(s string) string {
	re := regexp.MustCompile("[^a-zA-Z0-9-]")
	s = re.ReplaceAllString(s, "-")
	if s[len(s)-1] == '-' {
		s = s + "X"
	}
	return s
}
