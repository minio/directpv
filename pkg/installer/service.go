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
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
)

func installServiceDefault(ctx context.Context, c *Config) error {
	if err := createService(ctx, c); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}
	if !c.DryRun {
		klog.Infof("'%s' service created", utils.Bold(c.serviceName()))
	}
	return nil
}

func uninstallServiceDefault(ctx context.Context, c *Config) error {
	if err := client.GetKubeClient().CoreV1().Services(c.namespace()).Delete(ctx, c.serviceName(), metav1.DeleteOptions{}); err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	klog.Infof("'%s' service deleted", utils.Bold(c.serviceName()))
	return nil
}

func createService(ctx context.Context, c *Config) error {
	csiPort := corev1.ServicePort{
		Port: 12345,
		Name: "unused",
	}
	webhookPort := corev1.ServicePort{
		Name: conversionWebhookPortName,
		Port: conversionWebhookPort,
		TargetPort: intstr.IntOrString{
			Type:   intstr.String,
			StrVal: conversionWebhookPortName,
		},
	}
	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        c.serviceName(),
			Namespace:   c.namespace(),
			Annotations: defaultAnnotations,
			Labels:      defaultLabels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{csiPort, webhookPort},
			Selector: map[string]string{
				webhookSelector: selectorValueEnabled,
			},
		},
	}

	if c.DryRun {
		return c.postProc(svc)
	}

	if _, err := client.GetKubeClient().CoreV1().Services(c.namespace()).Create(ctx, svc, metav1.CreateOptions{}); err != nil {
		return err
	}
	return c.postProc(svc)
}
