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

	"github.com/golang/glog"
	"github.com/pborman/uuid"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/mount"
)

func NewVolume(ctx context.Context, name string, volumeAccessMode VolumeAccessMode, nodeID string, parameters map[string]string) (*Volume, error) {
	vID := uuid.NewUUID().String()
	vol := &Volume{
		TypeMeta: metav1.TypeMeta{
			APIVersion: version,
			Kind:       "volume",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: vID,
		},
		Name:             name,
		VolumeID:         vID,
		VolumeAccessMode: volumeAccessMode,
		NodeID:           nodeID,
		Parameters:       parameters,
	}

	return vol, vClient.Create(ctx, vol)
}

func GetVolume(ctx context.Context, vID string) (*Volume, error) {
	v := &Volume{
		VolumeID: vID,
	}
	err := vClient.Get(ctx, types.NamespacedName{
		Name:      vID,
		Namespace: "",
	}, v)
	if err != nil {
		return nil, err
	}

	return v, nil
}

func DeleteVolume(ctx context.Context, vID string) error {
	return vClient.Delete(ctx, &Volume{
		VolumeID: vID,
	})
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
	if vSource == VolumeSourceTypeDirectory ||
		vSource == VolumeSourceTypeBlockDevice {
		return true
	}
	return false
}

// Bind binds the volume to a symlink and presents it as a block device
// inside the container. All access modes are only enforced while provisioning.
// It is assumed that the container will honor these privileges in good faith
func (v *Volume) Bind(ctx context.Context, targetPath string, readOnly bool, volContext map[string]string) error {
	if !v.IsBlockAccessible() {
		return status.Error(codes.FailedPrecondition, "volume does not have block access capability")
	}
	if len(v.MountAccess) != 0 {
		return status.Error(codes.FailedPrecondition, "volume capability does not allow provisioning as block device")
	}

	if len(v.BlockAccess) != 0 {
		if v.VolumeAccessMode == VolumeAccessModeSingleNodeWriter ||
			v.VolumeAccessMode == VolumeAccessModeSingleNodeReadOnly {
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

	access := AccessRW
	if readOnly {
		access = AccessRO
	}

	deviceID, _, err := mount.GetDeviceNameFromMount(mount.New(""), v.VolumeSource.VolumeSourcePath)
	if err != nil {
		glog.Errorf("could not get deviceID: %v", err)
		deviceID = v.VolumeSource.VolumeSourcePath
	}
	v.BlockAccess = append(v.BlockAccess, BlockAccessType{
		Device: deviceID,
		Link:   targetPath,
		Access: access,
	})

	return vClient.Update(ctx, v)
}

func (v *Volume) Mount(ctx context.Context, targetPath string, fsType string, mountFlags []string, readOnly bool, volContext map[string]string) error {
	if !v.IsMountAccessible() {
		return status.Error(codes.FailedPrecondition, "volume does not have mount access capability")
	}

	if len(v.BlockAccess) != 0 {
		return status.Error(codes.FailedPrecondition, "volume capability does not allow provisioning as mounted directory")
	}

	if len(v.MountAccess) != 0 {
		if v.VolumeAccessMode == VolumeAccessModeSingleNodeWriter ||
			v.VolumeAccessMode == VolumeAccessModeSingleNodeReadOnly {
			return status.Error(codes.FailedPrecondition, "volume capability does not allow provisioning for different nodes")
		}

		if v.VolumeAccessMode == VolumeAccessModeMultiNodeReadOnly {
			if !readOnly {
				return status.Error(codes.FailedPrecondition, "volume capability does not allow provisioning with RW access")
			}
		}

		if v.VolumeAccessMode == VolumeAccessModeMultiNodeSingleWriter {
			singleWriterFound := false

			for _, b := range v.MountAccess {
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

	options := []string{"bind"}
	access := AccessRW

	if readOnly {
		access = AccessRO
		options = append(options, "ro")
	}

	for _, f := range mountFlags {
		options = append(options, f)
	}

	mounter := mount.New("")
	notMount, err := mount.IsNotMountPoint(mounter, targetPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return status.Errorf(codes.Internal, "error checking path %s for mount: %s", targetPath, err)
		}
		notMount = true
	}
	if !notMount {
		glog.V(5).Infof("Skipping bind-mounting subpath %s: already mounted", targetPath)
		return nil
	}

	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return err
	}

	if err := mounter.Mount(v.VolumeSource.VolumeSourcePath, targetPath, fsType, options); err != nil {
		return err
	}

	v.MountAccess = append(v.MountAccess, MountAccessType{
		FsType: FsType(fsType),
		MountFlags: func() []MountFlag {
			mf := make([]MountFlag, len(mountFlags))
			for _, m := range mountFlags {
				mf = append(mf, MountFlag(m))
			}
			return mf
		}(),
		MountPoint: targetPath,
		Access:     access,
	})

	return vClient.Update(ctx, v)
}

func (v *Volume) UnpublishVolume(ctx context.Context, targetPath string) error {
	for i, b := range v.BlockAccess {
		if b.Link == targetPath {
			if err := os.Remove(b.Link); err != nil {
				return err
			}
			v.BlockAccess = append(v.BlockAccess[:i], v.BlockAccess[i+1:]...)
			return vClient.Update(ctx, v)
		}
	}

	for i, m := range v.MountAccess {
		if m.MountPoint == targetPath {
			// Unmount only if the target path is really a mount point.
			if notMnt, err := mount.IsNotMountPoint(mount.New(""), targetPath); err != nil {
				if !os.IsNotExist(err) {
					return status.Error(codes.Internal, err.Error())
				}
			} else if !notMnt {
				// Unmounting the image or filesystem.
				err = mount.New("").Unmount(targetPath)
				if err != nil {
					return status.Error(codes.Internal, err.Error())
				}
			}
			v.MountAccess = append(v.MountAccess[:i], v.MountAccess[i+1:]...)
			return vClient.Update(ctx, v)
		}
	}
	return nil
}

func (v *Volume) StageVolume(ctx context.Context, volumeID, stagePath string) error {
	if v.StagingPath != "" {
		if v.StagingPath != stagePath {
			return status.Error(codes.FailedPrecondition, "volume staging path does not match old staging path")
		}
		return nil
	}

	dir, err := Provision(volumeID)
	if err != nil {
		return status.Errorf(codes.Internal, "volume provisioning failed: %v", err)
	}

	if err := os.MkdirAll(stagePath, 0755); err != nil {
		return err
	}

	if err := mount.New("").Mount(dir, stagePath, "", []string{"bind"}); err != nil {
		return err
	}

	v.VolumeSource = VolumeSource{
		VolumeSourceType: VolumeSourceTypeDirectory,
		VolumeSourcePath: dir,
	}
	v.StagingPath = stagePath

	return vClient.Update(ctx, v)
}

func (v *Volume) UnstageVolume(ctx context.Context, volumeID, stagePath string) error {
	// Unmount only if the target path is really a mount point.
	if notMnt, err := mount.IsNotMountPoint(mount.New(""), stagePath); err != nil {
		if !os.IsNotExist(err) {
			return status.Error(codes.Internal, err.Error())
		}
	} else if !notMnt {
		// Unmounting the image or filesystem.
		err = mount.New("").Unmount(stagePath)
		if err != nil {
			return status.Error(codes.Internal, err.Error())
		}
	}
	if err := Unprovision(v.VolumeSource.VolumeSourcePath); err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	
	v.StagingPath = ""
	v.VolumeSource = VolumeSource{}
	
	return vClient.Update(ctx, v)
}
