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

package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/minio/directpv/pkg/installer"
	"github.com/minio/directpv/pkg/utils"

	"k8s.io/klog/v2"
)

var installCmd = &cobra.Command{
	Use:           "install",
	Short:         utils.BinaryNameTransform("Install {{ . }} in k8s cluster"),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		return install(c.Context(), args)
	},
}

var (
	installCRD             = false
	admissionControl       = false
	image                  = "directpv:" + Version
	registry               = "quay.io"
	org                    = "minio"
	loopbackOnly           = false
	nodeSelectorParameters = []string{}
	tolerationParameters   = []string{}
	seccompProfile         = ""
	apparmorProfile        = ""
	dynamicDriveDiscovery  = false
	auditInstall           = "install"
	imagePullSecrets       = []string{}
)

func init() {
	installCmd.PersistentFlags().BoolVarP(&installCRD, "crd", "c", installCRD, "register crds along with installation")
	installCmd.PersistentFlags().StringVarP(&image, "image", "i", image, "DirectPV image")
	installCmd.PersistentFlags().StringSliceVarP(&imagePullSecrets, "image-pull-secrets", "", imagePullSecrets, "image pull secrets to be set in pod specs")
	installCmd.PersistentFlags().StringVarP(&registry, "registry", "r", registry, "registry where DirectPV images are available")
	installCmd.PersistentFlags().StringVarP(&org, "org", "g", org, "organization name where DirectPV images are available")
	installCmd.PersistentFlags().BoolVarP(&admissionControl, "admission-control", "", admissionControl, "turn on DirectPV admission controller")
	installCmd.PersistentFlags().MarkDeprecated("crd", "Will be removed in version 1.5 or greater")
	installCmd.PersistentFlags().StringSliceVarP(&nodeSelectorParameters, "node-selector", "n", nodeSelectorParameters, "node selector parameters")
	installCmd.PersistentFlags().StringSliceVarP(&tolerationParameters, "tolerations", "t", tolerationParameters, "tolerations parameters")
	installCmd.PersistentFlags().StringVarP(&seccompProfile, "seccomp-profile", "", seccompProfile, "set Seccomp profile")
	installCmd.PersistentFlags().StringVarP(&apparmorProfile, "apparmor-profile", "", apparmorProfile, "set Apparmor profile")

	installCmd.PersistentFlags().BoolVarP(&loopbackOnly, "loopback-only", "", loopbackOnly, "Uses 4 free loopback devices per node and treat them as DirectCSIDrive resources. This is recommended only for testing/development purposes")
	installCmd.PersistentFlags().MarkHidden("loopback-only")
	installCmd.PersistentFlags().BoolVarP(&dynamicDriveDiscovery, "enable-dynamic-discovery", "", dynamicDriveDiscovery, "Enable dynamic drive discovery (experimental)")
}

func install(ctx context.Context, args []string) (err error) {
	if err := validImage(image); err != nil {
		return fmt.Errorf("invalid argument. format of '--image' must be [image:tag] err=%v", err)
	}
	if err := validOrg(org); err != nil {
		return fmt.Errorf("invalid org. format of '--org' must be [a-zA-Z][a-zA-Z0-9-.]* err=%v", err)
	}
	if err := validRegistry(registry); err != nil {
		return fmt.Errorf("invalid registry. format of '--registry' must be [host:port?]")
	}
	nodeSelector, err := parseNodeSelector(nodeSelectorParameters)
	if err != nil {
		return fmt.Errorf("invalid node selector. format of '--node-selector' must be [<key>=<value>]")
	}
	tolerations, err := parseTolerations(tolerationParameters)
	if err != nil {
		return fmt.Errorf("invalid tolerations. format of '--tolerations' must be <key>[=value]:<NoSchedule|PreferNoSchedule|NoExecute>")
	}

	if !dynamicDriveDiscovery {
		klog.Infof("Enable dynamic drive change management using " + utils.Bold("--enable-dynamic-discovery") + " flag")
	}
	klog.Warning("dynamic-discovery feature is strictly experimental and NOT production ready yet")

	file, err := utils.OpenAuditFile(auditInstall)
	if err != nil {
		klog.Errorf("error in audit logging: %w", err)
	}

	defer func() {
		if file != nil {
			if err := file.Close(); err != nil {
				klog.Errorf("unable to close audit file : %w", err)
			}
		}
	}()

	installConfig := &installer.Config{
		Identity:                   identity,
		DirectCSIContainerImage:    image,
		DirectCSIContainerOrg:      org,
		DirectCSIContainerRegistry: registry,
		AdmissionControl:           admissionControl,
		LoopbackMode:               loopbackOnly,
		NodeSelector:               nodeSelector,
		Tolerations:                tolerations,
		SeccompProfile:             seccompProfile,
		ApparmorProfile:            apparmorProfile,
		DynamicDriveDiscovery:      true,
		DryRun:                     dryRun,
		AuditFile:                  file,
		ImagePullSecrets:           imagePullSecrets,
	}

	return installer.Install(ctx, installConfig)
}
