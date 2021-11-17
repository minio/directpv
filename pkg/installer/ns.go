// This file is part of MinIO Direct CSI
// Copyright (c) 2021 MinIO, Inc.
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

	"github.com/minio/direct-csi/pkg/client"
	"github.com/minio/direct-csi/pkg/utils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ Installer = &nsInstaller{}

type nsInstaller struct {
	name string

	*installConfig
}

func (n *nsInstaller) Install(ctx context.Context) error {
	if n.installConfig == nil {
		return errInstallationFailed("bad arguments: empty configuration", "Namespace")
	}

	ns := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
		ObjectMeta: metav1.ObjectMeta{
			Name:        utils.SanitizeKubeResourceName(n.name),
			Namespace:   metav1.NamespaceNone,
			Annotations: defaultAnnotations,
			Labels:      defaultLabels,
			Finalizers:  []string{metav1.FinalizerDeleteDependents}, // foregroundDeletion finalizer
		},
	}

	// Create Namespace Obj
	createdNS, err := client.GetKubeClient().CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{
		DryRun: n.getDryRunDirectives(),
	})
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}

	return n.PostProc(createdNS)
}

func (n *nsInstaller) Uninstall(ctx context.Context) error {
	if n.installConfig == nil {
		return errInstallationFailed("bad arguments: empty configuration", "Namespace")
	}

	nsName := utils.SanitizeKubeResourceName(n.name)
	foregroundDeletePropagation := metav1.DeletePropagationForeground

	// Delete Namespace Obj
	if err := client.GetKubeClient().CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{
		DryRun:            n.getDryRunDirectives(),
		PropagationPolicy: &foregroundDeletePropagation,
	}); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
