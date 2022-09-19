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
	"math/rand"
	"path"
	"time"

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	defaultLabels = map[string]string{ // labels
		appNameLabel: consts.GroupName,
		appTypeLabel: "CSIDriver",

		string(types.CreatedByLabelKey): pluginName,
		string(types.VersionLabelKey):   consts.LatestAPIVersion,
	}

	defaultAnnotations = map[string]string{}
)

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

func stringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func newRandomString(length int) string {
	return stringWithCharset(length, charset)
}

func newPluginsSocketDir(kubeletDir, name string) string {
	return path.Join(kubeletDir, "plugins", k8s.SanitizeResourceName(name))
}

func newHostPathVolume(name, path string) corev1.Volume {
	hostPathType := corev1.HostPathDirectoryOrCreate
	volumeSource := corev1.VolumeSource{
		HostPath: &corev1.HostPathVolumeSource{
			Path: path,
			Type: &hostPathType,
		},
	}

	return corev1.Volume{
		Name:         name,
		VolumeSource: volumeSource,
	}
}

func newSecretVolume(name, secretName string) corev1.Volume {
	volumeSource := corev1.VolumeSource{
		Secret: &corev1.SecretVolumeSource{
			SecretName: secretName,
		},
	}
	return corev1.Volume{
		Name:         name,
		VolumeSource: volumeSource,
	}
}

func newVolumeMount(name, path string, mountPropogation corev1.MountPropagationMode, readOnly bool) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:             name,
		ReadOnly:         readOnly,
		MountPath:        path,
		MountPropagation: &mountPropogation,
	}
}

func getReadinessHandler() corev1.ProbeHandler {
	return corev1.ProbeHandler{
		HTTPGet: &corev1.HTTPGetAction{
			Path:   consts.ReadinessPath,
			Port:   intstr.FromString(readinessPortName),
			Scheme: corev1.URISchemeHTTP,
		},
	}
}

func generateSanitizedUniqueNameFrom(name string) string {
	sanitizedName := k8s.SanitizeResourceName(name)
	// Max length of name is 255. If needed, cut out last 6 bytes
	// to make room for randomstring
	if len(sanitizedName) >= 255 {
		sanitizedName = sanitizedName[0:249]
	}

	// Get a 5 byte randomstring
	shortUUID := newRandomString(5)

	// Concatenate sanitizedName (249) and shortUUID (5) with a '-' in between
	// Max length of the returned name cannot be more than 255 bytes
	return fmt.Sprintf("%s-%s", sanitizedName, shortUUID)
}

func deleteDeployment(ctx context.Context, identity, name string) error {
	dClient := k8s.KubeClient().AppsV1().Deployments(k8s.SanitizeResourceName(identity))

	getDeleteProtectionFinalizer := func() string {
		return k8s.SanitizeResourceName(identity) + deleteProtectionFinalizer
	}

	clearFinalizers := func(name string) error {
		deployment, err := dClient.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		finalizer := getDeleteProtectionFinalizer()
		deployment.SetFinalizers(k8s.RemoveFinalizer(&deployment.ObjectMeta, finalizer))
		if _, err := dClient.Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
			return err
		}
		return nil
	}

	if err := clearFinalizers(name); err != nil {
		return err
	}

	if err := dClient.Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func createOrUpdateSecret(ctx context.Context, secretName string, dataMap map[string][]byte, c *Config) error {
	secretsClient := k8s.KubeClient().CoreV1().Secrets(c.namespace())
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        secretName,
			Namespace:   c.namespace(),
			Annotations: defaultAnnotations,
			Labels:      defaultLabels,
		},
		Data: dataMap,
	}

	if c.DryRun {
		return c.postProc(secret)
	}

	existingSecret, err := secretsClient.Get(ctx, secret.Name, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		if _, err := secretsClient.Create(ctx, secret, metav1.CreateOptions{}); err != nil {
			return err
		}
		return nil
	}

	existingSecret.Data = secret.Data
	if _, err := secretsClient.Update(ctx, existingSecret, metav1.UpdateOptions{}); err != nil {
		return err
	}
	return nil
}
