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
	"reflect"

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createOrUpdateSecret(ctx context.Context, args *Args, name string, data map[string][]byte) error {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: map[string]string{},
			Labels:      defaultLabels,
		},
		Data: data,
	}

	if args.DryRun {
		fmt.Print(mustGetYAML(secret))
		return nil
	}

	secretsClient := k8s.KubeClient().CoreV1().Secrets(namespace)
	existingSecret, err := secretsClient.Get(ctx, secret.Name, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		if _, err := secretsClient.Create(ctx, secret, metav1.CreateOptions{}); err != nil {
			return err
		}

		_, err = io.WriteString(args.auditWriter, mustGetYAML(secret))
		return err
	}

	if reflect.DeepEqual(existingSecret.Data, secret.Data) {
		return nil
	}

	existingSecret.Data = secret.Data
	if _, err := secretsClient.Update(ctx, existingSecret, metav1.UpdateOptions{}); err != nil {
		return err
	}

	_, err = io.WriteString(args.auditWriter, mustGetYAML(existingSecret))
	return err
}

func deleteSecret(ctx context.Context, name string) error {
	err := k8s.KubeClient().CoreV1().Secrets(namespace).Delete(
		ctx, name, metav1.DeleteOptions{},
	)
	if err != nil {
		if apierrors.IsNotFound(err) {
			err = nil
		}
	}
	return err
}

func createAdminSecrets(ctx context.Context, args *Args) error {
	return createOrUpdateSecret(
		ctx, args, consts.CredentialsSecretName, args.credential.ToSecretData(),
	)
}

func deleteAdminSecrets(ctx context.Context) error {
	return deleteSecret(ctx, consts.CredentialsSecretName)
}
