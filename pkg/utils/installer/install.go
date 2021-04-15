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
	"errors"
	"fmt"
	"path/filepath"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	validationWebhookCaBundle []byte
	conversionWebhookCaBundle []byte
)

// well defined errors
var (
	ErrKubeVersionNotSupported = errors.New("Kubernetes version unsupported")
	ErrEmptyCABundle           = errors.New("CA bundle is empty")
)

func NewObjMeta(name, namespace string, kvs ...string) metav1.ObjectMeta {
	return newObjMeta(name, namespace, kvs...)
}

// Exported
func SanitizeName(s string) string {
	return sanitizeName(s)
}

func newObjMeta(name, namespace string, kvs ...string) metav1.ObjectMeta {
	labels := map[string]string{
		"application": directCSILabel,
	}

	key := ""
	for i := 0; i < len(kvs); i++ {
		// if even, it is a key
		if i%2 == 0 {
			key = kvs[i]
			labels[key] = ""
			continue
		}
		// if odd, it is a value
		value := kvs[i]
		labels[key] = value
	}

	return metav1.ObjectMeta{
		Name:      sanitizeName(name),
		Namespace: sanitizeName(namespace),
		Annotations: map[string]string{
			CreatedByLabel: DirectCSIPluginName,
		},
		Labels: labels,
	}

}


func sanitizeName(s string) string {
	if len(s) == 0 {
		return s
	}

	re := regexp.MustCompile("[^a-zA-Z0-9-]")
	s = re.ReplaceAllString(s, "-")
	if s[len(s)-1] == '-' {
		s = s + "X"
	}
	return s
}

func generateSanitizedUniqueNameFrom(name string) string {
	sanitizedName := sanitizeName(name)
	// Max length of name is 255. If needed, cut out last 6 bytes
	// to make room for randomstring
	if len(sanitizedName) >= 255 {
		sanitizedName = sanitizedName[0:249]
	}

	// Get a 5 byte randomstring
	shortUUID := NewRandomString(5)

	// Concatenate sanitizedName (249) and shortUUID (5) with a '-' in between
	// Max length of the returned name cannot be more than 255 bytes
	return fmt.Sprintf("%s-%s", sanitizedName, shortUUID)
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

func newDirectCSIPluginsSocketDir(kubeletDir, name string) string {
	return filepath.Join(kubeletDir, "plugins", sanitizeName(name))
}

func newVolumeMount(name, path string, bidirectional bool) corev1.VolumeMount {
	mountProp := corev1.MountPropagationNone
	if bidirectional {
		mountProp = corev1.MountPropagationBidirectional
	}
	return corev1.VolumeMount{
		Name:             name,
		MountPath:        path,
		MountPropagation: &mountProp,
	}
}


