// This file is part of MinIO DirectPV
// Copyright (c) 2021, 2022 MinIO, Inc.
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

package installer

import (
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"

	"github.com/minio/directpv/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/version"
)

const (
	// csiProvisionerImage = csi-provisioner:v5.2.0-0
	csiProvisionerImage = "csi-provisioner@sha256:24816a743663d153060f0c3fc30005f05bea23c8b0fd2551fd667042741e8562"
	// csiProvisionerImageV2_2_0 = csi-provisioner:v2.2.0-go1.18
	csiProvisionerImageV2_2_0 = "csi-provisioner@sha256:c185db49ba02c384633165894147f8d7041b34b173e82a49d7145e50e809b8d6"
	// nodeDriverRegistrarImage = csi-node-driver-registrar:v2.13.0-0
	nodeDriverRegistrarImage = "csi-node-driver-registrar@sha256:8f00013a19bd3bea021f3e96b093814fb110b383c0fd050d2a5309afb0ed0ccb"
	// livenessProbeImage = livenessprobe:v2.15.0-0
	livenessProbeImage = "livenessprobe@sha256:d8f7d431a2a148970dcb337f24b265d173bcee58bbeeae9af7ae60f01ce49be2"
	// csiResizerImage = csi-resizer:v1.13.1-0
	csiResizerImage = "csi-resizer@sha256:fc0c1f9cbc0ebb16283c0e425c031041eedb0e8ebbe6a1adec3de41584548ce6"

	// openshiftCSIProvisionerImage = registry.redhat.io/openshift4/ose-csi-external-provisioner-rhel8:v4.15.0-202504220035.p0.gce5a1a3.assembly.stream.el8
	openshiftCSIProvisionerImage = "registry.redhat.io/openshift4/ose-csi-external-provisioner-rhel8@sha256:ffc107c70d24eb86d2120d4bce217bcfebb46694817217f134ead9e9b72d2ff3"
	// openshiftNodeDriverRegistrarImage = registry.redhat.io/openshift4/ose-csi-node-driver-registrar-rhel8:v4.15.0-202504220035.p0.g9005584.assembly.stream.el8
	openshiftNodeDriverRegistrarImage = "registry.redhat.io/openshift4/ose-csi-node-driver-registrar-rhel8@sha256:9006d4a80df02b51102961e70e55ae08e2757fc4c62d3c2605b8ffed3a344e8d"
	// openshiftLivenessProbeImage = registry.redhat.io/openshift4/ose-csi-livenessprobe-rhel8:v4.15.0-202504220035.p0.g240bb8c.assembly.stream.el8
	openshiftLivenessProbeImage = "registry.redhat.io/openshift4/ose-csi-livenessprobe-rhel8@sha256:9309cf88631e1cbb5362081a65f2de752cc130d231e8ba2bdd9ef640b51a6b66"
	// openshiftCSIResizerImage = registry.redhat.io/openshift4/ose-csi-external-resizer-rhel8:v4.15.0-202504220035.p0.g3b4236d.assembly.stream.el8
	openshiftCSIResizerImage = "registry.redhat.io/openshift4/ose-csi-external-resizer-rhel8@sha256:ed845b49de7e1c363bc8e8485a04288c03125a3cff7c7f100d35b5b88645725a"
)

// Args represents DirectPV installation arguments.
type Args struct {
	image string

	// Optional arguments
	Registry         string
	Org              string
	ImagePullSecrets []string
	NodeSelector     map[string]string
	Tolerations      []corev1.Toleration
	SeccompProfile   string
	AppArmorProfile  string
	Quiet            bool
	KubeVersion      *version.Version
	Legacy           bool
	ObjectWriter     io.Writer
	DryRun           bool
	Declarative      bool
	Openshift        bool
	ObjectMarshaler  func(runtime.Object) ([]byte, error)
	ProgressCh       chan<- Message
	ForceUninstall   bool
	PluginVersion    string

	podSecurityAdmission     bool
	csiProvisionerImage      string
	nodeDriverRegistrarImage string
	livenessProbeImage       string
	csiResizerImage          string
	imageTag                 string
}

var imageTagRegex = regexp.MustCompile(`:([^/]+)$`)

// NewArgs creates arguments for DirectPV installation.
func NewArgs(image string) *Args {
	imageTag := "dev"
	matchIndex := imageTagRegex.FindStringSubmatchIndex(image)
	if len(matchIndex) > 0 && len(image) > matchIndex[0]+1 {
		imageTag = image[matchIndex[0]+1:]
	}
	return &Args{
		image:    image,
		Registry: "quay.io",
		Org:      "minio",

		csiProvisionerImage:      csiProvisionerImage,
		nodeDriverRegistrarImage: nodeDriverRegistrarImage,
		livenessProbeImage:       livenessProbeImage,
		csiResizerImage:          csiResizerImage,
		imageTag:                 imageTag,
	}
}

func (args *Args) validate() error {
	if args.image == "" {
		return errors.New("image name must be provided")
	}

	if !args.DryRun && !args.Declarative && args.ObjectWriter == nil {
		return errors.New("object writer must be provided")
	}

	if args.DryRun && args.ObjectMarshaler == nil {
		return errors.New("object converter must be provided")
	}

	if args.KubeVersion == nil {
		return errors.New("kubeversion is not set")
	}

	return nil
}

func (args *Args) writeObject(obj runtime.Object) (err error) {
	var data []byte
	if args.ObjectMarshaler != nil {
		data, err = args.ObjectMarshaler(obj)
	} else {
		data, err = utils.ToYAML(obj)
	}
	if err != nil {
		return err
	}

	if args.ObjectWriter != nil {
		_, err = args.ObjectWriter.Write(data)
	} else {
		fmt.Print(string(data))
	}

	return err
}

func (args *Args) getImagePullSecrets() (refs []corev1.LocalObjectReference) {
	for _, name := range args.ImagePullSecrets {
		refs = append(refs, corev1.LocalObjectReference{Name: name})
	}
	return refs
}

func (args *Args) getContainerImage() string {
	return path.Join(args.Registry, args.Org, args.image)
}

func (args *Args) getNodeDriverRegistrarImage() string {
	if args.Openshift {
		return openshiftNodeDriverRegistrarImage
	}
	return path.Join(args.Registry, args.Org, args.nodeDriverRegistrarImage)
}

func (args *Args) getLivenessProbeImage() string {
	if args.Openshift {
		return openshiftLivenessProbeImage
	}
	return path.Join(args.Registry, args.Org, args.livenessProbeImage)
}

func (args *Args) getCSIProvisionerImage() string {
	if args.Openshift {
		return openshiftCSIProvisionerImage
	}
	return path.Join(args.Registry, args.Org, args.csiProvisionerImage)
}

func (args *Args) getCSIResizerImage() string {
	if args.Openshift {
		return openshiftCSIResizerImage
	}
	return path.Join(args.Registry, args.Org, args.csiResizerImage)
}
