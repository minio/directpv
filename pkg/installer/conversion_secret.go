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

	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func deleteConversionSecrets(ctx context.Context, c *Config) error {
	secretsClient := client.GetKubeClient().CoreV1().Secrets(c.namespace())
	if err := secretsClient.Delete(ctx, conversionKeyPair, metav1.DeleteOptions{}); err != nil && !k8serror.IsNotFound(err) {
		return err
	}
	return secretsClient.Delete(ctx, conversionCACert, metav1.DeleteOptions{})
}

func uninstallConversionSecretDefault(ctx context.Context, c *Config) error {
	if err := deleteConversionSecrets(ctx, c); err != nil && !k8serror.IsNotFound(err) {
		return err
	}
	klog.Infof("'%s' conversion secrets deleted", utils.Bold(c.Identity))
	return nil
}
