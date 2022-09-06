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

	"github.com/fatih/color"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/installer"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var installCmd = &cobra.Command{
	Use:           "install",
	Short:         "Install " + consts.AppPrettyName + " in kubernetes cluster",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		return install(c.Context(), args)
	},
}

var (
	admissionControl       = false
	image                  = consts.AppName + ":" + Version
	registry               = "quay.io"
	org                    = "minio"
	nodeSelectorParameters = []string{}
	tolerationParameters   = []string{}
	seccompProfile         = ""
	apparmorProfile        = ""
	auditInstall           = "install"
	imagePullSecrets       = []string{}
	disableUDevListener    = false
)

func init() {
	installCmd.PersistentFlags().StringVarP(&image, "image", "i", image, consts.AppPrettyName+" image")
	installCmd.PersistentFlags().StringSliceVarP(&imagePullSecrets, "image-pull-secrets", "", imagePullSecrets, "image pull secrets to be set in pod specs")
	installCmd.PersistentFlags().StringVarP(&registry, "registry", "r", registry, "registry where "+consts.AppPrettyName+" images are available")
	installCmd.PersistentFlags().StringVarP(&org, "org", "g", org, "organization name where "+consts.AppPrettyName+" images are available")
	installCmd.PersistentFlags().BoolVarP(&admissionControl, "admission-control", "", admissionControl, "turn on "+consts.AppPrettyName+" admission controller")
	installCmd.PersistentFlags().StringSliceVarP(&nodeSelectorParameters, "node-selector", "n", nodeSelectorParameters, "node selector parameters")
	installCmd.PersistentFlags().StringSliceVarP(&tolerationParameters, "tolerations", "t", tolerationParameters, "tolerations parameters")
	installCmd.PersistentFlags().StringVarP(&seccompProfile, "seccomp-profile", "", seccompProfile, "set Seccomp profile")
	installCmd.PersistentFlags().StringVarP(&apparmorProfile, "apparmor-profile", "", apparmorProfile, "set Apparmor profile")
	installCmd.PersistentFlags().BoolVarP(&disableUDevListener, "disable-udev-listener", "", disableUDevListener, "disable uevent listener and rely on 30secs internal drive-sync mechanism")
}

func install(ctx context.Context, args []string) (err error) {
	// if err := validImage(image); err != nil {
	// 	return fmt.Errorf("invalid argument. format of '--image' must be [image:tag] err=%v", err)
	// }
	// if err := validOrg(org); err != nil {
	// 	return fmt.Errorf("invalid org. format of '--org' must be [a-zA-Z][a-zA-Z0-9-.]* err=%v", err)
	// }
	// if err := validRegistry(registry); err != nil {
	// 	return fmt.Errorf("invalid registry. format of '--registry' must be [host:port?]")
	// }
	nodeSelector, err := parseNodeSelector(nodeSelectorParameters)
	if err != nil {
		return fmt.Errorf("invalid node selector. format of '--node-selector' must be [<key>=<value>]")
	}
	tolerations, err := parseTolerations(tolerationParameters)
	if err != nil {
		return fmt.Errorf("invalid tolerations. format of '--tolerations' must be <key>[=value]:<NoSchedule|PreferNoSchedule|NoExecute>")
	}

	file, err := OpenAuditFile(auditInstall)
	if err != nil {
		klog.ErrorS(err, "unable to open audit file", "file", auditInstall)
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
		AdmissionControl:  admissionControl,
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
