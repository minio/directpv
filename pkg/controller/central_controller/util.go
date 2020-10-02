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

package centralcontroller

import (
	"context"
	"fmt"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientset "k8s.io/client-go/kubernetes"
	k8sretry "k8s.io/client-go/util/retry"

	"github.com/golang/glog"
	direct "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	directclientset "github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/util"
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

func createDirectCSINamespace(ctx context.Context, kClient clientset.Interface, name string) error {
	ns := &corev1.Namespace{
		metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		metav1.ObjectMeta{
			Name: name,
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

func setFinalizer(ctx context.Context, stClient directclientset.Interface, st *direct.StorageTopology) error {
	new := st.DeepCopy()

	new.SetFinalizers([]string{st.Name})
	
	// Set Finalizer
	return retry(func() error {
		_, err := stClient.DirectV1alpha1().StorageTopologies().Update(ctx, new, metav1.UpdateOptions{})
		return err
	})
}

func deleteFinalizer(ctx context.Context, stClient directclientset.Interface, st *direct.StorageTopology) error {
	finalizers := st.GetFinalizers()
	newFins := []string{}
	for _, fin := range finalizers {
		if fin != st.Name {
			newFins = append(newFins, fin)
		}
	}
	
	new := st.DeepCopy()
	new.SetFinalizers(newFins)
	
	// Set Finalizer
	return retry(func() error {
		_, err := stClient.DirectV1alpha1().StorageTopologies().Update(ctx, new, metav1.UpdateOptions{})
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

func createRBACRoles(ctx context.Context, kClient clientset.Interface, name string) error {
	if err := createServiceAccount(ctx, kClient, name); err != nil {
		return err
	}
	if err := createClusterRole(ctx, kClient, name); err != nil {
		return err
	}
	if err := createClusterRoleBinding(ctx, kClient, name); err != nil {
		return err
	}
	return nil
}

func createServiceAccount(ctx context.Context, kClient clientset.Interface, name string) error {
	serviceAccount := &corev1.ServiceAccount{
		metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		metav1.ObjectMeta{
			Name:      name,
			Namespace: name,
			Labels: map[string]string{
				CreateByLabel: DirectCSIController,
			},
		},
		[]corev1.ObjectReference{},
		[]corev1.LocalObjectReference{},
		nil,
	}

	return retry(func() error {
		_, err := kClient.CoreV1().ServiceAccounts(name).Create(ctx, serviceAccount, metav1.CreateOptions{})
		return err
	})
}

func createClusterRoleBinding(ctx context.Context, kClient clientset.Interface, name string) error {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				CreateByLabel: DirectCSIController,
			},
			Annotations: map[string]string{
				"rbac.authorization.kubernetes.io/autoupdate": "true",
			},
		},
		[]rbacv1.Subject{
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      name,
				Namespace: name,
			},
		},
		rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     name,
		},
	}

	return retry(func() error {
		_, err := kClient.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
		return err
	})
}

func createClusterRole(ctx context.Context, kClient clientset.Interface, name string) error {
	clusterRole := &rbacv1.ClusterRole{
		metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				CreateByLabel: DirectCSIController,
			},
			Annotations: map[string]string{
				"rbac.authorization.kubernetes.io/autoupdate": "true",
			},
		},
		[]rbacv1.PolicyRule{
			rbacv1.PolicyRule{
				Verbs: []string{
					ClusterRoleVerbGet,
					ClusterRoleVerbList,
					ClusterRoleVerbWatch,
					ClusterRoleVerbCreate,
					ClusterRoleVerbDelete,
				},
				Resources: []string{
					"persistentvolumes",
				},
				APIGroups: []string{
					"",
				},
			},
			rbacv1.PolicyRule{
				Verbs: []string{
					ClusterRoleVerbGet,
					ClusterRoleVerbList,
					ClusterRoleVerbWatch,
					ClusterRoleVerbUpdate,
				},
				Resources: []string{
					"persistentvolumeclaims",
				},
				APIGroups: []string{
					"",
				},
			},
			rbacv1.PolicyRule{
				Verbs: []string{
					ClusterRoleVerbGet,
					ClusterRoleVerbList,
					ClusterRoleVerbWatch,
				},
				Resources: []string{
					"storageclasses",
				},
				APIGroups: []string{
					"storage.k8s.io",
				},
			},
			rbacv1.PolicyRule{
				Verbs: []string{
					ClusterRoleVerbCreate,
					ClusterRoleVerbList,
					ClusterRoleVerbWatch,
					ClusterRoleVerbUpdate,
					ClusterRoleVerbPatch,
				},
				Resources: []string{
					"events",
				},
				APIGroups: []string{
					"",
				},
			},
			rbacv1.PolicyRule{
				Verbs: []string{
					ClusterRoleVerbGet,
					ClusterRoleVerbList,
				},
				Resources: []string{
					"volumesnapshots",
				},
				APIGroups: []string{
					"snapshot.storage.k8s.io",
				},
			},
			rbacv1.PolicyRule{
				Verbs: []string{
					ClusterRoleVerbGet,
					ClusterRoleVerbList,
				},
				Resources: []string{
					"volumesnapshotcontents",
				},
				APIGroups: []string{
					"snapshot.storage.k8s.io",
				},
			},
			rbacv1.PolicyRule{
				Verbs: []string{
					ClusterRoleVerbGet,
					ClusterRoleVerbList,
					ClusterRoleVerbWatch,
				},
				Resources: []string{
					"csinodes",
				},
				APIGroups: []string{
					"storage.k8s.io",
				},
			},
			rbacv1.PolicyRule{
				Verbs: []string{
					ClusterRoleVerbGet,
					ClusterRoleVerbList,
					ClusterRoleVerbWatch,
				},
				Resources: []string{
					"nodes",
				},
				APIGroups: []string{
					"",
				},
			},
			rbacv1.PolicyRule{
				Verbs: []string{
					ClusterRoleVerbGet,
					ClusterRoleVerbList,
					ClusterRoleVerbWatch,
				},
				Resources: []string{
					"volumeattachments",
				},
				APIGroups: []string{
					"storage.k8s.io",
				},
			},
			rbacv1.PolicyRule{
				Verbs: []string{
					ClusterRoleVerbGet,
					ClusterRoleVerbList,
					ClusterRoleVerbWatch,
					ClusterRoleVerbCreate,
					ClusterRoleVerbUpdate,
					ClusterRoleVerbDelete,
				},
				Resources: []string{
					"endpoints",
				},
				APIGroups: []string{
					"",
				},
			},
			rbacv1.PolicyRule{
				Verbs: []string{
					ClusterRoleVerbGet,
					ClusterRoleVerbList,
					ClusterRoleVerbWatch,
					ClusterRoleVerbCreate,
					ClusterRoleVerbUpdate,
					ClusterRoleVerbDelete,
				},
				Resources: []string{
					"leases",
				},
				APIGroups: []string{
					"coordination.k8s.io",
				},
			},
			rbacv1.PolicyRule{
				Verbs: []string{
					ClusterRoleVerbGet,
					ClusterRoleVerbList,
					ClusterRoleVerbWatch,
					ClusterRoleVerbCreate,
					ClusterRoleVerbUpdate,
					ClusterRoleVerbDelete,
				},
				Resources: []string{
					"volumes",
				},
				APIGroups: []string{
					"direct.csi.min.io",
				},
			},
			rbacv1.PolicyRule{
				Verbs: []string{
					ClusterRoleVerbGet,
					ClusterRoleVerbList,
					ClusterRoleVerbWatch,
					ClusterRoleVerbCreate,
					ClusterRoleVerbUpdate,
					ClusterRoleVerbDelete,
				},
				Resources: []string{
					"customresourcedefinitions",
				},
				APIGroups: []string{
					"apiextensions.k8s.io",
				},
			},
		},
		nil,
	}

	return retry(func() error {
		_, err := kClient.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
		return err
	})
}

func createDeployment(ctx context.Context, kClient clientset.Interface, name string, identity string) error {
	generatedSelectorValue := util.GenerateSanitizedUniqueNameFrom(name)

	privileged := false
	podSpec := corev1.PodSpec{
		ServiceAccountName: name,
		Volumes: []corev1.Volume{
			newHostPathVolume(VolumeNameSocketDir, newDirectCSIPluginsSocketDir(name)),
		},
		Containers: []corev1.Container{
			{
				Name:  CSIProvisionerContainerName,
				Image: CSIProvisionerContainerImage,
				Args: []string{
					"--v=5",
					"--csi-address=/csi/csi.sock",
					"--timeout=300s",
					"--enable-leader-election",
					"--leader-election-type=leases",
					"--feature-gates=Topology=true",
					"--strict-topology",
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
					{
						Name:  CSIEndpointEnvVar,
						Value: "unix:///csi/csi.sock",
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(VolumeNameSocketDir, "/csi"),
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
				Ports: []corev1.ContainerPort{
					{
						Name:          HealthZContainerPortName,
						ContainerPort: HealthZContainerPort,
						Protocol:      HealthZContainerPortProtocol,
					},
				},
			},
			{
				Name:  DirectCSIContainerName,
				Image: DirectCSIContainerImage,
				Args: []string{
					fmt.Sprintf("--identity=%s", identity),
					"--v=5",
					"--csi-address=/csi/csi.sock",
					fmt.Sprintf("--endpoint=$(%s)", CSIEndpointEnvVar),
					fmt.Sprintf("--node-id=$(%s)", KubeNodeNameEnvVar),
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
					{
						Name:  CSIEndpointEnvVar,
						Value: "unix:///csi/csi.sock",
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(VolumeNameSocketDir, "/csi"),
				},
			},
		},
	}

	deployment := &appsv1.Deployment{
		metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		metav1.ObjectMeta{
			Name:      name,
			Namespace: name,
			Labels: map[string]string{
				CreateByLabel: DirectCSIController,
			},
		},
		appsv1.DeploymentSpec{
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
		appsv1.DeploymentStatus{},
	}
	return retry(func() error {
		_, err := kClient.AppsV1().Deployments(name).Create(ctx, deployment, metav1.CreateOptions{})
		return err
	})
}
