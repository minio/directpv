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

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/jobs"
	"github.com/minio/directpv/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	podsecurityadmissionapi "k8s.io/pod-security-admission/api"
)

type namespaceTask struct{}

func (namespaceTask) Name() string {
	return "Namespace"
}

func (namespaceTask) Start(ctx context.Context, args *Args) error {
	if !sendStartMessage(ctx, args.ProgressCh, 1) {
		return errSendProgress
	}
	return nil
}

func (namespaceTask) Execute(ctx context.Context, args *Args) error {
	return createNamespace(ctx, args)
}

func (namespaceTask) End(ctx context.Context, args *Args, err error) error {
	if !sendEndMessage(ctx, args.ProgressCh, err) {
		return errSendProgress
	}
	return nil
}

func (namespaceTask) Delete(ctx context.Context, _ *Args) error {
	return deleteNamespace(ctx)
}

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

	labels := func() map[string]string {
		if !args.podSecurityAdmission {
			return defaultLabels
		}

		labels := map[string]string{}
		for key, value := range defaultLabels {
			labels[key] = value
		}

		// Policy violations will cause the pods to be rejected
		labels[podsecurityadmissionapi.EnforceLevelLabel] = string(podsecurityadmissionapi.LevelPrivileged)
		return labels
	}()

	ns := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespace,
			Namespace: metav1.NamespaceNone,
			Annotations: map[string]string{
				string(directpvtypes.PluginVersionLabelKey): args.PluginVersion,
			},
			Labels:     labels,
			Finalizers: []string{metav1.FinalizerDeleteDependents},
		},
	}

	if !args.DryRun && !args.Declarative {
		_, err = k8s.KubeClient().CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
	}

	return args.writeObject(ns)
}

func deleteNamespace(ctx context.Context) error {
	jobObjects, err := jobs.NewLister().IgnoreNotFound(true).Get(ctx)
	if err != nil {
		return err
	}
	if len(jobObjects) > 0 {
		// Do not delete the namespace if there
		// are jobs in directpv namespace
		return nil
	}
	propagationPolicy := metav1.DeletePropagationForeground
	err = k8s.KubeClient().CoreV1().Namespaces().Delete(
		ctx, namespace, metav1.DeleteOptions{PropagationPolicy: &propagationPolicy},
	)
	if err != nil {
		if apierrors.IsNotFound(err) {
			err = nil
		}
	}
	return err
}
