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

	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/utils"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func deleteConversionDeployment(ctx context.Context, identity string) error {
	return deleteDeployment(ctx, identity, conversionWebhookDeploymentName)
}

func deleteLegacyConversionSecret(ctx context.Context, identity string) error {
	return client.GetKubeClient().CoreV1().Secrets(utils.SanitizeKubeResourceName(identity)).Delete(ctx, conversionWebhookSecretName, metav1.DeleteOptions{})
}

func deleteLegacyConversionWebhookCertsSecret(ctx context.Context, identity string) error {
	if err := client.GetKubeClient().CoreV1().Secrets(utils.SanitizeKubeResourceName(identity)).Delete(ctx, conversionWebhookCertsSecret, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func deleteLegacyConversionDeployment(ctx context.Context, identity string) error {
	if err := deleteConversionDeployment(ctx, identity); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	if err := deleteLegacyConversionSecret(ctx, identity); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	if err := deleteLegacyConversionWebhookCertsSecret(ctx, identity); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}
