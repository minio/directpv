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

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewDriveTypeMeta gets new drive CRD type meta.
func NewDriveTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: string(directpvtypes.LatestVersionLabelKey),
		Kind:       consts.DriveKind,
	}
}

// NewVolumeTypeMeta gets new drive CRD type meta.
func NewVolumeTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: string(directpvtypes.LatestVersionLabelKey),
		Kind:       consts.VolumeKind,
	}
}

// NewNodeTypeMeta gets new node CRD type meta.
func NewNodeTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: string(directpvtypes.LatestVersionLabelKey),
		Kind:       consts.NodeKind,
	}
}

// NewInitRequestTypeMeta gets new node CRD type meta.
func NewInitRequestTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: string(directpvtypes.LatestVersionLabelKey),
		Kind:       consts.InitRequestKind,
	}
}

// GetDriveMountDir returns drive mount directory.
func GetDriveMountDir(fsuuid string) string {
	return path.Join(consts.MountRootDir, fsuuid)
}

// GetDriveMetaDir returns drive meta directory.
func GetDriveMetaDir(fsuuid string) string {
	return path.Join(GetDriveMountDir(fsuuid), "."+consts.AppName)
}

// GetDriveMetaFile returns drive meta file.
func GetDriveMetaFile(fsuuid string) string {
	return path.Join(GetDriveMetaDir(fsuuid), "meta.info")
}

// GetVolumeRootDir returns volume root directory.
func GetVolumeRootDir(fsuuid string) string {
	return path.Join(GetDriveMountDir(fsuuid), ".FSUUID."+fsuuid)
}

// GetVolumeDir returns volume directory.
func GetVolumeDir(fsuuid, volumeName string) string {
	return path.Join(GetVolumeRootDir(fsuuid), volumeName)
}
