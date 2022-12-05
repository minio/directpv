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
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"path"
	"strings"

	"github.com/minio/directpv/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

func mustGetYAML(i interface{}) string {
	data, err := yaml.Marshal(i)
	if err != nil {
		klog.Fatalf("unable to marshal object to YAML; %w", err)
	}
	return fmt.Sprintf("%v\n---\n", string(data))
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

func newPluginsSocketDir(kubeletDir, name string) string {
	return path.Join(kubeletDir, "plugins", k8s.SanitizeResourceName(name))
}

func newVolumeMount(name, path string, mountPropogation corev1.MountPropagationMode, readOnly bool) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:             name,
		ReadOnly:         readOnly,
		MountPath:        path,
		MountPropagation: &mountPropogation,
	}
}

func getRandSuffix() string {
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		klog.Fatalf("unable to generate random bytes; %v", err)
	}
	return strings.ToLower(base32.StdEncoding.EncodeToString(b)[:5])
}
