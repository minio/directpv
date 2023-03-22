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
	"io"

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

const pspClusterRoleBindingName = "psp-" + consts.Identity

var errPSPUnsupported = errors.New("pod security policy is not supported in your kubernetes version")

type pspTask struct{}

func (pspTask) Name() string {
	return "PSP"
}

func (pspTask) Start(ctx context.Context, args *Args) error {
	if !sendStartMessage(ctx, args.ProgressCh, 2) {
		return errSendProgress
	}
	return nil
}

func (pspTask) End(ctx context.Context, args *Args, err error) error {
	if !sendEndMessage(ctx, args.ProgressCh, err) {
		return errSendProgress
	}
	return nil
}

func (pspTask) Execute(ctx context.Context, args *Args) error {
	return createPSP(ctx, args)
}

func (pspTask) Delete(ctx context.Context, _ *Args) error {
	major, minor, err := getKubeVersion()
	if err != nil {
		return err
	}
	podSecurityAdmission := (major == 1 && minor > 24)
	if podSecurityAdmission {
		return nil
	}
	return deletePSP(ctx)
}

func createPSPClusterRoleBinding(ctx context.Context, args *Args) (err error) {
	if !sendProgressMessage(ctx, args.ProgressCh, "Creating psp cluster role binding", 2, nil) {
		return errSendProgress
	}
	defer func() {
		if err == nil {
			if !sendProgressMessage(ctx, args.ProgressCh, "Created psp cluster role binding", 2, clusterRoleBindingComponent(pspClusterRoleBindingName)) {
				err = errSendProgress
			}
		}
	}()

	update := false
	creationTimestamp := metav1.Time{}
	resourceVersion := ""
	uid := types.UID("")
	if !args.dryRun() {
		crb, err := k8s.KubeClient().RbacV1().ClusterRoleBindings().Get(
			ctx, pspClusterRoleBindingName, metav1.GetOptions{},
		)
		switch {
		case err != nil:
			if !apierrors.IsNotFound(err) {
				return err
			}
		default:
			update = true
			creationTimestamp = crb.CreationTimestamp
			resourceVersion = crb.ResourceVersion
			uid = crb.UID
			if _, err = io.WriteString(args.backupWriter, utils.MustGetYAML(crb)); err != nil {
				return err
			}
		}
	}

	crb := &rbac.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
			ResourceVersion:   resourceVersion,
			UID:               uid,
			Name:              pspClusterRoleBindingName,
			Namespace:         metav1.NamespaceNone,
			Annotations:       map[string]string{},
			Labels:            defaultLabels,
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

	if args.dryRun() {
		args.DryRunPrinter(crb)
		return nil
	}

	if update {
		_, err = k8s.KubeClient().RbacV1().ClusterRoleBindings().Update(ctx, crb, metav1.UpdateOptions{})
	} else {
		_, err = k8s.KubeClient().RbacV1().ClusterRoleBindings().Create(ctx, crb, metav1.CreateOptions{})
	}
	if err != nil {
		return err
	}

	_, err = io.WriteString(args.auditWriter, utils.MustGetYAML(crb))
	return err
}

func createPodSecurityPolicy(ctx context.Context, args *Args) (err error) {
	if !sendProgressMessage(ctx, args.ProgressCh, "Creating pod security policy", 1, nil) {
		return errSendProgress
	}
	defer func() {
		if err == nil {
			if !sendProgressMessage(ctx, args.ProgressCh, "Created pod security policy", 1, podSecurityPolicyComponent(consts.Identity)) {
				err = errSendProgress
			}
		}
	}()

	update := false
	creationTimestamp := metav1.Time{}
	resourceVersion := ""
	uid := types.UID("")
	if !args.dryRun() {
		psp, err := k8s.KubeClient().PolicyV1beta1().PodSecurityPolicies().Get(
			ctx, consts.Identity, metav1.GetOptions{},
		)
		switch {
		case err != nil:
			if !apierrors.IsNotFound(err) {
				return err
			}
		default:
			update = true
			creationTimestamp = psp.CreationTimestamp
			resourceVersion = psp.ResourceVersion
			uid = psp.UID
			if _, err = io.WriteString(args.backupWriter, utils.MustGetYAML(psp)); err != nil {
				return err
			}
		}
	}

	psp := &policy.PodSecurityPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1beta1",
			Kind:       "PodSecurityPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
			ResourceVersion:   resourceVersion,
			UID:               uid,
			Name:              consts.Identity,
			Namespace:         metav1.NamespaceNone,
			Annotations:       map[string]string{},
			Labels:            defaultLabels,
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

	if args.dryRun() {
		args.DryRunPrinter(psp)
		return nil
	}

	if update {
		_, err = k8s.KubeClient().PolicyV1beta1().PodSecurityPolicies().Update(
			ctx, psp, metav1.UpdateOptions{},
		)
	} else {
		_, err = k8s.KubeClient().PolicyV1beta1().PodSecurityPolicies().Create(
			ctx, psp, metav1.CreateOptions{},
		)
	}
	if err != nil {
		return err
	}

	_, err = io.WriteString(args.auditWriter, utils.MustGetYAML(psp))
	return err
}

func createPSP(ctx context.Context, args *Args) error {
	if args.podSecurityAdmission {
		return nil
	}
	var gvk *schema.GroupVersionKind
	if !args.dryRun() {
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
		ctx, pspClusterRoleBindingName, metav1.DeleteOptions{},
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
