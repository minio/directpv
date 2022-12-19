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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createNodeAPIService(ctx context.Context, args *Args) error {
	nodeAPIPort := corev1.ServicePort{
		Port: consts.NodeAPIPort,
		Name: consts.NodeAPIPortName,
	}
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        consts.NodeAPIServerHLSVC,
			Namespace:   namespace,
			Annotations: map[string]string{},
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

	if args.DryRun {
		fmt.Print(mustGetYAML(service))
		return nil
	}

	_, err := k8s.KubeClient().CoreV1().Services(namespace).Create(
		ctx, service, metav1.CreateOptions{},
	)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			err = nil
		}
		return err
	}

	_, err = io.WriteString(args.auditWriter, mustGetYAML(service))
	return err
}

func deleteNodeAPIService(ctx context.Context) error {
	err := k8s.KubeClient().CoreV1().Services(namespace).Delete(
		ctx, consts.NodeAPIServerHLSVC, metav1.DeleteOptions{},
	)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}
