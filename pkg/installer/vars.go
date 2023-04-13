// This file is part of MinIO DirectPV
// Copyright (c) 2022 MinIO, Inc.
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
	"os"
	"path"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const readinessPortName = "readinessport"

var (
	kubeletDirPath = "/var/lib/kubelet"

	defaultLabels = map[string]string{
		"application-name":                      consts.GroupName,
		"application-type":                      "CSIDriver",
		string(directpvtypes.CreatedByLabelKey): pluginName,
		string(directpvtypes.VersionLabelKey):   consts.LatestAPIVersion,
	}

	kubeNodeNameEnvVar = corev1.EnvVar{
		Name: kubeNodeNameEnvVarName,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				APIVersion: "v1",
				FieldPath:  "spec.nodeName",
			},
		},
	}

	csiEndpointEnvVar = corev1.EnvVar{
		Name:  csiEndpointEnvVarName,
		Value: UnixCSIEndpoint,
	}

	commonContainerPorts = []corev1.ContainerPort{
		{
			ContainerPort: consts.ReadinessPort,
			Name:          readinessPortName,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			ContainerPort: healthZContainerPort,
			Name:          healthZContainerPortName,
			Protocol:      corev1.ProtocolTCP,
		},
	}

	readinessHandler = corev1.ProbeHandler{
		HTTPGet: &corev1.HTTPGetAction{
			Path:   consts.ReadinessPath,
			Port:   intstr.FromString(readinessPortName),
			Scheme: corev1.URISchemeHTTP,
		},
	}
)

func init() {
	if value := os.Getenv("KUBELET_DIR_PATH"); value != "" {
		kubeletDirPath = path.Clean(value)
	}
}
