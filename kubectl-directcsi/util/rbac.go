/*
 * This file is part of MinIO Direct CSI
 * Copyright (C) 2020, MinIO, Inc.
 *
 * This code is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License, version 3,
 * as published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License, version 3,
 * along with this program.  If not, see <http://www.gnu.org/licenses/>
 *
 */

package util

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clientset "k8s.io/client-go/kubernetes"
)

// CreateRBACRoles creates SA, ClusterRole and CRBs
func CreateRBACRoles(ctx context.Context, kClient clientset.Interface, name, identity string) error {
	if err := createServiceAccount(ctx, kClient, name, identity); err != nil {
		return err
	}
	fmt.Println("Created ServiceAccount ", name)
	if err := createClusterRole(ctx, kClient, name, identity); err != nil {
		return err
	}
	fmt.Println("Created ClusterRole ", name)
	if err := createClusterRoleBinding(ctx, kClient, name, identity); err != nil {
		return err
	}
	fmt.Println("Created ClusterRoleBinding ", name)
	return nil
}

func createServiceAccount(ctx context.Context, kClient clientset.Interface, name, identity string) error {
	serviceAccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: identity,
			Labels: map[string]string{
				createByLabel: directCSIController,
			},
		},
		Secrets:                      []corev1.ObjectReference{},
		ImagePullSecrets:             []corev1.LocalObjectReference{},
		AutomountServiceAccountToken: nil,
	}

	return retry(func() error {
		_, err := kClient.CoreV1().ServiceAccounts(identity).Create(ctx, serviceAccount, metav1.CreateOptions{})
		return err
	})
}

func createClusterRoleBinding(ctx context.Context, kClient clientset.Interface, name, identity string) error {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				createByLabel: directCSIController,
			},
			Annotations: map[string]string{
				"rbac.authorization.kubernetes.io/autoupdate": "true",
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      name,
				Namespace: identity,
			},
		},
		RoleRef: rbacv1.RoleRef{
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

func createClusterRole(ctx context.Context, kClient clientset.Interface, name, identity string) error {
	clusterRole := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: identity,
			Labels: map[string]string{
				createByLabel: directCSIController,
			},
			Annotations: map[string]string{
				"rbac.authorization.kubernetes.io/autoupdate": "true",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs: []string{
					clusterRoleVerbGet,
					clusterRoleVerbList,
					clusterRoleVerbWatch,
					clusterRoleVerbCreate,
					clusterRoleVerbDelete,
				},
				Resources: []string{
					"persistentvolumes",
				},
				APIGroups: []string{
					"",
				},
			},
			{
				Verbs: []string{
					clusterRoleVerbGet,
					clusterRoleVerbList,
					clusterRoleVerbWatch,
					clusterRoleVerbUpdate,
				},
				Resources: []string{
					"persistentvolumeclaims",
				},
				APIGroups: []string{
					"",
				},
			},
			{
				Verbs: []string{
					clusterRoleVerbGet,
					clusterRoleVerbList,
					clusterRoleVerbWatch,
				},
				Resources: []string{
					"storageclasses",
				},
				APIGroups: []string{
					"storage.k8s.io",
				},
			},
			{
				Verbs: []string{
					clusterRoleVerbCreate,
					clusterRoleVerbList,
					clusterRoleVerbWatch,
					clusterRoleVerbUpdate,
					clusterRoleVerbPatch,
				},
				Resources: []string{
					"events",
				},
				APIGroups: []string{
					"",
				},
			},
			{
				Verbs: []string{
					clusterRoleVerbGet,
					clusterRoleVerbList,
				},
				Resources: []string{
					"volumesnapshots",
				},
				APIGroups: []string{
					"snapshot.storage.k8s.io",
				},
			},
			{
				Verbs: []string{
					clusterRoleVerbGet,
					clusterRoleVerbList,
				},
				Resources: []string{
					"volumesnapshotcontents",
				},
				APIGroups: []string{
					"snapshot.storage.k8s.io",
				},
			},
			{
				Verbs: []string{
					clusterRoleVerbGet,
					clusterRoleVerbList,
					clusterRoleVerbWatch,
				},
				Resources: []string{
					"csinodes",
				},
				APIGroups: []string{
					"storage.k8s.io",
				},
			},
			{
				Verbs: []string{
					clusterRoleVerbGet,
					clusterRoleVerbList,
					clusterRoleVerbWatch,
				},
				Resources: []string{
					"nodes",
				},
				APIGroups: []string{
					"",
				},
			},
			{
				Verbs: []string{
					clusterRoleVerbGet,
					clusterRoleVerbList,
					clusterRoleVerbWatch,
				},
				Resources: []string{
					"volumeattachments",
				},
				APIGroups: []string{
					"storage.k8s.io",
				},
			},
			{
				Verbs: []string{
					clusterRoleVerbGet,
					clusterRoleVerbList,
					clusterRoleVerbWatch,
					clusterRoleVerbCreate,
					clusterRoleVerbUpdate,
					clusterRoleVerbDelete,
				},
				Resources: []string{
					"endpoints",
				},
				APIGroups: []string{
					"",
				},
			},
			{
				Verbs: []string{
					clusterRoleVerbGet,
					clusterRoleVerbList,
					clusterRoleVerbWatch,
					clusterRoleVerbCreate,
					clusterRoleVerbUpdate,
					clusterRoleVerbDelete,
				},
				Resources: []string{
					"leases",
				},
				APIGroups: []string{
					"coordination.k8s.io",
				},
			},
			{
				Verbs: []string{
					clusterRoleVerbGet,
					clusterRoleVerbList,
					clusterRoleVerbWatch,
					clusterRoleVerbCreate,
					clusterRoleVerbUpdate,
					clusterRoleVerbDelete,
				},
				Resources: []string{
					"volumes",
				},
				APIGroups: []string{
					"direct.csi.min.io",
				},
			},
			{
				Verbs: []string{
					clusterRoleVerbGet,
					clusterRoleVerbList,
					clusterRoleVerbWatch,
					clusterRoleVerbCreate,
					clusterRoleVerbUpdate,
					clusterRoleVerbDelete,
				},
				Resources: []string{
					"customresourcedefinitions",
				},
				APIGroups: []string{
					"apiextensions.k8s.io",
				},
			},
			{
				Verbs: []string{
					clusterRoleVerbGet,
					clusterRoleVerbList,
					clusterRoleVerbWatch,
					clusterRoleVerbCreate,
					clusterRoleVerbUpdate,
					clusterRoleVerbDelete,
				},
				Resources: []string{
					"directcsidrives", "directcsivolumes",
				},
				APIGroups: []string{
					"direct.csi.min.io",
				},
			},
		},
		AggregationRule: nil,
	}

	return retry(func() error {
		_, err := kClient.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
		return err
	})
}

// RemoveRBACRoles deletes SA, ClusterRole and CRBs
func RemoveRBACRoles(ctx context.Context, kClient clientset.Interface, name, identity string) error {
	if err := removeServiceAccount(ctx, kClient, name, identity); err != nil {
		return err
	}
	fmt.Println("Deleted ServiceAccount ", name)
	if err := removeCluterRole(ctx, kClient, name); err != nil {
		return err
	}
	fmt.Println("Deleted ClusterRole ", name)
	if err := removeClusterRoleBinding(ctx, kClient, name); err != nil {
		return err
	}
	fmt.Println("Deleted ClusterRoleBinding ", name)
	return nil
}

func removeServiceAccount(ctx context.Context, kClient clientset.Interface, name, identity string) error {
	return retry(func() error {
		return kClient.CoreV1().ServiceAccounts(identity).Delete(ctx, name, metav1.DeleteOptions{})
	})
}

func removeClusterRoleBinding(ctx context.Context, kClient clientset.Interface, name string) error {
	return retry(func() error {
		return kClient.RbacV1().ClusterRoleBindings().Delete(ctx, name, metav1.DeleteOptions{})
	})
}

func removeCluterRole(ctx context.Context, kClient clientset.Interface, name string) error {
	return retry(func() error {
		return kClient.RbacV1().ClusterRoles().Delete(ctx, name, metav1.DeleteOptions{})
	})
}
