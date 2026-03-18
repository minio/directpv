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
	"io"

	"github.com/minio/directpv/pkg/admin/installer"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	legacyclient "github.com/minio/directpv/pkg/legacy/client"
	"github.com/minio/directpv/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	versionpkg "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/klog/v2"
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
	// PluginVersion denotes the plugin version; this will be set in node-server's annotations
	PluginVersion string
	// Quiet enables quiet mode
	Quiet bool
	// KubeVersion is required for declarative and dryrun manifests
	KubeVersion *versionpkg.Version
	// DryRun when set, runs in dryrun mode and generates the manifests
	DryRun bool
	// OutputFormat denotes the output format (yaml|json) for the manifests; to be used for DryRun
	OutputFormat string
	// Declarative when set, generates yaml manifests
	Declarative bool
	// Openshift when set, runs openshift specific installation
	Openshift bool
	// ProgressCh represents the progress channel
	ProgressCh chan<- installer.Message
	// AuditWriter denotes the writer passed to record the audit log
	AuditWriter io.Writer
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
func (client *Client) Install(ctx context.Context, args InstallArgs, installerTasks []installer.Task) error {
	var err error
	if err := args.Validate(); err != nil {
		return err
	}
	installerArgs := installer.NewArgs(args.Image)
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
	installerArgs.Legacy = client.isLegacyEnabled(ctx, args)
	installerArgs.PluginVersion = version
	if args.AuditWriter != nil {
		installerArgs.ObjectWriter = args.AuditWriter
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
		if installerArgs.KubeVersion == nil {
			// default higher version
			if installerArgs.KubeVersion, err = versionpkg.ParseSemantic("1.35.0"); err != nil {
				klog.Fatalf("this should not happen; %v", err)
			}
		}
	} else {
		major, minor, err := client.K8s().GetKubeVersion()
		if err != nil {
			return err
		}
		installerArgs.KubeVersion, err = versionpkg.ParseSemantic(fmt.Sprintf("%v.%v.0", major, minor))
		if err != nil {
			klog.Fatalf("this should not happen; %v", err)
		}
	}
	installerArgs.Declarative = args.Declarative
	installerArgs.Openshift = args.Openshift
	installerArgs.ProgressCh = args.ProgressCh

	return installer.Install(ctx, installerArgs, installerTasks)
}

func (client Client) isLegacyEnabled(ctx context.Context, args InstallArgs) bool {
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
	if result, ok := <-resultCh; ok {
		if result.Err == nil {
			return true
		}

		utils.Eprintf(args.Quiet, true, "unable to get volumes; %v", result.Err)
	}

	legacyClient, err := legacyclient.NewClient(client.K8s())
	if err != nil {
		utils.Eprintf(args.Quiet, true, "unable to create legacy client; %v", err)
		return false
	}

	if result, ok := <-legacyClient.ListVolumes(ctx); ok {
		if result.Err == nil {
			return true
		}

		utils.Eprintf(args.Quiet, true, "unable to get legacy volumes; %v", result.Err)
	}

	return false
}
