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
	"os"

	"github.com/fatih/color"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/installer"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var installCmd = &cobra.Command{
	Use:           "install",
	Short:         "Install " + consts.AppPrettyName + " in Kubernetes.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		return install(c.Context(), args)
	},
}

var (
	image                  = consts.AppName + ":" + Version
	registry               = "quay.io"
	org                    = "minio"
	nodeSelectorParameters = []string{}
	tolerationParameters   = []string{}
	seccompProfile         = ""
	apparmorProfile        = ""
	auditInstall           = "install"
	imagePullSecrets       = []string{}
)

func init() {
	if Version == "" {
		image = consts.AppName + ":0.0.0-dev"
	}
	installCmd.PersistentFlags().StringVarP(&image, "image", "i", image, consts.AppPrettyName+" image")
	installCmd.PersistentFlags().StringSliceVarP(&imagePullSecrets, "image-pull-secrets", "", imagePullSecrets, "Image pull secrets to be set in pod specs")
	installCmd.PersistentFlags().StringVarP(&registry, "registry", "r", registry, "Registry where "+consts.AppPrettyName+" images are available")
	installCmd.PersistentFlags().StringVarP(&org, "org", "g", org, "Organization name on the registry holds "+consts.AppPrettyName+" images")
	installCmd.PersistentFlags().StringSliceVarP(&nodeSelectorParameters, "node-selector", "n", nodeSelectorParameters, "Node selector parameters")
	installCmd.PersistentFlags().StringSliceVarP(&tolerationParameters, "tolerations", "t", tolerationParameters, "Tolerations parameters")
	installCmd.PersistentFlags().StringVarP(&seccompProfile, "seccomp-profile", "", seccompProfile, "Set Seccomp profile")
	installCmd.PersistentFlags().StringVarP(&apparmorProfile, "apparmor-profile", "", apparmorProfile, "Set Apparmor profile")
}

func install(ctx context.Context, args []string) (err error) {
	nodeSelector, err := parseNodeSelector(nodeSelectorParameters)
	if err != nil {
		return fmt.Errorf("%w; format of '--node-selector' flag value must be [<key>=<value>]", err)
	}
	tolerations, err := parseTolerations(tolerationParameters)
	if err != nil {
		return fmt.Errorf("%w; format of '--tolerations' flag value must be <key>[=value]:<NoSchedule|PreferNoSchedule|NoExecute>", err)
	}

	file, err := openAuditFile(auditInstall)
	if err != nil {
		klog.ErrorS(err, "unable to open audit file", "file", auditInstall)
		fmt.Fprintln(os.Stderr, color.HiYellowString("Skipping audit logging"))
	}

	defer func() {
		if file != nil {
			if err := file.Close(); err != nil {
				klog.ErrorS(err, "unable to close audit file")
			}
		}
	}()

	installConfig := &installer.Config{
		Identity:          identity,
		ContainerImage:    image,
		ContainerOrg:      org,
		ContainerRegistry: registry,
		NodeSelector:      nodeSelector,
		Tolerations:       tolerations,
		SeccompProfile:    seccompProfile,
		ApparmorProfile:   apparmorProfile,
		DryRun:            dryRun,
		AuditFile:         file,
		ImagePullSecrets:  imagePullSecrets,
	}

	if err = installer.Install(ctx, installConfig); err == nil && !dryRun {
		fmt.Println(color.HiWhiteString(consts.AppPrettyName), "is installed successfully")
	}

	return err
}
