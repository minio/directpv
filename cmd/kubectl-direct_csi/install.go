// This file is part of MinIO Direct CSI
// Copyright (c) 2021 MinIO, Inc.
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
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/minio/direct-csi/pkg/installer"
	"github.com/minio/direct-csi/pkg/utils"

	"k8s.io/klog/v2"
)

var installCmd = &cobra.Command{
	Use:           "install",
	Short:         "Install direct-csi in k8s cluster",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		return install(c.Context(), args)
	},
}

var (
	installCRD         = false
	overwriteCRD       = false
	admissionControl   = false
	image              = "direct-csi:" + Version
	registry           = "quay.io"
	org                = "minio"
	loopBackOnly       = false
	nodeSelectorValues = []string{}
	tolerationValues   = []string{}
	seccompProfile     = ""
	apparmorProfile    = ""
)

func init() {
	installCmd.PersistentFlags().BoolVarP(&installCRD, "crd", "c", installCRD, "register crds along with installation")
	installCmd.PersistentFlags().BoolVarP(&overwriteCRD, "force", "f", overwriteCRD, "delete and recreate CRDs")
	installCmd.PersistentFlags().StringVarP(&image, "image", "i", image, "direct-csi image")
	installCmd.PersistentFlags().StringVarP(&registry, "registry", "r", registry, "registry where direct-csi images are available")
	installCmd.PersistentFlags().StringVarP(&org, "org", "g", org, "organization name where direct-csi images are available")
	installCmd.PersistentFlags().BoolVarP(&admissionControl, "admission-control", "", admissionControl, "turn on direct-csi admission controller")
	installCmd.PersistentFlags().MarkDeprecated("crd", "Will be removed in version 1.5 or greater")
	installCmd.PersistentFlags().StringSliceVarP(&nodeSelectorValues, "node-selector", "n", nodeSelectorValues, "node selector parameters")
	installCmd.PersistentFlags().StringSliceVarP(&tolerationValues, "tolerations", "t", tolerationValues, "tolerations parameters")
	installCmd.PersistentFlags().StringVarP(&seccompProfile, "seccomp-profile", "", seccompProfile, "set Seccomp profile")
	installCmd.PersistentFlags().StringVarP(&apparmorProfile, "apparmor-profile", "", apparmorProfile, "set Apparmor profile")

	installCmd.PersistentFlags().BoolVarP(&loopBackOnly, "loopback-only", "", loopBackOnly, "Uses 4 free loopback devices per node and treat them as DirectCSIDrive resources. This is recommended only for testing/development purposes")
	installCmd.PersistentFlags().MarkHidden("loopback-only")
}

func install(ctx context.Context, args []string) error {
	if err := validImage(image); err != nil {
		return fmt.Errorf("invalid argument. format of '--image' must be [image:tag] err=%v", err)
	}
	if err := validOrg(org); err != nil {
		return fmt.Errorf("invalid org. format of '--org' must be [a-zA-Z][a-zA-Z0-9-.]* err=%v", err)
	}
	if err := validRegistry(registry); err != nil {
		return fmt.Errorf("invalid registry. format of '--registry' must be [host:port?]")
	}
	nodeSelector, err := parseNodeSelector(nodeSelectorValues)
	if err != nil {
		return fmt.Errorf("invalid node selector. format of '--node-selector' must be [<key>=<value>]")
	}
	tolerations, err := parseTolerations(tolerationValues)
	if err != nil {
		return fmt.Errorf("invalid tolerations. format of '--tolerations' must be <key>[=value]:<NoSchedule|PreferNoSchedule|NoExecute>")
	}

	if err := installer.CreateNamespace(ctx, identity, dryRun); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}
	if !dryRun {
		klog.Infof("'%s' namespace created", utils.Bold(identity))
	}

	if err := installer.CreatePodSecurityPolicy(ctx, identity, dryRun); err != nil {
		switch {
		case errors.Is(err, installer.ErrKubeVersionNotSupported):
			klog.Infof("pod security policy is not supported in your kubernetes")
		case !k8serrors.IsAlreadyExists(err):
			return err
		}
	} else if !dryRun {
		klog.Infof("'%s' pod security policy created", utils.Bold(identity))
	}

	if err := installer.CreateRBACRoles(ctx, identity, dryRun); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}
	if !dryRun {
		klog.Infof("'%s' rbac roles created", utils.Bold(identity))
	}

	if err := installer.CreateOrUpdateConversionDeployment(ctx, identity, image, dryRun, registry, org); err != nil {
		return err
	}
	if !dryRun {
		klog.Infof("'%s' conversion deployment created", utils.Bold(identity))
	}

crdInstall:
	if err := registerCRDs(ctx, identity); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
		// if it exists
		if !dryRun && overwriteCRD {
			klog.V(4).Infof("overwriting CRDs")
			if err := unregisterCRDs(ctx); err != nil {
				if !k8serrors.IsNotFound(err) {
					return err
				}
			}
			klog.V(4).Infof("Writing CRDs again")
			goto crdInstall
		}
	}
	if !dryRun {
		klog.Infof("crds successfully registered")
	}

	if err := installer.CreateCSIDriver(ctx, identity, dryRun); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}
	if !dryRun {
		klog.Infof("'%s' csidriver created", utils.Bold(identity))
	}

	if err := installer.CreateStorageClass(ctx, identity, dryRun); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}
	if !dryRun {
		klog.Infof("'%s' storageclass created", utils.Bold(identity))
	}

	if err := installer.CreateService(ctx, identity, dryRun); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}
	if !dryRun {
		klog.Infof("'%s' service created", utils.Bold(identity))
	}

	if err := installer.CreateDaemonSet(ctx, identity, image, dryRun, registry, org, loopBackOnly, nodeSelector, tolerations, seccompProfile, apparmorProfile); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}
	if !dryRun {
		klog.Infof("'%s' daemonset created", utils.Bold(identity))
	}

	if err := installer.CreateDeployment(ctx, identity, image, dryRun, registry, org); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}
	if !dryRun {
		klog.Infof("'%s' deployment created", utils.Bold(identity))
	}

	if admissionControl {
		if err := installer.RegisterDriveValidationRules(ctx, identity, dryRun); err != nil {
			if !k8serrors.IsAlreadyExists(err) {
				return err
			}
		}
		if !dryRun {
			klog.Infof("'%s' drive validation rules registered", utils.Bold(identity))
		}
	}

	return nil
}
