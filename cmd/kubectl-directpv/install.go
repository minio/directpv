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
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
	k8sVersion       = "1.35.0"
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
			eprintf(true, "%v\n", err)
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
		return fmt.Errorf("%w; format of '--node-selector' flag value must be [<key>=<value>]", err)
	}
	tolerations, err = k8s.ParseTolerations(tolerationArgs)
	if err != nil {
		return fmt.Errorf("%w; format of '--tolerations' flag value must be <key>[=value]:<NoSchedule|PreferNoSchedule|NoExecute>", err)
	}
	if err = validateOutputFormat(false); err != nil {
		return err
	}
	if dryRunPrinter != nil && k8sVersion != "" {
		if kubeVersion, err = version.ParseSemantic(k8sVersion); err != nil {
			return fmt.Errorf("invalid kubernetes version %v; %w", k8sVersion, err)
		}
	}
	return nil
}

func installMain(ctx context.Context) {
	pluginVersion := "dev"
	if Version != "" {
		pluginVersion = Version
	}
	dryRun := dryRunPrinter != nil
	var file *utils.SafeFile
	var err error
	if !dryRun && !declarativeFlag {
		auditFile := fmt.Sprintf("install.log.%v", time.Now().UTC().Format(time.RFC3339Nano))
		file, err = openAuditFile(auditFile)
		if err != nil {
			eprintf(true, "unable to open audit file %v; %v\n", auditFile, err)
			eprintf(false, "%v\n", color.HiYellowString("Skipping audit logging"))
		}
		defer func() {
			if file != nil {
				if err := file.Close(); err != nil {
					eprintf(true, "unable to close audit file; %v\n", err)
				}
			}
		}()
	}

	args := admin.InstallArgs{
		Image:            image,
		Registry:         registry,
		Org:              org,
		ImagePullSecrets: imagePullSecrets,
		NodeSelector:     nodeSelector,
		Tolerations:      tolerations,
		SeccompProfile:   seccompProfile,
		AppArmorProfile:  apparmorProfile,
		EnableLegacy:     legacyFlag,
		PluginVersion:    pluginVersion,
		Quiet:            quietFlag,
		KubeVersion:      kubeVersion,
		DryRun:           dryRunPrinter != nil,
		OutputFormat:     outputFormat,
		Declarative:      declarativeFlag,
		Openshift:        openshiftFlag,
	}
	if file != nil {
		args.AuditWriter = file
	}
	var failed bool
	var wg sync.WaitGroup
	var installedComponents []installer.Component
	installerTasks := installer.GetDefaultTasks(adminClient.Client, legacyClient)
	enableProgress := !dryRun && !declarativeFlag && !quietFlag
	if enableProgress {
		m := newProgressModel(true)
		teaProgram := tea.NewProgram(m)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := teaProgram.Run(); err != nil {
				fmt.Println("error running program:", err)
				return
			}
		}()
		var totalSteps, step, completedTasks int
		var currentPercent float64
		totalTasks := len(installerTasks)
		weightagePerTask := 1.0 / totalTasks
		progressCh := make(chan installer.Message)
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(args.ProgressCh)
			var perStepWeightage float64
			var done bool
			var message string
			for {
				var log string
				var installedComponent *installer.Component
				var err error
				select {
				case progressMessage, ok := <-progressCh:
					if !ok {
						teaProgram.Send(progressNotification{
							done: true,
						})
						return
					}
					switch progressMessage.Type() {
					case installer.StartMessageType:
						totalSteps = progressMessage.StartMessage()
						perStepWeightage = float64(weightagePerTask) / float64(totalSteps)
					case installer.ProgressMessageType:
						message, step, installedComponent = progressMessage.ProgressMessage()
						if step > totalSteps {
							step = totalSteps
						}
						if step > 0 {
							currentPercent += float64(step) * perStepWeightage
						}
					case installer.EndMessageType:
						installedComponent, err = progressMessage.EndMessage()
						if err == nil {
							completedTasks++
							currentPercent = float64(completedTasks) / float64(totalTasks)
						}
					case installer.LogMessageType:
						log = progressMessage.LogMessage()
					case installer.DoneMessageType:
						err = progressMessage.DoneMessage()
						message = ""
						done = true
					}
					if err != nil {
						failed = true
					}
					if installedComponent != nil {
						installedComponents = append(installedComponents, *installedComponent)
					}
					teaProgram.Send(progressNotification{
						message: message,
						log:     log,
						percent: currentPercent,
						done:    done,
						err:     err,
					})
					if done {
						return
					}
				case <-ctx.Done():
					fmt.Println("exiting installation; ", ctx.Err())
					os.Exit(1)
				}
			}
		}()
		args.ProgressCh = progressCh
	}

	if err := adminClient.Install(ctx, args, installerTasks); err != nil && args.ProgressCh == nil {
		eprintf(true, "%v\n", err)
		os.Exit(1)
	}
	if args.ProgressCh != nil {
		wg.Wait()
		if !failed {
			printTable(installedComponents)
			color.HiGreen("\nDirectPV installed successfully")
		}
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
