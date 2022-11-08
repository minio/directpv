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
	return installNSDefault(ctx, v.Config)
}

func (v *defaultInstaller) installSecrets(ctx context.Context) error {
	return installSecretsDefault(ctx, v.Config)
}

func (v *defaultInstaller) installRBAC(ctx context.Context) error {
	return installRBACDefault(ctx, v.Config)
}

func (v *defaultInstaller) installCRD(ctx context.Context) error {
	return installCRDDefault(ctx, v.Config)
}

func (v *defaultInstaller) installCSIDriver(ctx context.Context) error {
	return installCSIDriverDefault(ctx, v.Config)
}

func (v *defaultInstaller) installStorageClass(ctx context.Context) error {
	return installStorageClassDefault(ctx, v.Config)
}

func (v *defaultInstaller) installService(ctx context.Context) error {
	return installServiceDefault(ctx, v.Config)
}

func (v *defaultInstaller) installDaemonset(ctx context.Context) error {
	return installDaemonsetDefault(ctx, v.Config)
}

func (v *defaultInstaller) installDeployment(ctx context.Context) error {
	return installDeploymentDefault(ctx, v.Config)
}

func (v *defaultInstaller) installAdminServerDeployment(ctx context.Context) error {
	return installAdminServerDeploymentDefault(ctx, v.Config)
}

// uninstallers
func (v *defaultInstaller) uninstallNS(ctx context.Context) error {
	return uninstallNSDefault(ctx, v.Config)
}

func (v *defaultInstaller) uninstallSecrets(ctx context.Context) error {
	return uninstallSecretsDefault(ctx, v.Config)
}

func (v *defaultInstaller) uninstallRBAC(ctx context.Context) error {
	return uninstallRBACDefault(ctx, v.Config)
}

func (v *defaultInstaller) uninstallCRD(ctx context.Context) error {
	return uninstallCRDDefault(ctx, v.Config)
}

func (v *defaultInstaller) uninstallCSIDriver(ctx context.Context) error {
	return uninstallCSIDriverDefault(ctx, v.Config)
}

func (v *defaultInstaller) uninstallStorageClass(ctx context.Context) error {
	return uninstallStorageClassDefault(ctx, v.Config)
}

func (v *defaultInstaller) uninstallService(ctx context.Context) error {
	return uninstallServiceDefault(ctx, v.Config)
}

func (v *defaultInstaller) uninstallDaemonset(ctx context.Context) error {
	return uninstallDaemonsetDefault(ctx, v.Config)
}

func (v *defaultInstaller) uninstallDeployment(ctx context.Context) error {
	return uninstallDeploymentDefault(ctx, v.Config)
}

func (v *defaultInstaller) uninstallAdminServerDeployment(ctx context.Context) error {
	return uninstallAdminServerDeploymentDefault(ctx, v.Config)
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
