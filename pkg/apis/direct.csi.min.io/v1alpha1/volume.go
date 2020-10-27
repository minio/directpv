// This file is part of MinIO Direct CSI
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

package v1alpha1

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/pborman/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/utils/mount"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	sc      = runtime.NewScheme()
	vClient client.Client

	group   = "direct.csi.min.io"
	version = "v1alpha1"

	GroupVersion = schema.GroupVersion{
		Group:   group,
		Version: version,
	}
)

func VolumeClient(basePaths []string) error {
	glog.V(10).Infof("base paths: %s", strings.Join(basePaths, ","))

	InitializeFactory(basePaths)
	clientgoscheme.AddToScheme(sc)
	AddToScheme(sc)

	// init volume client
	c, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("could not get kubeconfig: %v", err)
	}

	extCl, err := apiextensions.NewForConfig(c)
	if err != nil {
		return fmt.Errorf("could not initialize apiExtentions Client: %v", err)
	}

	_, err = extCl.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "volumes.direct.csi.min.io", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("volume type not yet registered: %v", err)
	}

	mapper := func(c *rest.Config) meta.RESTMapper {
		m, err := apiutil.NewDynamicRESTMapper(c)
		if err != nil {
			glog.Errorf("unable to initialize rest mapper: %v", err)
			panic(err)
		}
		return m
	}(c)

	vc, err := client.New(c, client.Options{
		Scheme: sc,
		Mapper: mapper,
	})
	if err != nil {
		return fmt.Errorf("unable to initialize volume client: %v", err)
	}
	vClient = vc
	return nil
}

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

func (in *VolumeList) DeepCopy() *VolumeList {
	if in == nil {
		return nil
	}
	out := new(VolumeList)
	in.DeepCopyInto(out)
	return out
}

func (in *VolumeList) DeepCopyInto(out *VolumeList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)

	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Volume, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (in *Volume) DeepCopy() *Volume {
	if in == nil {
		return nil
	}
	out := new(Volume)
	in.DeepCopyInto(out)
	return out
}

func (in *Volume) DeepCopyInto(out *Volume) {
	*out = *in
	out.TypeMeta = in.TypeMeta

	out.VolumeID = in.VolumeID
	out.Name = in.Name
	in.VolumeSource.DeepCopyInto(&out.VolumeSource)
	out.VolumeStatus = in.VolumeStatus
	out.NodeID = in.NodeID

	out.StagingPath = in.StagingPath
	out.VolumeAccessMode = in.VolumeAccessMode

	for _, in := range in.BlockAccess {
		o := new(BlockAccessType)
		in.DeepCopyInto(o)
		out.BlockAccess = append(out.BlockAccess, *o)
	}

	for _, in := range in.MountAccess {
		o := new(MountAccessType)
		in.DeepCopyInto(o)
		out.MountAccess = append(out.MountAccess, *o)
	}

	mapDeepCopyInto := func(in map[string]string, out map[string]string) {
		if out == nil {
			out = make(map[string]string, len(in))
		}
		for k, v := range in {
			out[k] = v
		}
	}

	mapDeepCopyInto(in.PublishContext, out.PublishContext)
	mapDeepCopyInto(in.Parameters, out.Parameters)

	in.TopologyConstraint.DeepCopyInto(out.TopologyConstraint)
}

func (in *VolumeSource) DeepCopyInto(out *VolumeSource) {
	if out == nil {
		out = new(VolumeSource)
	}

	out.VolumeSourceType = in.VolumeSourceType
	out.VolumeSourcePath = in.VolumeSourcePath
}

func (in *MountAccessType) DeepCopyInto(out *MountAccessType) {
	out.FsType = in.FsType

	func(o []MountFlag) {
		if o == nil {
			o = make([]MountFlag, len(in.MountFlags))
		}
		copy(o, in.MountFlags)
	}(out.MountFlags)

	out.MountPoint = in.MountPoint
}

func (in *BlockAccessType) DeepCopyInto(out *BlockAccessType) {
	out.Device = in.Device
	out.Link = in.Link
}
