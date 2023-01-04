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
	csiProvisionerImage      = "csi-provisioner:v3.3.0"
	nodeDriverRegistrarImage = "csi-node-driver-registrar:v2.6.0"
	livenessProbeImage       = "livenessprobe:v2.8.0"
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
