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
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func createOrUpdateConversionKeyPairSecret(ctx context.Context, publicCertBytes, privateKeyBytes []byte, c *Config) error {
	secretsClient := client.GetKubeClient().CoreV1().Secrets(c.namespace())

	getCertsDataMap := func() map[string][]byte {
		mp := make(map[string][]byte)
		mp[privateKeyFileName] = privateKeyBytes
		mp[publicCertFileName] = publicCertBytes
		return mp
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        conversionKeyPair,
			Namespace:   c.namespace(),
			Annotations: defaultAnnotations,
			Labels:      defaultLabels,
		},
		Data: getCertsDataMap(),
	}

	if c.DryRun {
		return c.postProc(secret)
	}

	existingSecret, err := secretsClient.Get(ctx, conversionKeyPair, metav1.GetOptions{})
	if err != nil {
		if !k8serror.IsNotFound(err) {
			return err
		}
		if _, err := secretsClient.Create(ctx, secret, metav1.CreateOptions{}); err != nil {
			return err
		}
		return c.postProc(secret)
	}

	existingSecret.Data = secret.Data
	if _, err := secretsClient.Update(ctx, existingSecret, metav1.UpdateOptions{}); err != nil {
		return err
	}

	return c.postProc(secret)
}

func createOrUpdateConversionCACertSecret(ctx context.Context, caCertBytes []byte, c *Config) error {
	secretsClient := client.GetKubeClient().CoreV1().Secrets(c.namespace())

	getCertsDataMap := func() map[string][]byte {
		mp := make(map[string][]byte)
		mp[caCertFileName] = caCertBytes
		return mp
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        conversionCACert,
			Namespace:   c.namespace(),
			Annotations: defaultAnnotations,
			Labels:      defaultLabels,
		},
		Data: getCertsDataMap(),
	}

	if c.DryRun {
		return c.postProc(secret)
	}

	existingSecret, err := secretsClient.Get(ctx, conversionCACert, metav1.GetOptions{})
	if err != nil {
		if !k8serror.IsNotFound(err) {
			return err
		}
		if _, err := secretsClient.Create(ctx, secret, metav1.CreateOptions{}); err != nil {
			return err
		}
		return c.postProc(secret)
	}

	existingSecret.Data = secret.Data
	if _, err := secretsClient.Update(ctx, existingSecret, metav1.UpdateOptions{}); err != nil {
		return err
	}

	return c.postProc(secret)
}

func checkConversionSecrets(ctx context.Context, c *Config) error {
	secretsClient := client.GetKubeClient().CoreV1().Secrets(c.namespace())
	if _, err := secretsClient.Get(ctx, conversionKeyPair, metav1.GetOptions{}); err != nil {
		return err
	}
	_, err := secretsClient.Get(ctx, conversionCACert, metav1.GetOptions{})
	return err
}

func createConversionWebhookSecrets(ctx context.Context, c *Config) error {
	err := checkConversionSecrets(ctx, c)
	if err == nil {
		return nil
	}
	if !k8serror.IsNotFound(err) {
		return err
	}

	caCertBytes, publicCertBytes, privateKeyBytes, certErr := getCerts([]string{c.conversionWebhookDNSName()})
	if certErr != nil {
		return certErr
	}
	c.conversionWebhookCaBundle = caCertBytes

	if err := createOrUpdateConversionKeyPairSecret(ctx, publicCertBytes, privateKeyBytes, c); err != nil {
		return err
	}

	return createOrUpdateConversionCACertSecret(ctx, caCertBytes, c)
}

func deleteConversionSecrets(ctx context.Context, c *Config) error {
	secretsClient := client.GetKubeClient().CoreV1().Secrets(c.namespace())
	if err := secretsClient.Delete(ctx, conversionKeyPair, metav1.DeleteOptions{}); err != nil && !k8serror.IsNotFound(err) {
		return err
	}
	return secretsClient.Delete(ctx, conversionCACert, metav1.DeleteOptions{})
}

func installConversionSecretDefault(ctx context.Context, c *Config) error {
	if err := createConversionWebhookSecrets(ctx, c); err != nil {
		return err
	}
	if !c.DryRun {
		klog.Infof("'%s' conversion webhook secrets created", utils.Bold(c.Identity))
	}
	return nil
}

func uninstallConversionSecretDefault(ctx context.Context, c *Config) error {
	if err := deleteConversionSecrets(ctx, c); err != nil && !k8serror.IsNotFound(err) {
		return err
	}
	klog.Infof("'%s' conversion secrets deleted", utils.Bold(c.Identity))
	return nil
}
