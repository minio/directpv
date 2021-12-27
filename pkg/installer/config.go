// This file is part of MinIO Direct CSI
// Copyright (c) 2021 MinIO, Inc.
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
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/minio/directpv/pkg/utils"

	corev1 "k8s.io/api/core/v1"
)

// CSI provisioner images
const (
	// quay.io/minio/csi-provisioner:v2.2.0-go1.17
	CSIImageCSIProvisioner = "csi-provisioner@sha256:d4f94539565cf62aea57062b6a42c5156337003133fd3f51b93df9a789e69840"

	// quay.io/minio/csi-node-driver-registrar:v2.2.0-go1.17
	CSIImageNodeDriverRegistrar = "csi-node-driver-registrar@sha256:843fb23b1a3fa1de986378b0b8c08c35f8e62499d386de8ec57801fd029afe6d"

	// quay.io/minio/livenessprobe:v2.2.0-go1.17
	CSIImageLivenessProbe = "livenessprobe@sha256:928a80be4d363e0e438ff28dcdb00d8d674d3059c6149a8cda64ce6016a9a3f8"
)

func defaultIfZeroString(left, right string) string {
	if left != "" {
		return left
	}
	return right
}

type Config struct {
	Identity string

	// DirectCSIContainerImage properties
	DirectCSIContainerImage    string
	DirectCSIContainerOrg      string
	DirectCSIContainerRegistry string

	// CSIImage properties
	CSIProvisionerImage      string
	NodeDriverRegistrarImage string
	LivenessProbeImage       string

	// Admission controller
	AdmissionControl bool

	// Mode switches
	LoopbackMode bool

	// Selectors and tolerations
	NodeSelector map[string]string
	Tolerations  []corev1.Toleration

	// Security profiles
	SeccompProfile  string
	ApparmorProfile string

	DynamicDriveDiscovery bool

	// dry-run properties
	DryRun bool

	// CRD uninstallation
	ForceRemove  bool
	UninstallCRD bool

	// Audit
	AuditFile *utils.SafeFile

	// Image pull secrets
	ImagePullSecrets []string

	// internal
	conversionWebhookCaBundle []byte
	validationWebhookCaBundle []byte
}

type installer interface {
	Install(context.Context) error
	Uninstall(context.Context) error
}

func (c *Config) validate() error {
	if c.Identity == "" {
		return errors.New("identity cannot be empty")
	}
	return nil
}

func (i *Config) namespace() string {
	return utils.SanitizeKubeResourceName(i.Identity)
}

func (i *Config) serviceName() string {
	return utils.SanitizeKubeResourceName(i.Identity)
}

func (i *Config) identity() string {
	return utils.SanitizeKubeResourceName(i.Identity)
}

func (c *Config) conversionHealthzURL() string {
	return getConversionHealthzURL(c.identity())
}

func (i *Config) getCSIProvisionerImage() string {
	return defaultIfZeroString(i.CSIProvisionerImage, CSIImageCSIProvisioner)
}

func (i *Config) getNodeDriverRegistrarImage() string {
	return defaultIfZeroString(i.NodeDriverRegistrarImage, CSIImageNodeDriverRegistrar)
}

func (i *Config) getLivenessProbeImage() string {
	return defaultIfZeroString(i.LivenessProbeImage, CSIImageLivenessProbe)
}

func (i *Config) conversionWebhookDNSName() string {
	return strings.Join([]string{i.identity(), i.namespace(), "svc"}, ".") // "direct-csi-min-io.direct-csi-min-io.svc"
}

func (c *Config) csiDriverName() string {
	return c.identity()
}

func (c *Config) daemonsetName() string {
	return c.identity()
}

func (c *Config) deploymentName() string {
	return c.identity()
}

func (i *Config) getPSPName() string {
	return i.identity()
}

func (i *Config) getPSPClusterRoleBindingName() string {
	return utils.SanitizeKubeResourceName("psp-" + i.identity())
}

func (i *Config) serviceAccountName() string {
	return i.identity()
}

func (i *Config) clusterRoleName() string {
	return i.identity()
}

func (i *Config) roleBindingName() string {
	return i.identity()
}

func (c *Config) storageClassNameDirectCSI() string {
	return c.identity()
}

func (c *Config) storageClassNameDirectPV() string {
	return "directpv-min-io"
}

func (c *Config) driverIdentity() string {
	return c.identity()
}

func (c *Config) provisionerName() string {
	return c.identity()
}

func (c *Config) getImagePullSecrets() []corev1.LocalObjectReference {
	var localObjectReferences []corev1.LocalObjectReference
	for _, imagePullReferentName := range c.ImagePullSecrets {
		localObjectReferences = append(localObjectReferences, corev1.LocalObjectReference{
			Name: imagePullReferentName,
		})
	}
	return localObjectReferences
}

func (i *Config) postProc(obj interface{}) error {
	if i.DryRun {
		yamlString, err := utils.ToYAML(obj)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n---\n", yamlString)
	}
	if i.AuditFile != nil {
		if err := utils.WriteObject(i.AuditFile, obj); err != nil {
			return err
		}
	}
	return nil
}
