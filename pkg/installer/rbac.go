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

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	createVerb = "create"
	deleteVerb = "delete"
	getVerb    = "get"
	listVerb   = "list"
	patchVerb  = "patch"
	updateVerb = "update"
	useVerb    = "use"
	watchVerb  = "watch"
)

func newPolicyRule(resources []string, apiGroups []string, verbs ...string) rbacv1.PolicyRule {
	if apiGroups == nil {
		apiGroups = []string{""}
	}
	return rbacv1.PolicyRule{
		APIGroups: apiGroups,
		Resources: resources,
		Verbs:     verbs,
	}
}

type rbacTask struct{}

func (rbacTask) Name() string {
	return "RBAC"
}

func (rbacTask) Start(ctx context.Context, args *Args) error {
	if !sendStartMessage(ctx, args.ProgressCh, 5) {
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
			Name:      consts.Identity,
			Namespace: namespace,
			Annotations: map[string]string{
				string(directpvtypes.PluginVersionLabelKey): args.PluginVersion,
			},
			Labels: defaultLabels,
		},
		Secrets:                      []corev1.ObjectReference{},
		ImagePullSecrets:             []corev1.LocalObjectReference{},
		AutomountServiceAccountToken: nil,
	}

	if !args.DryRun && !args.Declarative {
		_, err = k8s.KubeClient().CoreV1().ServiceAccounts(namespace).Create(
			ctx, serviceAccount, metav1.CreateOptions{},
		)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
	}

	return args.writeObject(serviceAccount)
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
				string(directpvtypes.PluginVersionLabelKey):   args.PluginVersion,
			},
			Labels: defaultLabels,
		},
		Rules: []rbacv1.PolicyRule{
			newPolicyRule([]string{"persistentvolumes"}, nil, createVerb, deleteVerb, getVerb, listVerb, patchVerb, watchVerb),
			newPolicyRule([]string{"persistentvolumeclaims/status"}, nil, patchVerb),
			newPolicyRule([]string{"podsecuritypolicies"}, []string{"policy"}, useVerb),
			newPolicyRule([]string{"persistentvolumeclaims"}, nil, getVerb, listVerb, updateVerb, watchVerb),
			newPolicyRule([]string{"storageclasses"}, []string{"storage.k8s.io"}, getVerb, listVerb, watchVerb),
			newPolicyRule([]string{"events"}, nil, createVerb, listVerb, patchVerb, updateVerb, watchVerb),
			newPolicyRule([]string{"volumesnapshots"}, []string{"snapshot.storage.k8s.io"}, getVerb, listVerb),
			newPolicyRule([]string{"volumesnapshotcontents"}, []string{"snapshot.storage.k8s.io"}, getVerb, listVerb),
			newPolicyRule([]string{"csinodes"}, []string{"storage.k8s.io"}, getVerb, listVerb, watchVerb),
			newPolicyRule([]string{"nodes"}, nil, getVerb, listVerb, watchVerb),
			newPolicyRule([]string{"volumeattachments"}, []string{"storage.k8s.io"}, getVerb, listVerb, watchVerb),
			newPolicyRule([]string{"endpoints"}, nil, createVerb, deleteVerb, getVerb, listVerb, updateVerb, watchVerb),
			newPolicyRule([]string{"leases"}, []string{"coordination.k8s.io"}, createVerb, deleteVerb, getVerb, listVerb, updateVerb, watchVerb),
			newPolicyRule(
				[]string{"customresourcedefinitions", "customresourcedefinition"},
				[]string{"apiextensions.k8s.io", consts.GroupName},
				createVerb, deleteVerb, getVerb, listVerb, patchVerb, updateVerb, watchVerb,
			),
			newPolicyRule(
				[]string{consts.DriveResource, consts.VolumeResource, consts.NodeResource, consts.InitRequestResource},
				[]string{consts.GroupName},
				createVerb, deleteVerb, getVerb, listVerb, updateVerb, watchVerb,
			),
			newPolicyRule([]string{"pods", "pod"}, nil, getVerb, listVerb, watchVerb),
			newPolicyRule([]string{"secrets", "secret"}, nil, getVerb, listVerb, watchVerb),
			newPolicyRule([]string{"jobs"}, []string{"batch"}, createVerb, deleteVerb, getVerb, listVerb, updateVerb, watchVerb),
		},
		AggregationRule: nil,
	}

	if !args.DryRun && !args.Declarative {
		_, err = k8s.KubeClient().RbacV1().ClusterRoles().Create(
			ctx, clusterRole, metav1.CreateOptions{},
		)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
	}

	return args.writeObject(clusterRole)
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
				string(directpvtypes.PluginVersionLabelKey):   args.PluginVersion,
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

	if !args.DryRun && !args.Declarative {
		_, err = k8s.KubeClient().RbacV1().ClusterRoleBindings().Create(
			ctx, clusterRoleBinding, metav1.CreateOptions{},
		)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
	}

	return args.writeObject(clusterRoleBinding)
}

func createRole(ctx context.Context, args *Args) (err error) {
	if !sendProgressMessage(ctx, args.ProgressCh, "Creating role", 4, nil) {
		return errSendProgress
	}
	defer func() {
		if err == nil {
			if !sendProgressMessage(ctx, args.ProgressCh, "Created role", 4, roleComponent(consts.Identity)) {
				err = errSendProgress
			}
		}
	}()
	role := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.Identity,
			Namespace: namespace,
			Annotations: map[string]string{
				"rbac.authorization.kubernetes.io/autoupdate": "true",
				string(directpvtypes.PluginVersionLabelKey):   args.PluginVersion,
			},
			Labels: defaultLabels,
		},
		Rules: []rbacv1.PolicyRule{
			newPolicyRule([]string{"leases"}, []string{"coordination.k8s.io"}, createVerb, deleteVerb, getVerb, listVerb, updateVerb, watchVerb),
		},
	}

	if !args.DryRun && !args.Declarative {
		_, err = k8s.KubeClient().RbacV1().Roles(namespace).Create(
			ctx, role, metav1.CreateOptions{},
		)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
	}

	return args.writeObject(role)
}

func createRoleBinding(ctx context.Context, args *Args) (err error) {
	if !sendProgressMessage(ctx, args.ProgressCh, "Creating role binding", 5, nil) {
		return errSendProgress
	}
	defer func() {
		if err == nil {
			if !sendProgressMessage(ctx, args.ProgressCh, "Created role binding", 5, roleBindingComponent(consts.Identity)) {
				err = errSendProgress
			}
		}
	}()
	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.Identity,
			Namespace: namespace,
			Annotations: map[string]string{
				"rbac.authorization.kubernetes.io/autoupdate": "true",
				string(directpvtypes.PluginVersionLabelKey):   args.PluginVersion,
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
			Kind:     "Role",
			Name:     consts.Identity,
		},
	}

	if !args.DryRun && !args.Declarative {
		_, err = k8s.KubeClient().RbacV1().RoleBindings(namespace).Create(
			ctx, roleBinding, metav1.CreateOptions{},
		)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
	}

	return args.writeObject(roleBinding)
}

func createRBAC(ctx context.Context, args *Args) (err error) {
	if err = createServiceAccount(ctx, args); err != nil {
		return err
	}
	if err = createClusterRole(ctx, args); err != nil {
		return err
	}
	if err = createClusterRoleBinding(ctx, args); err != nil {
		return err
	}
	if err = createRole(ctx, args); err != nil {
		return err
	}
	return createRoleBinding(ctx, args)
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

	err = k8s.KubeClient().RbacV1().Roles(namespace).Delete(
		ctx, consts.Identity, metav1.DeleteOptions{},
	)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	err = k8s.KubeClient().RbacV1().RoleBindings(namespace).Delete(
		ctx, consts.Identity, metav1.DeleteOptions{},
	)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}
