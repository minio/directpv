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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func installNSDefault(ctx context.Context, i *Config) error {
	name := i.namespace()
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

	if i.DryRun {
		return i.postProc(ns)
	}

	// Create Namespace Obj
	if _, err := client.GetKubeClient().CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{}); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}

	klog.Infof("'%s' namespace created", utils.Bold(name))

	return i.postProc(ns)
}

func deleteNS(ctx context.Context, name string) error {
	foregroundDeletePropagation := metav1.DeletePropagationForeground
	if err := client.GetKubeClient().CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &foregroundDeletePropagation,
	}); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func uninstallNSDefault(ctx context.Context, i *Config) error {
	// Delete Namespace Obj
	name := i.namespace()
	if err := deleteNS(ctx, name); err != nil {
		return err
	}

	klog.Infof("'%s' namespace deleted", utils.Bold(name))

	return nil
}
