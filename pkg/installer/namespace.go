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

	"github.com/minio/directpv/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	podsecurityadmissionapi "k8s.io/pod-security-admission/api"
)

const (
	totalNamespaceSteps = 1
)

func createNamespace(ctx context.Context, args *Args) (err error) {
	if !sendProgressMessage(ctx, args.ProgressCh, "Creating namespace", 1, nil) {
		return errSendProgress
	}
	defer func() {
		if err == nil {
			if !sendProgressMessage(ctx, args.ProgressCh, "Created namespace", 1, namespaceComponent(namespace)) {
				err = errSendProgress
			}
		}
	}()
	annotations := map[string]string{}
	if args.podSecurityAdmission {
		// Policy violations will cause the pods to be rejected
		annotations[podsecurityadmissionapi.EnforceLevelLabel] = string(podsecurityadmissionapi.LevelPrivileged)
	}

	ns := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        namespace,
			Namespace:   metav1.NamespaceNone,
			Annotations: annotations,
			Labels:      defaultLabels,
			Finalizers:  []string{metav1.FinalizerDeleteDependents},
		},
	}

	if args.DryRun {
		fmt.Print(mustGetYAML(ns))
		return nil
	}
	_, err = k8s.KubeClient().CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			err = nil
		}
		return err
	}
	_, err = io.WriteString(args.auditWriter, mustGetYAML(ns))
	return err
}

func deleteNamespace(ctx context.Context) error {
	propagationPolicy := metav1.DeletePropagationForeground
	err := k8s.KubeClient().CoreV1().Namespaces().Delete(
		ctx, namespace, metav1.DeleteOptions{PropagationPolicy: &propagationPolicy},
	)
	if err != nil {
		if apierrors.IsNotFound(err) {
			err = nil
		}
	}
	return err
}
