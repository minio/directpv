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

	"github.com/minio/directpv/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	podsecurityadmission "k8s.io/pod-security-admission/api"
)

func installNSDefault(ctx context.Context, i *Config) error {
	name := i.namespace()
	annotations := defaultAnnotations
	if i.enablePodSecurityAdmission {
		// Policy violations will cause the pods to be rejected
		annotations[podsecurityadmission.EnforceLevelLabel] = string(podsecurityadmission.LevelPrivileged)
	}
	ns := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   metav1.NamespaceNone,
			Annotations: defaultAnnotations,
			Labels:      defaultLabels,
			Finalizers:  []string{metav1.FinalizerDeleteDependents}, // foregroundDeletion finalizer
		},
	}

	if !i.DryRun {
		if _, err := k8s.KubeClient().CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
	}

	return i.postProc(ns, "installed '%s' namespace %s", bold(name), tick)
}

func uninstallNSDefault(ctx context.Context, i *Config) error {
	foregroundDeletePropagation := metav1.DeletePropagationForeground
	if err := k8s.KubeClient().CoreV1().Namespaces().Delete(ctx, i.namespace(), metav1.DeleteOptions{
		PropagationPolicy: &foregroundDeletePropagation,
	}); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return i.postProc(nil, "uninstalled '%s' namespace %s", bold(i.namespace()), tick)
}
