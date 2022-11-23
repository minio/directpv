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
	"fmt"
	"io"

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var errPSPUnsupported = errors.New("pod security policy is not supported in your kubernetes version")

func createPSPClusterRoleBinding(ctx context.Context, args *Args) error {
	crb := &rbac.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "psp-" + consts.Identity,
			Namespace:   metav1.NamespaceNone,
			Annotations: map[string]string{},
			Labels:      defaultLabels,
		},
		Subjects: []rbac.Subject{
			{
				Kind:     "Group",
				APIGroup: "rbac.authorization.k8s.io",
				Name:     "system:serviceaccounts:" + consts.Identity,
			},
		},
		RoleRef: rbac.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     consts.Identity,
		},
	}

	if args.DryRun {
		fmt.Print(mustGetYAML(crb))
		return nil
	}

	_, err := k8s.KubeClient().RbacV1().ClusterRoleBindings().Create(ctx, crb, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			err = nil
		}
		return err
	}

	_, err = io.WriteString(args.auditWriter, mustGetYAML(crb))
	return err
}

func createPodSecurityPolicy(ctx context.Context, args *Args) error {
	psp := &policy.PodSecurityPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1beta1",
			Kind:       "PodSecurityPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        consts.Identity,
			Namespace:   metav1.NamespaceNone,
			Annotations: map[string]string{},
			Labels:      defaultLabels,
		},
		Spec: policy.PodSecurityPolicySpec{
			Privileged:          true,
			HostPID:             true,
			HostIPC:             false,
			AllowedCapabilities: []corev1.Capability{policy.AllowAllCapabilities},
			Volumes:             []policy.FSType{policy.HostPath},
			AllowedHostPaths: []policy.AllowedHostPath{
				{PathPrefix: "/proc", ReadOnly: true},
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

	if args.DryRun {
		fmt.Print(mustGetYAML(psp))
		return nil
	}

	_, err := k8s.KubeClient().PolicyV1beta1().PodSecurityPolicies().Create(
		ctx, psp, metav1.CreateOptions{},
	)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			err = nil
		}
		return err
	}

	_, err = io.WriteString(args.auditWriter, mustGetYAML(psp))
	return err
}

func createPSP(ctx context.Context, args *Args) error {
	var gvk *schema.GroupVersionKind
	if !args.DryRun {
		var err error
		if gvk, err = k8s.GetGroupVersionKind("policy", "PodSecurityPolicy", "v1beta1"); err != nil {
			return err
		}
	} else {
		gvk = &schema.GroupVersionKind{Version: "v1beta1"}
	}

	if gvk.Version == "v1beta1" {
		if err := createPodSecurityPolicy(ctx, args); err != nil {
			return err
		}
		return createPSPClusterRoleBinding(ctx, args)
	}

	return errPSPUnsupported
}

func deletePSP(ctx context.Context) error {
	err := k8s.KubeClient().RbacV1().ClusterRoleBindings().Delete(
		ctx, "psp-"+consts.Identity, metav1.DeleteOptions{},
	)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	err = k8s.KubeClient().PolicyV1beta1().PodSecurityPolicies().Delete(
		ctx, consts.Identity, metav1.DeleteOptions{},
	)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}
