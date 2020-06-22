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
	"os"

	corev1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	"github.com/golang/glog"
)

var (
	sc      = runtime.NewScheme()
	vClient client.Client

	GroupVersion = schema.GroupVersion{
		Group:   "jbod.csi.min.io",
		Version: "v1alpha1",
	}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme
)

func init() {
	clientgoscheme.AddToScheme(sc)
	corev1.AddToScheme(sc)
	AddToScheme(sc)

	// init volume client
	c := config.GetConfigOrDie()
	mapper := func(c *rest.Config) meta.RESTMapper {
		m, err := apiutil.NewDynamicRESTMapper(c)
		if err != nil {
			glog.Errorf("unable to initialize rest mapper: %v", err)
			os.Exit(1)
		}
		return m
	}(c)

	vc, err := client.New(c, client.Options{
		Scheme: sc,
		Mapper: mapper,
	})
	if err != nil {
		glog.Errorf("unable to initialize volume client: %v", err)
		os.Exit(1)
	}
	vClient = vc
}

func (in *VolumeList) DeepCopy() *VolumeList {
	if in == nil {
		return nil
	}
	out := new(VolumeList)
	in.DeepCopyInto(out)
	return out
}

func (in *VolumeList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
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

func (in *Volume) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
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
