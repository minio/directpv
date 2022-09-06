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

package types

import (
	"path"

	"github.com/minio/directpv/pkg/consts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewDriveTypeMeta gets new drive CRD type meta.
func NewDriveTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: string(LatestVersionLabelKey),
		Kind:       consts.DriveKind,
	}
}

// NewVolumeTypeMeta gets new drive CRD type meta.
func NewVolumeTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: string(LatestVersionLabelKey),
		Kind:       consts.VolumeKind,
	}
}

func GetDriveMountDir(fsuuid string) string {
	return path.Join(consts.MountRootDir, fsuuid)
}

func GetVolumeRootDir(fsuuid string) string {
	return path.Join(GetDriveMountDir(fsuuid), ".FSUUID."+fsuuid)
}

func GetVolumeDir(fsuuid, volumeID string) string {
	return path.Join(GetVolumeRootDir(fsuuid), volumeID)
}
