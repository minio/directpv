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

package utils

import (
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/minio/direct-csi/pkg/sys"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
