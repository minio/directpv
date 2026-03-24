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
	// csiProvisionerImage = csi-provisioner:v6.2.0-0
	csiProvisionerImage = "csi-provisioner@sha256:f83e880ce4290b1ef4fa15a588138eafecdb40106208a5295c0b24d03cbaddbd"
	// csiProvisionerImageV2_2_0 = csi-provisioner:v2.2.0-go1.18
	csiProvisionerImageV2_2_0 = "csi-provisioner@sha256:c185db49ba02c384633165894147f8d7041b34b173e82a49d7145e50e809b8d6"
	// nodeDriverRegistrarImage = csi-node-driver-registrar:v2.16.0-0
	nodeDriverRegistrarImage = "csi-node-driver-registrar@sha256:183b3ac969d133457595fa2abfd9d81d20e83a6bc4606375662d036f56462dde"
	// livenessprobeImage = livenessprobe:v2.18.0-0
	livenessProbeImage = "livenessprobe@sha256:af8bac7b24bbfcc064e58d45c1c2ebaf75b9ac71315a604e0870100fa6aed8da"
	// csiResizerImage = csi-resizer:v2.1.0-0
	csiResizerImage = "csi-resizer@sha256:cb338f5c5a9f781f289b6f25fedebbeeb4eec9fda2aeb2c0a1eaa8529c4c9738"

	// openshiftCSIProvisionerImage = registry.redhat.io/openshift4/ose-csi-external-provisioner-rhel8:v4.15
	openshiftCSIProvisionerImage = "registry.redhat.io/openshift4/ose-csi-external-provisioner-rhel8@sha256:ecf86bed1b174e57b9b52ebf5c2792da25d7ab2daccef15bdac98a47aa09ff3e"
	// openshiftNodeDriverRegistrarImage = registry.redhat.io/openshift4/ose-csi-node-driver-registrar-rhel8:v4.15
	openshiftNodeDriverRegistrarImage = "registry.redhat.io/openshift4/ose-csi-node-driver-registrar-rhel8@sha256:b61fcc8774f2dde2ed46df52630b495ff8f5bb53a80a4733006b0197297b0264"
	// openshiftLivenessProbeImage = registry.redhat.io/openshift4/ose-csi-livenessprobe-rhel8:v4.15
	openshiftLivenessProbeImage = "registry.redhat.io/openshift4/ose-csi-livenessprobe-rhel8@sha256:a516448355b6c360953151ed14c1b9eaf977a674a9ff65216d3076ddc9355987"
	// openshiftCSIResizerImage = registry.redhat.io/openshift4/ose-csi-external-resizer-rhel8:v4.15
	openshiftCSIResizerImage = "registry.redhat.io/openshift4/ose-csi-external-resizer-rhel8@sha256:370f6a90b4792ac9275b355f17b457c8348d3230fd2d272c8a447513ba3473b8"
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
