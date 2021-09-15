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

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/topology"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	PodNameLabel      = NewDirectCSILabel("pod.name")
	PodNamespaceLabel = NewDirectCSILabel("pod.namespace")

	NodeLabel       = NewDirectCSILabel("node")
	DriveLabel      = NewDirectCSILabel("drive")
	DrivePathLabel  = NewDirectCSILabel("path")
	AccessTierLabel = NewDirectCSILabel("access-tier")

	VersionLabel   = NewDirectCSILabel("version")
	CreatedByLabel = NewDirectCSILabel("created-by")

	ReservedDrivePathLabel = NewDirectCSILabel("drive-path")

	DirectCSIGroupVersion = SanitizeLabelK(directcsi.Group + "/" + directcsi.Version)
)

func NewDirectCSILabel(key string) string {
	return SanitizeLabelK(directcsi.Group + "/" + key)
}

func SanitizeDrivePath(in string) string {
	path := strings.ReplaceAll(in, sys.DirectCSIPartitionInfix, "")
	path = strings.ReplaceAll(path, sys.DirectCSIDevRoot+"/", "")
	path = strings.ReplaceAll(path, sys.HostDevRoot+"/", "")
	return filepath.Base(path)
}

func SetAccessTierLabel(obj metav1.Object, accessTier directcsi.AccessTier) {
	labels := SafeGetLabels(obj)
	labels[AccessTierLabel] = SanitizeLabelV(string(accessTier))
	obj.SetLabels(labels)
}

func NewIdentityTopologySelector(identity string) corev1.TopologySelectorTerm {
	return corev1.TopologySelectorTerm{
		MatchLabelExpressions: []corev1.TopologySelectorLabelRequirement{
			{
				Key: SanitizeLabelK(topology.TopologyDriverIdentity),
				Values: []string{
					SanitizeLabelV(identity),
				},
			},
		},
	}
}

func DirectCSIDriveTypeMeta() metav1.TypeMeta {
	return NewTypeMeta(DirectCSIGroupVersion, "DirectCSIDrive")
}

func DirectCSIVolumeTypeMeta() metav1.TypeMeta {
	return NewTypeMeta(DirectCSIGroupVersion, "DirectCSIVolume")
}
