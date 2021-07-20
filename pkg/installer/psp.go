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
	policy "k8s.io/api/policy/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createPodSecurityPolicy(ctx context.Context, identity string, dryRun bool) error {
	psp := &policy.PodSecurityPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1beta1",
			Kind:       "PodSecurityPolicy",
		},
		ObjectMeta: objMeta(identity),
		Spec: policy.PodSecurityPolicySpec{
			Privileged: true,
			HostPID:    true,
			HostIPC:    true,
			AllowedHostPaths: []policy.AllowedHostPath{
				{PathPrefix: "/proc", ReadOnly: true},
				{PathPrefix: "/sys", ReadOnly: true},
				{PathPrefix: "/var/lib/direct-csi"},
				{PathPrefix: "/csi"},
				{PathPrefix: "/var/lib/kubelet"},
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

	if dryRun {
		utils.LogYAML(psp)
	} else if _, err := utils.GetKubeClient().PolicyV1beta1().PodSecurityPolicies().Create(ctx, psp, metav1.CreateOptions{}); err != nil {
		return err
	}

	crb := &rbac.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.SanitizeKubeResourceName("psp-" + identity),
			Namespace: utils.SanitizeKubeResourceName(identity),
			Annotations: map[string]string{
				CreatedByLabel: DirectCSIPluginName,
			},
			Labels: map[string]string{
				"app":  DirectCSI,
				"type": CSIDriver,
			},
		},
		Subjects: []rbac.Subject{
			{
				Kind:     "Group",
				APIGroup: "rbac.authorization.k8s.io",
				Name:     "system:serviceaccounts:" + utils.SanitizeKubeResourceName(identity),
			},
		},
		RoleRef: rbac.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     utils.SanitizeKubeResourceName(identity),
		},
	}

	if dryRun {
		return utils.LogYAML(crb)
	}

	_, err := utils.GetKubeClient().RbacV1().ClusterRoleBindings().Create(ctx, crb, metav1.CreateOptions{})
	return err
}

func CreatePodSecurityPolicy(ctx context.Context, identity string, dryRun bool) error {
	info, err := utils.GetGroupKindVersions("policy", "PodSecurityPolicy", "v1beta1")
	if err != nil {
		return err
	}

	if info.Version == "v1beta1" {
		return createPodSecurityPolicy(ctx, identity, dryRun)
	}

	return ErrKubeVersionNotSupported
}
