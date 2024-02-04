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

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/minio/directpv/pkg/admin"
	"github.com/minio/directpv/pkg/admin/installer"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/utils"
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
	k8sVersion       = "1.29.0"
	kubeVersion      *version.Version
	legacyFlag       bool
	declarativeFlag  bool
	openshiftFlag    bool
)

var installCmd = &cobra.Command{
	Use:           "install",
	Short:         "Install " + consts.AppPrettyName + " in Kubernetes",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`1. Install DirectPV
   $ kubectl {PLUGIN_NAME} install

2. Pull images from private registry (eg, private-registry.io/org-name) for DirectPV installation
   $ kubectl {PLUGIN_NAME} install --registry private-registry.io --org org-name

3. Specify '--node-selector' to deploy DirectPV daemonset pods only on selective nodes
   $ kubectl {PLUGIN_NAME} install --node-selector node-label-key=node-label-value

4. Specify '--tolerations' to tolerate and deploy DirectPV daemonset pods on tainted nodes (Example: key=value:NoSchedule)
   $ kubectl {PLUGIN_NAME} install --tolerations key=value:NoSchedule

5. Generate DirectPV installation manifest in YAML
   $ kubectl {PLUGIN_NAME} install -o yaml > directpv-install.yaml

6. Install DirectPV with apparmor profile
   $ kubectl {PLUGIN_NAME} install --apparmor-profile directpv

7. Install DirectPV with seccomp profile
   $ kubectl {PLUGIN_NAME} install --seccomp-profile profiles/seccomp.json`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := validateInstallCmd(); err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", err)
			os.Exit(-1)
		}
		disableInit = dryRunPrinter != nil
		if parent := cmd.Parent(); parent != nil {
			return parent.PersistentPreRunE(parent, args)
		}
		return nil
	},
	Run: func(c *cobra.Command, _ []string) {
		installMain(c.Context())
	},
}

func init() {
	if Version == "" {
		image = consts.AppName + ":0.0.0-dev"
	}

	setFlagOpts(installCmd)

	installCmd.PersistentFlags().StringSliceVar(&nodeSelectorArgs, "node-selector", nodeSelectorArgs, "Select the storage nodes using labels (KEY=VALUE,..)")
	installCmd.PersistentFlags().StringSliceVar(&tolerationArgs, "tolerations", tolerationArgs, "Set toleration labels on the storage nodes (KEY[=VALUE]:EFFECT,..)")
	installCmd.PersistentFlags().StringVar(&registry, "registry", registry, "Name of container registry")
	installCmd.PersistentFlags().StringVar(&org, "org", org, "Organization name in the registry")
	installCmd.PersistentFlags().StringVar(&image, "image", image, "Name of the "+consts.AppPrettyName+" image")
	installCmd.PersistentFlags().StringSliceVar(&imagePullSecrets, "image-pull-secrets", imagePullSecrets, "Image pull secrets for "+consts.AppPrettyName+" images (SECRET1,..)")
	installCmd.PersistentFlags().StringVar(&apparmorProfile, "apparmor-profile", apparmorProfile, "Set path to Apparmor profile")
	installCmd.PersistentFlags().StringVar(&seccompProfile, "seccomp-profile", seccompProfile, "Set path to Seccomp profile")
	addOutputFormatFlag(installCmd, "Generate installation manifest. One of: yaml|json")
	installCmd.PersistentFlags().StringVar(&k8sVersion, "kube-version", k8sVersion, "Select the kubernetes version for manifest generation")
	installCmd.PersistentFlags().BoolVar(&legacyFlag, "legacy", legacyFlag, "Enable legacy mode (Used with '-o')")
	installCmd.PersistentFlags().BoolVar(&declarativeFlag, "declarative", declarativeFlag, "Output YAML for declarative installation")
	installCmd.PersistentFlags().MarkHidden("declarative")
	installCmd.PersistentFlags().BoolVar(&openshiftFlag, "openshift", openshiftFlag, "Use OpenShift specific installation")
}

func validateInstallCmd() (err error) {
	nodeSelector, err = k8s.ParseNodeSelector(nodeSelectorArgs)
	if err != nil {
		return fmt.Errorf("%v; format of '--node-selector' flag value must be [<key>=<value>]", err)
	}
	tolerations, err = k8s.ParseTolerations(tolerationArgs)
	if err != nil {
		return fmt.Errorf("%v; format of '--tolerations' flag value must be <key>[=value]:<NoSchedule|PreferNoSchedule|NoExecute>", err)
	}
	if err = validateOutputFormat(false); err != nil {
		return err
	}
	if dryRunPrinter != nil && k8sVersion != "" {
		if kubeVersion, err = version.ParseSemantic(k8sVersion); err != nil {
			return fmt.Errorf("invalid kubernetes version %v; %v", k8sVersion, err)
		}
	}
	return nil
}

func installMain(ctx context.Context) {
	pluginVersion := "dev"
	if Version != "" {
		pluginVersion = Version
	}
	enableProgress := dryRunPrinter == nil && !declarativeFlag && !quietFlag
	installedComponents, err := adminClient.Install(ctx, admin.InstallArgs{
		Image:            image,
		Registry:         registry,
		Org:              org,
		ImagePullSecrets: imagePullSecrets,
		NodeSelector:     nodeSelector,
		Tolerations:      tolerations,
		SeccompProfile:   seccompProfile,
		AppArmorProfile:  apparmorProfile,
		EnableLegacy:     legacyFlag,
		EnableAudit:      dryRunPrinter == nil && !declarativeFlag,
		PluginVersion:    pluginVersion,
		Quiet:            quietFlag,
		KubeVersion:      kubeVersion,
		DryRun:           dryRunPrinter != nil,
		OutputFormat:     outputFormat,
		Declarative:      declarativeFlag,
		Openshift:        openshiftFlag,
		PrintProgress:    enableProgress,
	})
	if err != nil {
		if !enableProgress {
			utils.Eprintf(quietFlag, true, "%v\n", err)
		}
		os.Exit(1)
	}
	if enableProgress {
		printTable(installedComponents)
		color.HiGreen("\nDirectPV installed successfully")
	}
}

func printTable(components []installer.Component) {
	writer := newTableWriter(
		table.Row{
			"NAME",
			"KIND",
		},
		nil,
		false,
	)

	for _, component := range components {
		row := []interface{}{
			color.HiGreenString(component.Name),
			component.Kind,
		}
		writer.AppendRow(row)
	}
	writer.Render()
}
