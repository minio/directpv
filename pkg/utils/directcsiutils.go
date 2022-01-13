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

package utils

import (
	"path/filepath"
	"strings"

	"github.com/minio/directpv/pkg/sys"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SanitizeDrivePath converts older v1.3 formatted drive name to device name.
func SanitizeDrivePath(in string) string {
	path := strings.ReplaceAll(in, sys.DirectCSIPartitionInfix, "")
	path = strings.ReplaceAll(path, sys.DirectCSIDevRoot+"/", "")
	path = strings.ReplaceAll(path, sys.HostDevRoot+"/", "")
	return filepath.Base(path)
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
