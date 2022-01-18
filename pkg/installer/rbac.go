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

package installer

import (
	"context"

	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/utils"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func createServiceAccount(ctx context.Context, c *Config) error {
	serviceAccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        c.serviceAccountName(),
			Namespace:   c.namespace(),
			Annotations: defaultAnnotations,
			Labels:      defaultLabels,
		},
		Secrets:                      []corev1.ObjectReference{},
		ImagePullSecrets:             []corev1.LocalObjectReference{},
		AutomountServiceAccountToken: nil,
	}

	if c.DryRun {
		return c.postProc(serviceAccount)
	}

	if _, err := client.GetKubeClient().CoreV1().ServiceAccounts(c.namespace()).Create(ctx, serviceAccount, metav1.CreateOptions{}); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}

	return c.postProc(serviceAccount)
}

func createClusterRoleBinding(ctx context.Context, c *Config) error {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        c.roleBindingName(),
			Namespace:   metav1.NamespaceNone,
			Annotations: defaultAnnotations,
			Labels:      defaultLabels,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      c.serviceAccountName(),
				Namespace: c.namespace(),
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     c.clusterRoleName(),
		},
	}

	clusterRoleBinding.Annotations["rbac.authorization.kubernetes.io/autoupdate"] = "true"
	if c.DryRun {
		return c.postProc(clusterRoleBinding)
	}

	if _, err := client.GetKubeClient().RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{}); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}

	return c.postProc(clusterRoleBinding)
}

func createClusterRole(ctx context.Context, c *Config) error {
	clusterRole := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        c.clusterRoleName(),
			Namespace:   metav1.NamespaceNone,
			Annotations: defaultAnnotations,
			Labels:      defaultLabels,
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
					clusterRoleVerbPatch,
				},
				Resources: []string{
					"customresourcedefinitions",
					"customresourcedefinition",
				},
				APIGroups: []string{
					"apiextensions.k8s.io",
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
					"directcsidrives", "directcsivolumes",
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
				},
				Resources: []string{
					"pods",
					"pod",
				},
				APIGroups: []string{
					"",
				},
			},
		},
		AggregationRule: nil,
	}

	clusterRole.Annotations["rbac.authorization.kubernetes.io/autoupdate"] = "true"

	if c.DryRun {
		return c.postProc(clusterRole)
	}

	if _, err := client.GetKubeClient().RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{}); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}

	return c.postProc(clusterRole)
}

func installRBACDefault(ctx context.Context, c *Config) error {
	if err := createServiceAccount(ctx, c); err != nil {
		return err
	}
	if err := createClusterRole(ctx, c); err != nil {
		return err
	}
	if err := createClusterRoleBinding(ctx, c); err != nil {
		return err
	}

	if !c.DryRun {
		klog.Infof("'%s' rbac created", utils.Bold(c.Identity))
	}

	return nil
}

func uninstallRBACDefault(ctx context.Context, c *Config) error {
	if err := client.GetKubeClient().CoreV1().ServiceAccounts(c.namespace()).Delete(ctx, c.serviceAccountName(), metav1.DeleteOptions{}); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}

	if err := client.GetKubeClient().RbacV1().ClusterRoles().Delete(ctx, c.clusterRoleName(), metav1.DeleteOptions{}); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}

	if err := client.GetKubeClient().RbacV1().ClusterRoleBindings().Delete(ctx, c.roleBindingName(), metav1.DeleteOptions{}); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}

	klog.Infof("'%s' rbac roles deleted", utils.Bold(c.Identity))

	return nil
}
