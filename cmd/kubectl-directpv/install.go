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
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/minio/directpv/pkg/admin"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/installer"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

const installAuditFile = "install.log"

var installCmd = &cobra.Command{
	Use:           "install",
	Short:         "Install " + consts.AppPrettyName + " in Kubernetes.",
	SilenceUsage:  true,
	SilenceErrors: true,
	Run: func(c *cobra.Command, args []string) {
		installMain(c.Context(), args)
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
	installCmd.PersistentFlags().StringVarP(&configDir, "config-dir", "", configDir, "Path to configuration directory")
}

func getCredential() (*admin.Credential, bool, error) {
	file, err := os.Open(getCredFile())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &admin.Credential{
				AccessKey: "directpvadmin",
				SecretKey: "directpvadmin",
			}, false, nil
		}

		return nil, false, err
	}
	defer file.Close()

	var cred admin.Credential
	if err := json.NewDecoder(file).Decode(&cred); err != nil {
		return nil, false, err
	}

	eprintf(color.HiYellowString("Using credential from "+getCredFile()), false)
	return &cred, true, nil
}

func installMain(ctx context.Context, args []string) {
	nodeSelector, err := parseNodeSelector(nodeSelectorParameters)
	if err != nil {
		eprintf(fmt.Sprintf("%v; format of '--node-selector' flag value must be [<key>=<value>]", err), true)
		os.Exit(-1)
	}
	tolerations, err := parseTolerations(tolerationParameters)
	if err != nil {
		eprintf(fmt.Sprintf("%v; format of '--tolerations' flag value must be <key>[=value]:<NoSchedule|PreferNoSchedule|NoExecute>", err), true)
		os.Exit(-1)
	}

	file, err := openAuditFile(installAuditFile)
	if err != nil {
		eprintf(fmt.Sprintf("unable to open audit file %v; %v", installAuditFile, err), true)
		fmt.Fprintln(os.Stderr, color.HiYellowString("Skipping audit logging"))
	}

	defer func() {
		if file != nil {
			if err := file.Close(); err != nil {
				klog.ErrorS(err, "unable to close audit file")
			}
		}
	}()

	cred, fromFile, err := getCredential()
	if err != nil {
		eprintf(fmt.Sprintf("unable to get credential; %v", err), true)
		os.Exit(1)
	}

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
		Credential:        cred,
	}

	if err = installer.Install(ctx, installConfig); err != nil {
		eprintf(fmt.Sprintf("unable to install %v; %v", consts.AppPrettyName, err), true)
		os.Exit(1)
	}

	if !dryRun {
		if !fromFile {
			data, err := json.Marshal(cred)
			if err == nil {
				err = os.WriteFile(getCredFile(), data, 0o644)
			}
			if err != nil {
				eprintf(fmt.Sprintf("unable to create credential file %v; %v", getCredFile(), err), true)
				eprintf(fmt.Sprintf("Below credential is set on server\nAccessKey: %v, SecretKey: %v", cred.AccessKey, cred.SecretKey), false)
			} else {
				eprintf(fmt.Sprintf("Credential is saved at %v", getCredFile()), false)
			}
		}

		fmt.Println(color.HiWhiteString(consts.AppPrettyName), "is installed successfully")
	}
}
