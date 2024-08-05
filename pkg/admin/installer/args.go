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
	// csiProvisionerImage = csi-provisioner:v5.0.1
	csiProvisionerImage = "csi-provisioner@sha256:acf6e876a3fc5f367ea3ad5aeb2ce16c47b051cb853d8b2c5fb498c51166726e"
	// csiProvisionerImageV2_2_0 = "csi-provisioner:v2.2.0-go1.18"
	csiProvisionerImageV2_2_0 = "csi-provisioner@sha256:c185db49ba02c384633165894147f8d7041b34b173e82a49d7145e50e809b8d6"
	// nodeDriverRegistrarImage = csi-node-driver-registrar:v2.11.1
	nodeDriverRegistrarImage = "csi-node-driver-registrar@sha256:7cd4053e52c0028bd9c711ca6de667a7c459a024679ae967c7f800e81baec0bf"
	// livenessProbeImage = livenessprobe:v2.13.1
	livenessProbeImage = "livenessprobe@sha256:4106724032620c7679bf065e6e11c7d841cf1d9db69c5e33514a6139681d0689"
	// csiResizerImage = csi-resizer:v1.11.2
	csiResizerImage = "csi-resizer@sha256:33f6fd174f2dc121c0e87176856cc36b25374737c5441bccb5b3679905df7ab9"

	// openshiftCSIProvisionerImage = openshift4/ose-csi-external-provisioner-rhel8:v4.12.0-202407151105.p0.g3aa7c52.assembly.stream.el8
	openshiftCSIProvisionerImage = "registry.redhat.io/openshift4/ose-csi-external-provisioner-rhel8@sha256:8bf8aa8975790e19ba107fd58699f98389e3fb692d192f4df3078fff7f0a4bba"
	// openshiftNodeDriverRegistrarImage = openshift4/ose-csi-node-driver-registrar-rhel8:v4.12.0-202407151105.p0.gc316b89.assembly.stream.el8
	openshiftNodeDriverRegistrarImage = "registry.redhat.io/openshift4/ose-csi-node-driver-registrar-rhel8@sha256:ab54e6a2e8a6a1ca2da5aaf25f784c09f5bf22ea32224ec1bdb6c564f88695a9"
	// openshiftLivenessProbeImage = openshift4/ose-csi-livenessprobe-rhel8:v4.12.0-202407151105.p0.ge6545e7.assembly.stream.el8
	openshiftLivenessProbeImage = "registry.redhat.io/openshift4/ose-csi-livenessprobe-rhel8@sha256:b28029f929fe2a28e666910d1acc57c3474fabdb2f9129688ef1ca56c7231d90"
	// openshiftCSIResizerImage = openshift4/ose-csi-external-resizer-rhel8:v4.12.0-202407151105.p0.g5b066ba.assembly.stream.el8
	openshiftCSIResizerImage = "registry.redhat.io/openshift4/ose-csi-external-resizer-rhel8@sha256:bed8de36bac80108909205342b2d92e4de5adbfa33bf13f9346236fca52a0d3e"
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
