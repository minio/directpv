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
	"strings"
	"time"

	"github.com/fatih/color"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/installer"
	legacyclient "github.com/minio/directpv/pkg/legacy/client"
	"github.com/minio/directpv/pkg/utils"
	"github.com/minio/directpv/pkg/volume"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/version"
)

var (
	image            = consts.AppName + ":" + Version
	registry         = "quay.io"
	org              = "minio"
	nodeSelectorArgs = []string{}
	tolerationArgs   = []string{}
	seccompProfile   = ""
	apparmorProfile  = ""
	imagePullSecrets = []string{}
	nodeSelector     map[string]string
	tolerations      []corev1.Toleration
	k8sVersion       string
	kubeVersion      *version.Version
	legacyFlag       bool
)

var installCmd = &cobra.Command{
	Use:           "install",
	Short:         "Install " + consts.AppPrettyName + " in Kubernetes",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		disableInit = dryRunFlag
		if parent := cmd.Parent(); parent != nil {
			return parent.PersistentPreRunE(parent, args)
		}
		return nil
	},
	Run: func(c *cobra.Command, args []string) {
		if err := validateInstallCmd(); err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", err)
			os.Exit(-1)
		}

		installMain(c.Context())
	},
}

func init() {
	if Version == "" {
		image = consts.AppName + ":0.0.0-dev"
	}

	installCmd.Flags().SortFlags = false
	installCmd.InheritedFlags().SortFlags = false
	installCmd.LocalFlags().SortFlags = false
	installCmd.LocalNonPersistentFlags().SortFlags = false
	installCmd.NonInheritedFlags().SortFlags = false
	installCmd.PersistentFlags().SortFlags = false

	installCmd.PersistentFlags().StringVar(&image, "image", image, consts.AppPrettyName+" image")
	installCmd.PersistentFlags().StringSliceVar(&imagePullSecrets, "image-pull-secrets", imagePullSecrets, "Image pull secrets to be set in pod specs")
	installCmd.PersistentFlags().StringVar(&registry, "registry", registry, "Registry where "+consts.AppPrettyName+" images are available")
	installCmd.PersistentFlags().StringVar(&org, "org", org, "Organization name on the registry holds "+consts.AppPrettyName+" images")
	installCmd.PersistentFlags().StringSliceVar(&nodeSelectorArgs, "node-selector", nodeSelectorArgs, "Node selector parameters")
	installCmd.PersistentFlags().StringSliceVar(&tolerationArgs, "tolerations", tolerationArgs, "Tolerations parameters")
	installCmd.PersistentFlags().StringVar(&seccompProfile, "seccomp-profile", seccompProfile, "Set Seccomp profile")
	installCmd.PersistentFlags().StringVar(&apparmorProfile, "apparmor-profile", apparmorProfile, "Set Apparmor profile")
	installCmd.PersistentFlags().StringVar(&configDir, "config-dir", configDir, "Path to configuration directory")
	installCmd.PersistentFlags().StringVar(&k8sVersion, "kube-version", k8sVersion, "If present, use this as kubernetes version for dry-run")
	installCmd.PersistentFlags().BoolVar(&legacyFlag, "legacy", legacyFlag, "If present, include legacy services for dry-run")
	addDryRunFlag(installCmd)
}

func validateNodeSelectorArgs() error {
	nodeSelector = map[string]string{}
	for _, value := range nodeSelectorArgs {
		tokens := strings.Split(value, "=")
		if len(tokens) != 2 {
			return fmt.Errorf("invalid node selector value %v", value)
		}
		if tokens[0] == "" {
			return fmt.Errorf("invalid key in node selector value %v", value)
		}
		nodeSelector[tokens[0]] = tokens[1]
	}
	return nil
}

func validateTolerationsArgs() error {
	for _, value := range tolerationArgs {
		var k, v, e string
		tokens := strings.SplitN(value, "=", 2)
		switch len(tokens) {
		case 1:
			k = tokens[0]
			tokens = strings.Split(k, ":")
			switch len(tokens) {
			case 1:
			case 2:
				k, e = tokens[0], tokens[1]
			default:
				if len(tokens) != 2 {
					return fmt.Errorf("invalid toleration %v", value)
				}
			}
		case 2:
			k, v = tokens[0], tokens[1]
		default:
			if len(tokens) != 2 {
				return fmt.Errorf("invalid toleration %v", value)
			}
		}
		if k == "" {
			return fmt.Errorf("invalid key in toleration %v", value)
		}
		if v != "" {
			if tokens = strings.Split(v, ":"); len(tokens) != 2 {
				return fmt.Errorf("invalid value in toleration %v", value)
			}
			v, e = tokens[0], tokens[1]
		}
		effect := corev1.TaintEffect(e)
		switch effect {
		case corev1.TaintEffectNoSchedule, corev1.TaintEffectPreferNoSchedule, corev1.TaintEffectNoExecute:
		default:
			return fmt.Errorf("invalid toleration effect in toleration %v", value)
		}
		operator := corev1.TolerationOpExists
		if v != "" {
			operator = corev1.TolerationOpEqual
		}
		tolerations = append(tolerations, corev1.Toleration{
			Key:      k,
			Operator: operator,
			Value:    v,
			Effect:   effect,
		})
	}

	return nil
}

func validateInstallCmd() error {
	if err := validateNodeSelectorArgs(); err != nil {
		return fmt.Errorf("%v; format of '--node-selector' flag value must be [<key>=<value>]", err)
	}

	if err := validateTolerationsArgs(); err != nil {
		return fmt.Errorf("%v; format of '--tolerations' flag value must be <key>[=value]:<NoSchedule|PreferNoSchedule|NoExecute>", err)
	}

	if dryRunFlag && k8sVersion != "" {
		var err error
		if kubeVersion, err = version.ParseSemantic(k8sVersion); err != nil {
			return fmt.Errorf("invalid kubernetes version %v; %v", k8sVersion, err)
		}
	}

	return nil
}

func getLegacyFlag(ctx context.Context) bool {
	if dryRunFlag {
		return legacyFlag
	}

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh := volume.NewLister().
		LabelSelector(
			map[directpvtypes.LabelKey]directpvtypes.LabelValue{
				directpvtypes.MigratedLabelKey: "true",
			},
		).
		IgnoreNotFound(true).
		List(ctx)
	for result := range resultCh {
		if result.Err != nil {
			utils.Eprintf(quietFlag, true, "unable to get volumes; %v", result.Err)
			break
		} else {
			return true
		}
	}

	legacyclient.Init()

	for result := range legacyclient.ListVolumes(ctx) {
		if result.Err != nil {
			utils.Eprintf(quietFlag, true, "unable to get legacy volumes; %v", result.Err)
			break
		} else {
			return true
		}
	}

	return false
}

func installMain(ctx context.Context) {
	legacyFlag = getLegacyFlag(ctx)

	auditFile := fmt.Sprintf("install.log.%v", time.Now().UTC().Format(time.RFC3339Nano))
	file, err := openAuditFile(auditFile)
	if err != nil {
		utils.Eprintf(quietFlag, true, "unable to open audit file %v; %v\n", auditFile, err)
		utils.Eprintf(false, false, "%v\n", color.HiYellowString("Skipping audit logging"))
	}

	defer func() {
		if file != nil {
			if err := file.Close(); err != nil {
				utils.Eprintf(quietFlag, true, "unable to close audit file; %v\n", err)
			}
		}
	}()

	args, err := installer.NewArgs(image, file)
	if err != nil {
		utils.Eprintf(quietFlag, true, "%v\n", err)
		os.Exit(1)
	}

	args.Registry = registry
	args.Org = org
	args.ImagePullSecrets = imagePullSecrets
	args.NodeSelector = nodeSelector
	args.Tolerations = tolerations
	args.SeccompProfile = seccompProfile
	args.AppArmorProfile = apparmorProfile
	args.DryRun = dryRunFlag
	args.Quiet = quietFlag
	args.KubeVersion = kubeVersion
	args.Legacy = legacyFlag

	if err = installer.Install(ctx, args); err != nil {
		utils.Eprintf(quietFlag, true, "%v\n", err)
		os.Exit(1)
	}

	if !dryRunFlag && !quietFlag {
		fmt.Println("Done")
	}
}
