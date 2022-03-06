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
	"path/filepath"
	"strings"
	"time"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	defaultLabels = map[string]string{ // labels
		appNameLabel: "direct.csi.min.io",
		appTypeLabel: "CSIDriver",

		string(utils.CreatedByLabelKey): directCSIPluginName,
		string(utils.VersionLabelKey):   directcsi.Version,
	}

	defaultAnnotations = map[string]string{}
)

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

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

func newDirectCSIPluginsSocketDir(kubeletDir, name string) string {
	return filepath.Join(kubeletDir, "plugins", utils.SanitizeKubeResourceName(name))
}

func getConversionWebhookDNSName(identity string) string {
	return strings.Join([]string{utils.SanitizeKubeResourceName(identity), utils.SanitizeKubeResourceName(identity), "svc"}, ".") // "direct-csi-min-io.direct-csi-min-io.svc"
}

func getConversionHealthzURL(identity string) (conversionWebhookURL string) {
	conversionWebhookDNSName := getConversionWebhookDNSName(identity)
	conversionWebhookURL = fmt.Sprintf("https://%s:%d%s", conversionWebhookDNSName, conversionWebhookPort, healthZContainerPortPath) // https://direct-csi-min-io.direct-csi-min-io.svc:30443/healthz
	return
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

func getConversionHealthzHandler() corev1.Handler {
	return corev1.Handler{
		HTTPGet: &corev1.HTTPGetAction{
			Path:   healthZContainerPortPath,
			Port:   intstr.FromString(conversionWebhookPortName),
			Scheme: corev1.URISchemeHTTPS,
		},
	}
}

func generateSanitizedUniqueNameFrom(name string) string {
	sanitizedName := utils.SanitizeKubeResourceName(name)
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
	dClient := client.GetKubeClient().AppsV1().Deployments(utils.SanitizeKubeResourceName(identity))

	getDeleteProtectionFinalizer := func() string {
		return utils.SanitizeKubeResourceName(identity) + directCSIFinalizerDeleteProtection
	}

	clearFinalizers := func(name string) error {
		deployment, err := dClient.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		finalizer := getDeleteProtectionFinalizer()
		deployment.ObjectMeta.SetFinalizers(utils.RemoveFinalizer(&deployment.ObjectMeta, finalizer))
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
