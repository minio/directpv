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

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func installServiceDefault(ctx context.Context, c *Config) error {
	if err := createNodeAPIService(ctx, c); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	// Add more services here..
	return nil
}

func uninstallServiceDefault(ctx context.Context, c *Config) error {
	if err := k8s.KubeClient().CoreV1().Services(c.namespace()).Delete(ctx, consts.NodeAPIServerHLSVC, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func createNodeAPIService(ctx context.Context, c *Config) error {
	nodeAPIPort := corev1.ServicePort{
		Port: consts.NodeAPIPort,
		Name: consts.NodeAPIPortName,
	}
	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        consts.NodeAPIServerHLSVC,
			Namespace:   c.namespace(),
			Annotations: defaultAnnotations,
			Labels:      defaultLabels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{nodeAPIPort},
			Selector: map[string]string{
				serviceSelector: selectorValueEnabled,
			},
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: corev1.ClusterIPNone,
		},
	}

	if c.DryRun {
		return c.postProc(svc)
	}

	if _, err := k8s.KubeClient().CoreV1().Services(c.namespace()).Create(ctx, svc, metav1.CreateOptions{}); err != nil {
		return err
	}
	return c.postProc(svc)
}
