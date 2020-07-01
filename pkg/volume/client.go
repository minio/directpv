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
	"encoding/json"
	"fmt"
	"os"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	runtime "k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	group   = "jbod.csi.min.io"
	version = "v1alpha1"

	GroupVersion = schema.GroupVersion{
		Group:   group,
		Version: version,
	}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme
)

func InitializeClient(identity string) {
	// Register Volume
	SchemeBuilder.Register(&Volume{}, &VolumeList{})
	clientgoscheme.AddToScheme(sc)
	AddToScheme(sc)

	// init volume client
	c, err := config.GetConfig()
	if err != nil {
		glog.Errorf("could not get kubeconfig: %v", err)
		os.Exit(1)
	}

	extCl, err := apiextensions.NewForConfig(c)
	if err != nil {
		glog.Errorf("could not initialize apiExtentions Client: %v", err)
		os.Exit(1)
	}
	vCrd, err := extCl.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "volumes.jbod.csi.min.io", metav1.GetOptions{})
	if err != nil {
		glog.Errorf("volume type not yet registered: %v", err)

		field := func(in *apiextensionsv1.JSONSchemaProps, name, description, typ string) *apiextensionsv1.JSONSchemaProps {
			out := &apiextensionsv1.JSONSchemaProps{
				Description: description,
				Type:        typ,
				Properties:  map[string]apiextensionsv1.JSONSchemaProps{},
				Items:       new(apiextensionsv1.JSONSchemaPropsOrArray),
			}

			if in.Properties == nil {
				in.Properties = make(map[string]apiextensionsv1.JSONSchemaProps)
			}
			if in.Items == nil {
				in.Items = new(apiextensionsv1.JSONSchemaPropsOrArray)
			}
			in.Properties[name] = *out
			return out
		}

		volDef := &apiextensionsv1.JSONSchemaProps{
			XEmbeddedResource: false,
			Type:              "object",
		}
		field(volDef, "apiVersion", "APIVersion", "string")
		field(volDef, "kind", "", "string")

		metadataDef := field(volDef, "metadata", "", "object")
		field(metadataDef, "name", "", "string")

		field(volDef, "volumeID", "", "string")
		field(volDef, "name", "", "string")

		volSrc := field(volDef, "volumeSource", "", "object")
		field(volSrc, "volumeSourceType", "", "string")
		field(volSrc, "volumeSourcePath", "", "string")

		field(volDef, "volumeStatus", "", "string")
		field(volDef, "nodeID", "", "string")
		field(volDef, "stagingPath", "", "string")
		field(volDef, "volumeAccessMode", "", "integer")

		blockProps := &apiextensionsv1.JSONSchemaProps{
			Type: "object",
		}
		field(blockProps, "device", "", "string")
		field(blockProps, "link", "", "string")
		field(blockProps, "access", "", "string")

		block := field(volDef, "blockAccess", "", "array")
		block.Items.Schema = blockProps

		glog.Infof("block.Items: %#v", block.Items)

		mountProps := &apiextensionsv1.JSONSchemaProps{
			Type: "object",
		}
		field(mountProps, "fsType", "", "string")
		field(mountProps, "mountpoint", "", "string")
		field(mountProps, "access", "", "string")

		mFlagProps := &apiextensionsv1.JSONSchemaProps{
			Type: "string",
		}
		mFlags := field(mountProps, "mountFlags", "", "array")
		mFlags.Items.Schema = mFlagProps

		mount := field(volDef, "mountAccess", "", "array")
		mount.Items.Schema = mountProps

		field(volDef, "publishContext", "", "object")
		field(volDef, "parameters", "", "object")
		field(volDef, "topologyConstraint", "", "object")
		field(volDef, "auditTrail", "", "object")

		crdSpec := &apiextensionsv1.CustomResourceDefinitionSpec{
			Group: group,
			Scope: apiextensionsv1.ClusterScoped,
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:   "volumes",
				Singular: "volume",
				ShortNames: []string{
					"vol",
				},
				Kind:     "Volume",
				ListKind: "VolumeList",
			},
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1alpha1",
					Served:  true,
					Storage: true,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: volDef,
					},
				},
			},
		}
		apiextensionsv1.SetDefaults_CustomResourceDefinitionSpec(crdSpec)

		vCrd = &apiextensionsv1.CustomResourceDefinition{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "CustomResourceDefinition",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("volumes.%s", group),
			},
			Spec:   *crdSpec,
			Status: apiextensionsv1.CustomResourceDefinitionStatus{},
		}
		apiextensionsv1.SetDefaults_CustomResourceDefinition(vCrd)

		if out, err := json.MarshalIndent(vCrd, " ", "  "); err != nil {
			glog.Errorf("could not marshal volume defintion: %v", err)
			os.Exit(1)
		} else {
			fmt.Printf("%s\n", string(out))
		}

		_, err = extCl.ApiextensionsV1().CustomResourceDefinitions().Create(context.Background(), vCrd, metav1.CreateOptions{})
		if err != nil {
			glog.Errorf("could not create and register volumes.jbod.csi.min.io type: %v", err)
			os.Exit(1)
		}
	}

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
