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

package client

import (
	"path/filepath"
	"strings"

	"github.com/minio/direct-csi/pkg/sys"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"k8s.io/klog/v2"

	corev1 "k8s.io/api/core/v1"
)

// GetClientForNonCoreGroupKindVersions gets client for group/kind of given versions.
func GetClientForNonCoreGroupKindVersions(group, kind string, versions ...string) (rest.Interface, *schema.GroupVersionKind, error) {
	gvk, err := GetGroupKindVersions(group, kind, versions...)
	if err != nil {
		return nil, nil, err
	}

	gv := &schema.GroupVersion{
		Group:   gvk.Group,
		Version: gvk.Version,
	}

	config, err := GetKubeConfig()
	if err != nil {
		klog.Fatalf("could not find client configuration: %v", err)
	}
	klog.V(1).Infof("obtained client config successfully")

	config.GroupVersion = gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	client, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, nil, err
	}
	return client, gvk, nil
}

// DirectCSIDriveTypeMeta gets new direct-csi drive meta.
func DirectCSIDriveTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: string(DirectCSIVersionLabelKey),
		Kind:       "DirectCSIDrive",
	}
}

// DirectCSIVolumeTypeMeta gets new direct-csi volume meta.
func DirectCSIVolumeTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: string(DirectCSIVersionLabelKey),
		Kind:       "DirectCSIVolume",
	}
}

// SanitizeDrivePath sanitizes drive path.
func SanitizeDrivePath(in string) string {
	path := strings.ReplaceAll(in, sys.DirectCSIPartitionInfix, "")
	path = strings.ReplaceAll(path, sys.DirectCSIDevRoot+"/", "")
	path = strings.ReplaceAll(path, sys.HostDevRoot+"/", "")
	return filepath.Base(path)
}

// NewIdentityTopologySelector creates identity topology selector.
func NewIdentityTopologySelector(identity string) corev1.TopologySelectorTerm {
	return corev1.TopologySelectorTerm{
		MatchLabelExpressions: []corev1.TopologySelectorLabelRequirement{
			{
				Key:    string(TopologyDriverIdentity),
				Values: []string{string(NewLabelValue(identity))},
			},
		},
	}
}
