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
	"fmt"
	"io"

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	clusterRoleVerbList   = "list"
	clusterRoleVerbGet    = "get"
	clusterRoleVerbWatch  = "watch"
	clusterRoleVerbCreate = "create"
	clusterRoleVerbDelete = "delete"
	clusterRoleVerbUpdate = "update"
	clusterRoleVerbPatch  = "patch"
)

type rbacTask struct{}

func (rbacTask) Name() string {
	return "RBAC"
}

func (rbacTask) Start(ctx context.Context, args *Args) error {
	if !sendStartMessage(ctx, args.ProgressCh, 3) {
		return errSendProgress
	}
	return nil
}

func (rbacTask) End(ctx context.Context, args *Args, err error) error {
	if !sendEndMessage(ctx, args.ProgressCh, err) {
		return errSendProgress
	}
	return nil
}

func (rbacTask) Execute(ctx context.Context, args *Args) error {
	return createRBAC(ctx, args)
}

func (rbacTask) Delete(ctx context.Context, _ *Args) error {
	return deleteRBAC(ctx)
}

func createServiceAccount(ctx context.Context, args *Args) (err error) {
	if !sendProgressMessage(ctx, args.ProgressCh, "Creating service account", 1, nil) {
		return errSendProgress
	}
	defer func() {
		if err == nil {
			if !sendProgressMessage(ctx, args.ProgressCh, "Created service account", 1, serviceAccountComponent(consts.Identity)) {
				err = errSendProgress
			}
		}
	}()
	serviceAccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        consts.Identity,
			Namespace:   namespace,
			Annotations: map[string]string{},
			Labels:      defaultLabels,
		},
		Secrets:                      []corev1.ObjectReference{},
		ImagePullSecrets:             []corev1.LocalObjectReference{},
		AutomountServiceAccountToken: nil,
	}

	if args.DryRun {
		fmt.Print(mustGetYAML(serviceAccount))
		return nil
	}

	_, err = k8s.KubeClient().CoreV1().ServiceAccounts(namespace).Create(
		ctx, serviceAccount, metav1.CreateOptions{},
	)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			err = nil
		}
		return err
	}

	_, err = io.WriteString(args.auditWriter, mustGetYAML(serviceAccount))
	return err
}

func createClusterRoleBinding(ctx context.Context, args *Args) (err error) {
	if !sendProgressMessage(ctx, args.ProgressCh, "Creating cluster role binding", 3, nil) {
		return errSendProgress
	}
	defer func() {
		if err == nil {
			if !sendProgressMessage(ctx, args.ProgressCh, "Created cluster role binding", 3, clusterRoleBindingComponent(consts.Identity)) {
				err = errSendProgress
			}
		}
	}()
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.Identity,
			Namespace: metav1.NamespaceNone,
			Annotations: map[string]string{
				"rbac.authorization.kubernetes.io/autoupdate": "true",
			},
			Labels: defaultLabels,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      consts.Identity,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     consts.Identity,
		},
	}

	if args.DryRun {
		fmt.Print(mustGetYAML(clusterRoleBinding))
		return nil
	}

	_, err = k8s.KubeClient().RbacV1().ClusterRoleBindings().Create(
		ctx, clusterRoleBinding, metav1.CreateOptions{},
	)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			err = nil
		}
		return err
	}

	_, err = io.WriteString(args.auditWriter, mustGetYAML(clusterRoleBinding))
	return err
}

func createClusterRole(ctx context.Context, args *Args) (err error) {
	if !sendProgressMessage(ctx, args.ProgressCh, "Creating cluster role", 2, nil) {
		return errSendProgress
	}
	defer func() {
		if err == nil {
			if !sendProgressMessage(ctx, args.ProgressCh, "Created cluster role", 2, clusterRoleComponent(consts.Identity)) {
				err = errSendProgress
			}
		}
	}()
	clusterRole := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.Identity,
			Namespace: metav1.NamespaceNone,
			Annotations: map[string]string{
				"rbac.authorization.kubernetes.io/autoupdate": "true",
			},
			Labels: defaultLabels,
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
				Verbs:     []string{"use"},
				Resources: []string{"podsecuritypolicies"},
				APIGroups: []string{"policy"},
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
					consts.GroupName,
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
					consts.DriveResource,
					consts.VolumeResource,
					consts.NodeResource,
					consts.InitRequestResource,
				},
				APIGroups: []string{consts.GroupName},
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
			{
				Verbs: []string{
					clusterRoleVerbGet,
					clusterRoleVerbList,
					clusterRoleVerbWatch,
				},
				Resources: []string{
					"secrets",
					"secret",
				},
				APIGroups: []string{
					"",
				},
			},
		},
		AggregationRule: nil,
	}

	if args.DryRun {
		fmt.Print(mustGetYAML(clusterRole))
		return nil
	}

	_, err = k8s.KubeClient().RbacV1().ClusterRoles().Create(
		ctx, clusterRole, metav1.CreateOptions{},
	)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			err = nil
		}
		return err
	}

	_, err = io.WriteString(args.auditWriter, mustGetYAML(clusterRole))
	return err
}

func createRBAC(ctx context.Context, args *Args) (err error) {
	if err = createServiceAccount(ctx, args); err != nil {
		return err
	}
	if err = createClusterRole(ctx, args); err != nil {
		return err
	}
	return createClusterRoleBinding(ctx, args)
}

func deleteRBAC(ctx context.Context) error {
	err := k8s.KubeClient().CoreV1().ServiceAccounts(namespace).Delete(
		ctx, consts.Identity, metav1.DeleteOptions{},
	)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	err = k8s.KubeClient().RbacV1().ClusterRoles().Delete(
		ctx, consts.Identity, metav1.DeleteOptions{},
	)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	err = k8s.KubeClient().RbacV1().ClusterRoleBindings().Delete(
		ctx, consts.Identity, metav1.DeleteOptions{},
	)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}
