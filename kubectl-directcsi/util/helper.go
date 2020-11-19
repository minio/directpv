/*
 * This file is part of MinIO Direct CSI
 * Copyright (C) 2020, MinIO, Inc.
 *
 * This code is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License, version 3,
 * as published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License, version 3,
 * along with this program.  If not, see <http://www.gnu.org/licenses/>
 *
 */

package util

import (
	directv1alpha1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
)

// ListVolumesInDrive returns a slice of all the DirectCSIVolumes created on a given DirectCSIDrive
func ListVolumesInDrive(drive directv1alpha1.DirectCSIDrive, volumes *directv1alpha1.DirectCSIVolumeList) []directv1alpha1.DirectCSIVolume {
	vols := make([]directv1alpha1.DirectCSIVolume, 0)
	for _, volume := range volumes.Items {
		if volume.OwnerDrive.Name == drive.Name {
			vols = append(vols, volume)
		}
	}
	return vols
}
