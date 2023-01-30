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

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
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
	k8sVersion       = "1.25.0"
	kubeVersion      *version.Version
	legacyFlag       bool
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
   $ kubectl {PLUGIN_NAME} install --apparmor-profile apparmor.json`,
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
	Run: func(c *cobra.Command, args []string) {
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
	if err := validateOutputFormat(false); err != nil {
		return err
	}
	if dryRunPrinter != nil && k8sVersion != "" {
		var err error
		if kubeVersion, err = version.ParseSemantic(k8sVersion); err != nil {
			return fmt.Errorf("invalid kubernetes version %v; %v", k8sVersion, err)
		}
	}
	return nil
}

func getLegacyFlag(ctx context.Context) bool {
	if dryRunPrinter != nil {
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
	args.Quiet = quietFlag
	args.KubeVersion = kubeVersion
	args.Legacy = legacyFlag
	args.DryRunPrinter = dryRunPrinter

	var failed bool
	var installedComponents []installer.Component
	var wg sync.WaitGroup
	if dryRunPrinter == nil && !quietFlag {
		m := progressModel{
			model: progress.New(progress.WithGradient("#FFFFFF", "#FFFFFF")),
		}
		teaProgram := tea.NewProgram(m)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := teaProgram.Run(); err != nil {
				fmt.Println("error running program:", err)
				os.Exit(1)
			}
		}()
		wg.Add(1)
		var totalSteps, step, completedTasks int
		var currentPercent float64
		totalTasks := len(installer.Tasks)
		weightagePerTask := 1.0 / totalTasks
		progressCh := make(chan installer.Message)
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
	if err := installer.Install(ctx, args); err != nil && args.ProgressCh == nil {
		utils.Eprintf(quietFlag, true, "%v\n", err)
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
