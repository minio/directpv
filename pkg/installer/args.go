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

	"github.com/minio/directpv/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/version"
)

const (
	// csiProvisionerImage = csi-provisioner:v3.5.0
	csiProvisionerImage          = "csi-provisioner@sha256:7b5c070ec70d30b0895d91b10c39a0e6cc81c18e0d1566c77aeff2a3587fa316"
	openshiftCSIProvisionerImage = "registry.redhat.io/openshift4/ose-csi-external-provisioner@sha256:778aa6e5ea046bfcd865e665679c30362dc8c00cfb33a0cdcc56b2395e8ab504"

	// csiProvisionerImageV2_2_0 = "csi-provisioner:v2.2.0-go1.18"
	csiProvisionerImageV2_2_0 = "csi-provisioner@sha256:c185db49ba02c384633165894147f8d7041b34b173e82a49d7145e50e809b8d6"

	// nodeDriverRegistrarImage = csi-node-driver-registrar:v2.8.0
	nodeDriverRegistrarImage          = "csi-node-driver-registrar@sha256:c805fdc166761218dc9478e7ac8e0ad0e42ad442269e75608823da3eb761e67e"
	openshiftNodeDriverRegistrarImage = "registry.redhat.io/openshift4/ose-csi-node-driver-registrar@sha256:0db5ea72a708503516f33f221848f0adaee71901769699ef5322a79e2da4f6d1"

	// livenessProbeImage = livenessprobe:v2.10.0
	livenessProbeImage          = "livenessprobe@sha256:f3bc9a84f149cd7362e4bd0ae8cd90b26ad020c2591bfe19e63ff97aacf806c3"
	openshiftLivenessProbeImage = "registry.redhat.io/openshift4/ose-csi-livenessprobe@sha256:81f9f06a7de9a79013a4690ad616c1aff9638ab64284626491f44645a07051ec"

	// csiResizerImage = csi-resizer:v1.8.0
	csiResizerImage          = "csi-resizer@sha256:819f68a4daf75acec336302843f303cf360d4941249f9f5019ffbb690c8ac7c0"
	openshiftCSIResizerImage = "registry.redhat.io/openshift4/ose-csi-external-resizer-rhel8@sha256:837b32a0c432123e2605ad6d029e7f3b4489d9c52a9d272c7a133d41ad10db87"
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

	podSecurityAdmission     bool
	csiProvisionerImage      string
	nodeDriverRegistrarImage string
	livenessProbeImage       string
	csiResizerImage          string
}

// NewArgs creates arguments for DirectPV installation.
func NewArgs(image string) *Args {
	return &Args{
		image:    image,
		Registry: "quay.io",
		Org:      "minio",

		csiProvisionerImage:      csiProvisionerImage,
		nodeDriverRegistrarImage: nodeDriverRegistrarImage,
		livenessProbeImage:       livenessProbeImage,
		csiResizerImage:          csiResizerImage,
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
