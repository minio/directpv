/*
 * This file is part of MinIO Direct CSI
 * Copyright (C) 2021, MinIO, Inc.
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

package installer

import (
	"context"

	"github.com/minio/direct-csi/pkg/utils"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateRBACRoles creates SA, ClusterRole and CRBs
func CreateRBACRoles(ctx context.Context, identity string, dryRun bool) error {
	if err := createServiceAccount(ctx, identity, dryRun); err != nil {
		return err
	}
	if err := createClusterRole(ctx, identity, dryRun); err != nil {
		return err
	}
	if err := createClusterRoleBinding(ctx, identity, dryRun); err != nil {
		return err
	}
	return nil
}

func createServiceAccount(ctx context.Context, identity string, dryRun bool) error {
	serviceAccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta:                   objMeta(identity),
		Secrets:                      []corev1.ObjectReference{},
		ImagePullSecrets:             []corev1.LocalObjectReference{},
		AutomountServiceAccountToken: nil,
	}

	if dryRun {
		return utils.LogYAML(serviceAccount)
	}

	if _, err := utils.GetKubeClient().CoreV1().ServiceAccounts(sanitizeName(identity)).Create(ctx, serviceAccount, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func createClusterRoleBinding(ctx context.Context, identity string, dryRun bool) error {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: objMeta(identity),
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sanitizeName(identity),
				Namespace: sanitizeName(identity),
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     sanitizeName(identity),
		},
	}

	clusterRoleBinding.Annotations["rbac.authorization.kubernetes.io/autoupdate"] = "true"

	if dryRun {
		return utils.LogYAML(clusterRoleBinding)
	}

	if _, err := utils.GetKubeClient().RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func createClusterRole(ctx context.Context, identity string, dryRun bool) error {
	clusterRole := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: objMeta(identity),
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

	if dryRun {
		return utils.LogYAML(clusterRole)
	}

	if _, err := utils.GetKubeClient().RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

// RemoveRBACRoles deletes SA, ClusterRole and CRBs
func RemoveRBACRoles(ctx context.Context, identity string) error {
	if err := removeServiceAccount(ctx, identity); err != nil {
		return err
	}
	if err := removeClusterRole(ctx, identity); err != nil {
		return err
	}
	if err := removeClusterRoleBinding(ctx, identity); err != nil {
		return err
	}
	return nil
}

func removeServiceAccount(ctx context.Context, identity string) error {
	return utils.GetKubeClient().CoreV1().ServiceAccounts(sanitizeName(identity)).Delete(ctx, sanitizeName(identity), metav1.DeleteOptions{})
}

func removeClusterRoleBinding(ctx context.Context, identity string) error {
	return utils.GetKubeClient().RbacV1().ClusterRoleBindings().Delete(ctx, sanitizeName(identity), metav1.DeleteOptions{})
}

func removeClusterRole(ctx context.Context, identity string) error {
	return utils.GetKubeClient().RbacV1().ClusterRoles().Delete(ctx, sanitizeName(identity), metav1.DeleteOptions{})
}
