// This file is part of MinIO DirectPV
// Copyright (c) 2023 MinIO, Inc.
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

package admin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/installer"
	legacyclient "github.com/minio/directpv/pkg/legacy/client"
	"github.com/minio/directpv/pkg/utils"
	"github.com/mitchellh/go-homedir"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/version"
)

// ErrInstallationIncomplete denotes that the installation couldn't complete
var ErrInstallationIncomplete = errors.New("unable to complete the installation")

// InstallArgs represents the arguments required for installation
type InstallArgs struct {
	// Image of the DirectPV
	Image string
	// Registry denotes the private registry
	Registry string
	// Org denotes the organization name
	Org string
	// ImagePullSecrets for the images from priv registries
	ImagePullSecrets []string
	// NodeSelector denotes the nodeSelector to be set for the node-server
	NodeSelector map[string]string
	// Tolerations denotes the tolerations to be set for the node-server
	Tolerations []corev1.Toleration
	// SeccompProfile denotes the seccomp profile name to be set on the node-server
	SeccompProfile string
	// AppArmorProfile denotes the apparmor profile name to be set on the node-server
	AppArmorProfile string
	// EnableLegacy to run in legacy mode
	EnableLegacy bool
	// EnableAudit will create an audit file and writes to it
	EnableAudit bool
	// PluginVersion denotes the plugin version; this will be set in node-server's annotations
	PluginVersion string
	// Quiet enables quiet mode
	Quiet bool
	// KubeVersion is required for declarative and dryrun manifests
	KubeVersion *version.Version
	// DryRun when set, runs in dryrun mode and generates the manifests
	DryRun bool
	// OutputFormat denotes the output format (yaml|json) for the manifests; to be used for DryRun
	OutputFormat string
	// Declarative when set, generates yaml manifests
	Declarative bool
	// Openshift when set, runs openshift specific installation
	Openshift bool
	// PrintProgress when set, displays the progress in the UI
	PrintProgress bool
}

// Validate - validates the args
func (args *InstallArgs) Validate() error {
	if args.DryRun || args.Declarative {
		switch args.OutputFormat {
		case "yaml", "json":
		case "":
			args.OutputFormat = "yaml"
		}
	}
	return nil
}

// Install - installs directpv with the provided arguments
func Install(ctx context.Context, args InstallArgs) ([]installer.Component, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}
	var file *utils.SafeFile
	var err error
	if !args.DryRun && !args.Declarative {
		auditFile := fmt.Sprintf("install.log.%v", time.Now().UTC().Format(time.RFC3339Nano))
		file, err = openAuditFile(auditFile)
		if err != nil {
			utils.Eprintf(args.Quiet, true, "unable to open audit file %v; %v\n", auditFile, err)
			utils.Eprintf(false, false, "%v\n", color.HiYellowString("Skipping audit logging"))
		}

		defer func() {
			if file != nil {
				if err := file.Close(); err != nil {
					utils.Eprintf(args.Quiet, true, "unable to close audit file; %v\n", err)
				}
			}
		}()
	}

	installerArgs := installer.NewArgs(args.Image)
	if err != nil {
		return nil, err
	}

	var version string
	if args.PluginVersion != "" {
		version = args.PluginVersion
	}

	installerArgs.Registry = args.Registry
	installerArgs.Org = args.Org
	installerArgs.ImagePullSecrets = args.ImagePullSecrets
	installerArgs.NodeSelector = args.NodeSelector
	installerArgs.Tolerations = args.Tolerations
	installerArgs.SeccompProfile = args.SeccompProfile
	installerArgs.AppArmorProfile = args.AppArmorProfile
	installerArgs.Quiet = args.Quiet
	installerArgs.KubeVersion = args.KubeVersion
	installerArgs.Legacy = isLegacyEnabled(ctx, args)
	installerArgs.PluginVersion = version
	if file != nil {
		installerArgs.ObjectWriter = file
	}
	if args.DryRun {
		installerArgs.DryRun = true
		if args.OutputFormat == "yaml" {
			installerArgs.ObjectMarshaler = func(obj runtime.Object) ([]byte, error) {
				return utils.ToYAML(obj)
			}
		} else {
			installerArgs.ObjectMarshaler = func(obj runtime.Object) ([]byte, error) {
				return utils.ToJSON(obj)
			}
		}
	}
	installerArgs.Declarative = args.Declarative
	installerArgs.Openshift = args.Openshift

	var failed bool
	var installedComponents []installer.Component
	var wg sync.WaitGroup
	if args.PrintProgress {
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
		wg.Add(1)
		var totalSteps, step, completedTasks int
		var currentPercent float64
		totalTasks := len(installer.Tasks)
		weightagePerTask := 1.0 / totalTasks
		progressCh := make(chan installer.Message)
		go func() {
			defer wg.Done()
			defer close(installerArgs.ProgressCh)
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
		installerArgs.ProgressCh = progressCh
	}
	if err := installer.Install(ctx, installerArgs); err != nil && installerArgs.ProgressCh == nil {
		return installedComponents, err
	}
	if installerArgs.ProgressCh != nil {
		wg.Wait()
	}
	if failed {
		return installedComponents, ErrInstallationIncomplete
	}
	return installedComponents, nil
}

func isLegacyEnabled(ctx context.Context, args InstallArgs) bool {
	if args.DryRun {
		return args.EnableLegacy
	}

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh := client.NewVolumeLister().
		LabelSelector(
			map[directpvtypes.LabelKey]directpvtypes.LabelValue{
				directpvtypes.MigratedLabelKey: "true",
			},
		).
		IgnoreNotFound(true).
		List(ctx)
	for result := range resultCh {
		if result.Err != nil {
			utils.Eprintf(args.Quiet, true, "unable to get volumes; %v", result.Err)
			break
		}

		return true
	}

	legacyclient.Init()

	for result := range legacyclient.ListVolumes(ctx) {
		if result.Err != nil {
			utils.Eprintf(args.Quiet, true, "unable to get legacy volumes; %v", result.Err)
			break
		}

		return true
	}

	return false
}

func openAuditFile(auditFile string) (*utils.SafeFile, error) {
	defaultAuditDir, err := getDefaultAuditDir()
	if err != nil {
		return nil, fmt.Errorf("unable to get default audit directory; %w", err)
	}
	if err := os.MkdirAll(defaultAuditDir, 0o700); err != nil {
		return nil, fmt.Errorf("unable to create default audit directory; %w", err)
	}
	return utils.NewSafeFile(path.Join(defaultAuditDir, auditFile))
}

func getDefaultAuditDir() (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return path.Join(homeDir, "."+consts.AppName, "audit"), nil
}
