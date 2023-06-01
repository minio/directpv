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
	"io"
	"path"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/version"
)

const (
	// csiProvisionerImage = csi-provisioner:v3.4.0
	csiProvisionerImage = "csi-provisioner@sha256:704fe68a6344774d4d0fde980af64fac2f2ddd27fb2e7f7c5b3d8fbddeae4ec8"

	// csiProvisionerImageV2_2_0 = "csi-provisioner:v2.2.0-go1.18"
	csiProvisionerImageV2_2_0 = "csi-provisioner@sha256:c185db49ba02c384633165894147f8d7041b34b173e82a49d7145e50e809b8d6"

	// nodeDriverRegistrarImage = csi-node-driver-registrar:v2.6.3
	nodeDriverRegistrarImage = "csi-node-driver-registrar@sha256:68ee8f0b10acb4189e506d8a0e40c995362d886a35d5cbb17624e59913af0145"

	// livenessProbeImage = livenessprobe:v2.9.0
	livenessProbeImage = "livenessprobe@sha256:0522eff1d8e9269655080500c1f6388fe2573978e8a74e08beaf3452cd575c2e"

	// csiResizerImage = csi-resizer:v1.7.0
	csiResizerImage = "csi-resizer@sha256:a88ca4a9bfbd2e604aedae5a04a5c180540259e3ab75393755ff73d587a619b2"
)

// Args represents DirectPV installation arguments.
type Args struct {
	image       string
	auditWriter io.Writer

	// Optional arguments
	Registry                 string
	Org                      string
	ImagePullSecrets         []string
	NodeSelector             map[string]string
	Tolerations              []corev1.Toleration
	SeccompProfile           string
	AppArmorProfile          string
	Quiet                    bool
	KubeVersion              *version.Version
	Legacy                   bool
	podSecurityAdmission     bool
	csiProvisionerImage      string
	nodeDriverRegistrarImage string
	livenessProbeImage       string
	csiResizerImage          string
	ProgressCh               chan<- Message
	ForceUninstall           bool
	DryRunPrinter            func(interface{})
}

func (args Args) dryRun() bool {
	return args.DryRunPrinter != nil
}

// NewArgs creates arguments for DirectPV installation.
func NewArgs(image string, auditWriter io.Writer) (*Args, error) {
	args := &Args{
		image:       image,
		auditWriter: auditWriter,

		Registry: "quay.io",
		Org:      "minio",

		csiProvisionerImage:      csiProvisionerImage,
		nodeDriverRegistrarImage: nodeDriverRegistrarImage,
		livenessProbeImage:       livenessProbeImage,
		csiResizerImage:          csiResizerImage,
	}

	if err := args.validate(); err != nil {
		return nil, err
	}
	return args, nil
}

func (args *Args) validate() error {
	if args.image == "" {
		return errors.New("image name must be provided")
	}

	if args.auditWriter == nil {
		return errors.New("audit writer must be provided")
	}

	return nil
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
	return path.Join(args.Registry, args.Org, args.nodeDriverRegistrarImage)
}

func (args *Args) getLivenessProbeImage() string {
	return path.Join(args.Registry, args.Org, args.livenessProbeImage)
}

func (args *Args) getCSIProvisionerImage() string {
	return path.Join(args.Registry, args.Org, args.csiProvisionerImage)
}

func (args *Args) getCSIResizerImage() string {
	return path.Join(args.Registry, args.Org, args.csiResizerImage)
}
