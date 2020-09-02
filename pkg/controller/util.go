// This file is part of MinIO Kubernetes Cloud
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

package controller

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientset "k8s.io/client-go/kubernetes"
	k8sretry "k8s.io/client-go/util/retry"

	"github.com/golang/glog"
	"github.com/minio/direct-csi/pkg/util"
	"github.com/minio/minio/pkg/ellipses"
)

func retry(f func() error) error {
	if err := k8sretry.OnError(k8sretry.DefaultBackoff, func(err error) bool {
		if errors.IsAlreadyExists(err) {
			return false
		}
		return true
	}, f); err != nil {
		if !errors.IsAlreadyExists(err) {
			glog.Errorf("CSIDriver creation failed: %v", err)
			return err
		}
	}
	return nil
}

func createDirectCSINamespace(ctx context.Context, kClient clientset.Interface) error {
	ns := &corev1.Namespace{
		metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		metav1.ObjectMeta{
			Name: DirectCSINS,
			Labels: map[string]string{
				CreateByLabel: DirectCSIController,
			},
		},
		corev1.NamespaceSpec{
			Finalizers: []corev1.FinalizerName{},
		},
		corev1.NamespaceStatus{},
	}

	// Create Namespace Obj
	return retry(func() error {
		_, err := kClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		return err
	})
}

func createCSIDriver(ctx context.Context, kClient clientset.Interface, name string) error {
	podInfoOnMount := false
	attachRequired := false
	csiDriver := &storagev1.CSIDriver{
		metav1.TypeMeta{
			Kind:       "CSIDriver",
			APIVersion: "storage.k8s.io/v1",
		},
		metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				CreateByLabel: DirectCSIController,
			},
		},
		storagev1.CSIDriverSpec{
			PodInfoOnMount: &podInfoOnMount,
			AttachRequired: &attachRequired,
			VolumeLifecycleModes: []storagev1.VolumeLifecycleMode{
				storagev1.VolumeLifecyclePersistent,
			},
		},
	}

	// Create CSIDriver Obj
	return retry(func() error {
		_, err := kClient.StorageV1().CSIDrivers().Create(ctx, csiDriver, metav1.CreateOptions{})
		return err
	})
}

func createStorageClass(ctx context.Context, kClient clientset.Interface, name string) error {
	allowExpansion := false
	bindingMode := storagev1.VolumeBindingImmediate
	allowedTopologies := []corev1.TopologySelectorTerm{}
	parameters := map[string]string{}
	mountOptions := []string{}
	retainPolicy := corev1.PersistentVolumeReclaimRetain

	// Create StorageClass for the new driver
	storageClass := &storagev1.StorageClass{
		metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: "storage.k8s.io/v1",
		},
		metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				CreateByLabel: DirectCSIController,
			},
		},
		name, // Provisioner
		parameters,
		&retainPolicy,
		mountOptions,
		&allowExpansion,
		&bindingMode,
		allowedTopologies,
	}

	return retry(func() error {
		_, err := kClient.StorageV1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{})
		return err
	})
}

func newVolumeMount(name, path string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      name,
		MountPath: path,
	}
}

func newHostPathVolume(name, path string) corev1.Volume {
	hostPathType := corev1.HostPathDirectoryOrCreate
	volumeSource := corev1.VolumeSource{
		HostPath: &corev1.HostPathVolumeSource{
			Path: path,
			Type: &hostPathType,
		},
	}

	return corev1.Volume{
		name,
		volumeSource,
	}
}

func newDirectCSIPluginsSocketDir(name string) string {
	return filepath.Join(KubeletDir, "plugins", util.Sanitize(name))
}

func createDaemonSet(ctx context.Context, kClient clientset.Interface, name string, nodeSelector map[string]string, paths []string) error {
	generatedSelectorValue := util.GenerateSanitizedUniqueNameFrom(name)

	drives := []corev1.Volume{}
	basePaths := []string{}
	for _, path := range paths {
		if ellipses.HasEllipses(path) {
			p, err := ellipses.FindEllipsesPatterns(path)
			if err != nil {
				return err
			}
			patterns := p.Expand()
			for _, outer := range patterns {
				basePaths = append(basePaths, strings.Join(outer, ""))
			}
		} else {
			basePaths = append(basePaths, path)
		}
	}

	volumeIndex := 0
	nextVolName := func() string {
		volumeIndex = volumeIndex + 1
		return fmt.Sprintf("volume-%d", volumeIndex)
	}

	volumeMounts := []corev1.VolumeMount{}

	for _, path := range basePaths {
		volName := nextVolName()
		vol := newHostPathVolume(volName, path)
		drives = append(drives, vol)
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      volName,
			MountPath: path,
		})
	}

	privileged := true
	podSpec := corev1.PodSpec{
		NodeSelector: nodeSelector,
		Volumes: append([]corev1.Volume{
			newHostPathVolume(VolumeNameSocketDir, newDirectCSIPluginsSocketDir(name)),
			newHostPathVolume(VolumeNameMountpointDir, filepath.Join(KubeletDir, VolumePathMountpointDir)),
			newHostPathVolume(VolumeNamePluginsDir, filepath.Join(KubeletDir, VolumePathPluginsDir)),
			newHostPathVolume(VolumeNamePluginsRegistryDir, filepath.Join(KubeletDir, VolumePathPluginsRegistryDir)),
			newHostPathVolume(VolumeNameDevDir, VolumePathDevDir),
		}, drives...),
		ServiceAccountName: name,
		Containers: []corev1.Container{
			{
				Name:  NodeDriverRegistrarContainerName,
				Image: NodeDriverRegistrarContainerImage,
				Args: []string{
					"--v=5",
					"--csi-address=/csi/csi.sock",
					fmt.Sprintf("--kubelet-registration-path=%s", newDirectCSIPluginsSocketDir(name)),
				},
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Env: []corev1.EnvVar{
					{
						Name: KubeNodeNameEnvVar,
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "v1",
								FieldPath:  "spec.nodeName",
							},
						},
					},
				},
				VolumeMounts: append([]corev1.VolumeMount{
					newVolumeMount(VolumeNameSocketDir, "/csi"),
					newVolumeMount(VolumeNamePluginsRegistryDir, "/registration"),
					newVolumeMount(VolumeNameDevDir, VolumePathDevDir),
				}, volumeMounts...),
			},
			{
				Name:  DirectCSIContainerName,
				Image: DirectCSIContainerImage,
				Args: append([]string{
					"--v=5",
					"--csi-address=/csi/csi.sock",
					fmt.Sprintf("--endpoint=$(%s)", CSIEndpointEnvVar),
					fmt.Sprintf("--node-id=$(%s)", KubeNodeNameEnvVar),
				}, basePaths...),
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Env: []corev1.EnvVar{
					{
						Name: KubeNodeNameEnvVar,
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "v1",
								FieldPath:  "spec.nodeName",
							},
						},
					},
					{
						Name:  CSIEndpointEnvVar,
						Value: "unix:///csi/csi.sock",
					},
				},
				VolumeMounts: append([]corev1.VolumeMount{
					newVolumeMount(VolumeNameSocketDir, "/csi"),
					newVolumeMount(VolumeNameMountpointDir, filepath.Join(KubeletDir, VolumePathMountpointDir)),
					newVolumeMount(VolumeNamePluginsDir, filepath.Join(KubeletDir, VolumePathPluginsDir)),
					newVolumeMount(VolumeNameDevDir, VolumePathDevDir),
				}, volumeMounts...),
				Ports: []corev1.ContainerPort{
					{
						Name:          HealthZContainerPortName,
						ContainerPort: HealthZContainerPort,
						Protocol:      HealthZContainerPortProtocol,
					},
				},
				LivenessProbe: &corev1.Probe{
					FailureThreshold:    5,
					InitialDelaySeconds: 10,
					TimeoutSeconds:      3,
					PeriodSeconds:       2,
					Handler: corev1.Handler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: HealthZContainerPortPath,
							Port: intstr.FromString(HealthZContainerPortName),
						},
					},
				},
			},
			{
				Name:  LivenessProbeContainerName,
				Image: LivenessProbeContainerImage,
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(VolumeNameSocketDir, "/csi"),
				},
				Args: []string{
					fmt.Sprintf("--health-port=%d", HealthZContainerPort),
					"--csi-address=/csi/csi.sock",
				},
			},
		},
	}

	daemonSet := &appsv1.DaemonSet{
		metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		metav1.ObjectMeta{
			Name:      name,
			Namespace: DirectCSINS,
			Labels: map[string]string{
				CreateByLabel: DirectCSIController,
			},
		},
		appsv1.DaemonSetSpec{
			Selector: metav1.AddLabelToSelector(&metav1.LabelSelector{}, DirectCSISelector, generatedSelectorValue),
			Template: corev1.PodTemplateSpec{
				metav1.ObjectMeta{
					Name: name,
					Labels: map[string]string{
						DirectCSISelector: generatedSelectorValue,
					},
				},
				podSpec,
			},
		},
		appsv1.DaemonSetStatus{},
	}
	return retry(func() error {
		_, err := kClient.AppsV1().DaemonSets(DirectCSINS).Create(ctx, daemonSet, metav1.CreateOptions{})
		return err
	})
}

func createRBACRoles(ctx context.Context, kClient clientset.Interface, name string) error {
	return nil
}
