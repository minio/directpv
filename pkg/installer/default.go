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

package installer

import (
	"context"

	"github.com/minio/directpv/pkg/utils"
)

type defaultInstaller struct {
	*Config
}

func newDefaultInstaller(config *Config) *defaultInstaller {
	config.enablePodSecurityAdmission = true
	return &defaultInstaller{
		Config: config,
	}
}

// installers
func (v *defaultInstaller) installNS(ctx context.Context) error {
	err := installNSDefault(ctx, v.Config)
	if err != nil {
		utils.Eprintf(v.Quiet, true, "unable to create Namespace; %v", err)
	}
	return err
}

func (v *defaultInstaller) installSecrets(ctx context.Context) error {
	err := installSecretsDefault(ctx, v.Config)
	if err != nil {
		utils.Eprintf(v.Quiet, true, "unable to create Secrets; %v", err)
	}
	return err
}

func (v *defaultInstaller) installRBAC(ctx context.Context) error {
	err := installRBACDefault(ctx, v.Config)
	if err != nil {
		utils.Eprintf(v.Quiet, true, "unable to create RBAC; %v", err)
	}
	return err
}

func (v *defaultInstaller) installCRD(ctx context.Context) error {
	err := installCRDDefault(ctx, v.Config)
	if err != nil {
		utils.Eprintf(v.Quiet, true, "unable to create CRDs; %v", err)
	}
	return err
}

func (v *defaultInstaller) installCSIDriver(ctx context.Context) error {
	err := installCSIDriverDefault(ctx, v.Config)
	if err != nil {
		utils.Eprintf(v.Quiet, true, "unable to create CSI driver; %v", err)
	}
	return err
}

func (v *defaultInstaller) installStorageClass(ctx context.Context) error {
	err := installStorageClassDefault(ctx, v.Config)
	if err != nil {
		utils.Eprintf(v.Quiet, true, "unable to create storage class; %v", err)
	}
	return err
}

func (v *defaultInstaller) installService(ctx context.Context) error {
	err := installServiceDefault(ctx, v.Config)
	if err != nil {
		utils.Eprintf(v.Quiet, true, "unable to create service; %v", err)
	}
	return err
}

func (v *defaultInstaller) installDaemonset(ctx context.Context) error {
	err := installDaemonsetDefault(ctx, v.Config)
	if err != nil {
		utils.Eprintf(v.Quiet, true, "unable to create daemonset; %v")
	}
	return err
}

func (v *defaultInstaller) installDeployment(ctx context.Context) error {
	err := installDeploymentDefault(ctx, v.Config)
	if err != nil {
		utils.Eprintf(v.Quiet, true, "unable to create deployment; %v", err)
	}
	return err
}

func (v *defaultInstaller) installAdminServerDeployment(ctx context.Context) error {
	err := installAdminServerDeploymentDefault(ctx, v.Config)
	if err != nil {
		utils.Eprintf(v.Quiet, true, "unable to create API server deployment; %v", err)
	}
	return err
}

// uninstallers
func (v *defaultInstaller) uninstallNS(ctx context.Context) error {
	err := uninstallNSDefault(ctx, v.Config)
	if err != nil {
		utils.Eprintf(v.Quiet, true, "unable to delete namespace; %v", err)
	}
	return err
}

func (v *defaultInstaller) uninstallSecrets(ctx context.Context) error {
	err := uninstallSecretsDefault(ctx, v.Config)
	if err != nil {
		utils.Eprintf(v.Quiet, true, "unable to delete secrets; %v", err)
	}
	return err
}

func (v *defaultInstaller) uninstallRBAC(ctx context.Context) error {
	err := uninstallRBACDefault(ctx, v.Config)
	if err != nil {
		utils.Eprintf(v.Quiet, true, "unable to delete RBAC; %v", err)
	}
	return err
}

func (v *defaultInstaller) uninstallCRD(ctx context.Context) error {
	err := uninstallCRDDefault(ctx, v.Config)
	if err != nil {
		utils.Eprintf(v.Quiet, true, "unable to delete CRDs; %v", err)
	}
	return err
}

func (v *defaultInstaller) uninstallCSIDriver(ctx context.Context) error {
	err := uninstallCSIDriverDefault(ctx, v.Config)
	if err != nil {
		utils.Eprintf(v.Quiet, true, "unable to delete CSI driver; %v", err)
	}
	return err
}

func (v *defaultInstaller) uninstallStorageClass(ctx context.Context) error {
	err := uninstallStorageClassDefault(ctx, v.Config)
	if err != nil {
		utils.Eprintf(v.Quiet, true, "unable to delete storage class; %v", err)
	}
	return err
}

func (v *defaultInstaller) uninstallService(ctx context.Context) error {
	err := uninstallServiceDefault(ctx, v.Config)
	if err != nil {
		utils.Eprintf(v.Quiet, true, "unable to delete service; %v", err)
	}
	return err
}

func (v *defaultInstaller) uninstallDaemonset(ctx context.Context) error {
	err := uninstallDaemonsetDefault(ctx, v.Config)
	if err != nil {
		utils.Eprintf(v.Quiet, true, "unable to delete daemonset; %v", err)
	}
	return err
}

func (v *defaultInstaller) uninstallDeployment(ctx context.Context) error {
	err := uninstallDeploymentDefault(ctx, v.Config)
	if err != nil {
		utils.Eprintf(v.Quiet, true, "unable to delete deployment; %v", err)
	}
	return err
}

func (v *defaultInstaller) uninstallAdminServerDeployment(ctx context.Context) error {
	err := uninstallAdminServerDeploymentDefault(ctx, v.Config)
	if err != nil {
		utils.Eprintf(v.Quiet, true, "unable to delete API server Deployment; %v", err)
	}
	return err
}

func (v *defaultInstaller) Install(ctx context.Context) error {
	if err := v.installNS(ctx); err != nil {
		return err
	}
	if err := v.installSecrets(ctx); err != nil {
		return err
	}
	if err := v.installRBAC(ctx); err != nil {
		return err
	}
	if err := v.installCRD(ctx); err != nil {
		return err
	}
	if err := v.installCSIDriver(ctx); err != nil {
		return err
	}
	if err := v.installStorageClass(ctx); err != nil {
		return err
	}
	if err := v.installService(ctx); err != nil {
		return err
	}
	if err := v.installDaemonset(ctx); err != nil {
		return err
	}
	if err := v.installDeployment(ctx); err != nil {
		return err
	}
	return v.installAdminServerDeployment(ctx)
}

func (v *defaultInstaller) Uninstall(ctx context.Context) error {
	if err := v.uninstallCRD(ctx); err != nil {
		return err
	}
	if err := v.uninstallAdminServerDeployment(ctx); err != nil {
		return err
	}
	if err := v.uninstallDeployment(ctx); err != nil {
		return err
	}
	if err := v.uninstallDaemonset(ctx); err != nil {
		return err
	}
	if err := v.uninstallService(ctx); err != nil {
		return err
	}
	if err := v.uninstallStorageClass(ctx); err != nil {
		return err
	}
	if err := v.uninstallCSIDriver(ctx); err != nil {
		return err
	}
	if err := v.uninstallRBAC(ctx); err != nil {
		return err
	}
	if err := v.uninstallSecrets(ctx); err != nil {
		return err
	}
	return v.uninstallNS(ctx)
}
