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
	"errors"

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	errPspUnsupported = errors.New("pod security policy is not supported in your kubernetes version")
)

func installPSPDefault(ctx context.Context, i *Config) error {
	info, err := k8s.GetGroupVersionKind("policy", "PodSecurityPolicy", "v1beta1")
	if err != nil {
		return err
	}

	if info.Version == "v1beta1" {
		if err := createPodSecurityPolicy(ctx, i); err != nil {
			return err
		}
		return createPSPClusterRoleBinding(ctx, i)
	}

	return errPspUnsupported
}

func uninstallPSPDefault(ctx context.Context, i *Config) error {
	if err := k8s.KubeClient().RbacV1().ClusterRoleBindings().Delete(ctx, i.getPSPClusterRoleBindingName(), metav1.DeleteOptions{}); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	if err := k8s.KubeClient().PolicyV1beta1().PodSecurityPolicies().Delete(ctx, i.getPSPName(), metav1.DeleteOptions{}); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		return nil
	}
	return i.postProc(nil, "uninstalled '%s' clusterrolebinding, '%s' podsecuritypolicy %s", bold(i.getPSPClusterRoleBindingName()), bold(i.getPSPName()), tick)
}

func createPodSecurityPolicy(ctx context.Context, i *Config) error {
	pspObj := &policy.PodSecurityPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1beta1",
			Kind:       "PodSecurityPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        i.getPSPName(),
			Namespace:   metav1.NamespaceNone,
			Annotations: defaultAnnotations,
			Labels:      defaultLabels,
		},
		Spec: policy.PodSecurityPolicySpec{
			Privileged:          true,
			HostPID:             true,
			HostIPC:             false,
			AllowedCapabilities: []corev1.Capability{policy.AllowAllCapabilities},
			Volumes:             []policy.FSType{policy.HostPath},
			AllowedHostPaths: []policy.AllowedHostPath{
				{PathPrefix: procFSDir, ReadOnly: true},
				{PathPrefix: volumePathSysDir, ReadOnly: true},
				{PathPrefix: consts.UdevDataDir, ReadOnly: true},
				{PathPrefix: consts.AppRootDir},
				{PathPrefix: socketDir},
				{PathPrefix: kubeletDirPath},
			},
			SELinux: policy.SELinuxStrategyOptions{
				Rule: policy.SELinuxStrategyRunAsAny,
			},
			RunAsUser: policy.RunAsUserStrategyOptions{
				Rule: policy.RunAsUserStrategyRunAsAny,
			},
			SupplementalGroups: policy.SupplementalGroupsStrategyOptions{
				Rule: policy.SupplementalGroupsStrategyRunAsAny,
			},
			FSGroup: policy.FSGroupStrategyOptions{
				Rule: policy.FSGroupStrategyRunAsAny,
			},
		},
	}

	if !i.DryRun {
		if _, err := k8s.KubeClient().PolicyV1beta1().PodSecurityPolicies().Create(ctx, pspObj, metav1.CreateOptions{}); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return err
			}
			return nil
		}
	}

	return i.postProc(pspObj, "installed '%s' PodSecurityPolicy %s", bold(i.getPSPName()), tick)
}

func createPSPClusterRoleBinding(ctx context.Context, i *Config) error {
	crb := &rbac.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        i.getPSPClusterRoleBindingName(),
			Namespace:   metav1.NamespaceNone,
			Annotations: defaultAnnotations,
			Labels:      defaultLabels,
		},
		Subjects: []rbac.Subject{
			{
				Kind:     "Group",
				APIGroup: "rbac.authorization.k8s.io",
				Name:     "system:serviceaccounts:" + i.serviceAccountName(),
			},
		},
		RoleRef: rbac.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     i.clusterRoleName(),
		},
	}

	if !i.DryRun {
		if _, err := k8s.KubeClient().RbacV1().ClusterRoleBindings().Create(ctx, crb, metav1.CreateOptions{}); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return err
			}
			return nil
		}
	}

	return i.postProc(crb, "installed '%s' clusterrolebinding %s", bold(i.getPSPClusterRoleBindingName()), tick)
}
