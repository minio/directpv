// This file is part of MinIO Kubernetes Cloud
// Copyright (c) 2020 MinIO, Inc.
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

package volume

import (
	"context"
	"os"
	
	"k8s.io/apimachinery/pkg/types"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
)

func GetVolume(ctx context.Context, vID string) (*Volume, error) {
	v := &Volume{}
	err := vClient.Get(ctx, types.NamespacedName{
		Name:      vID,
		Namespace: "",
	}, v)
	if err != nil {
		return v, nil
	}

	return v, nil
}

func (v *Volume) ContainsTargetPaths(targetPath string) (AccessType, bool) {
	for _, b := range v.BlockAccess {
		if b.Link == targetPath {
			return b, true
		}
	}
	for _, m := range v.MountAccess {
		if m.MountPoint == targetPath {
			return m, true
		}
	}
	return nil, false
}

func (v *Volume) IsBlockAccessible() bool {
	if v.VolumeSource.VolumeSourceType == VolumeSourceTypeBlockDevice {
		return true
	}
	return false
}

func (v *Volume) IsMountAccessible() bool {
	vSource := v.VolumeSource.VolumeSourceType
	if vSource == VolumeSourceTypeDirectory || vSource == VolumeSourceTypeBlockDevice {
		return true
	}
	return false
}

// Bind binds the volume to a symlink and presents it as a block device
// inside the container. All access modes are only enforced while provisioning. 
// It is assumed that the container will honor these privileges in good faith
func (v *Volume) Bind(targetPath string, readOnly bool, volContext map[string]string) error {
	if !v.IsBlockAccessible() {
		return status.Error(codes.FailedPrecondition, "volume does not have block access capability")
	}
	if len(v.MountAccess) != 0 {
			return status.Error(codes.FailedPrecondition, "volume capability does not allow provisioning as block device")
	}
	
	if len(v.BlockAccess) != 0 {
		if v.VolumeAccessMode == VolumeAccessModeSingleNodeWriter || v.VolumeAccessMode == VolumeAccessModeSingleNodeReadOnly {
			return status.Error(codes.FailedPrecondition, "volume capability does not allow provisioning for different nodes")
		}

		if v.VolumeAccessMode == VolumeAccessModeMultiNodeReadOnly {
			if !readOnly {
				return status.Error(codes.FailedPrecondition, "volume capability does not allow provisioning with RW access")
			}
		}
		if v.VolumeAccessMode == VolumeAccessModeMultiNodeSingleWriter {
			singleWriterFound := false
			for _, b := range v.BlockAccess {
				if b.Access == AccessRW {
					singleWriterFound = true
					break
				}
			}
			
			if singleWriterFound && !readOnly {
				return status.Error(codes.FailedPrecondition, "volume capability does not allow provisioning with RW access")
			}
		}
	}

	if err := os.Symlink(v.VolumeSource.VolumeSourcePath, targetPath); err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	return nil
}


func (v *Volume) Mount(targetPath string, fs string, mountFlags []string, readOnly bool, volContext map[string]string) error {
	return nil
}
