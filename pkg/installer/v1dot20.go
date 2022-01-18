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

type v1dot20 struct {
	*Config
}

func newV1Dot20(config *Config) *v1dot20 {
	return &v1dot20{
		Config: config,
	}
}

// installers
func (v *v1dot20) installNS(ctx context.Context) error {
	return installNSDefault(ctx, v.Config)
}
func (v *v1dot20) installRBAC(ctx context.Context) error {
	return installRBACDefault(ctx, v.Config)
}
func (v *v1dot20) installPSP(ctx context.Context) error {
	return installPSPDefault(ctx, v.Config)
}
func (v *v1dot20) installConversionSecret(ctx context.Context) error {
	return installConversionSecretDefault(ctx, v.Config)
}
func (v *v1dot20) installCRD(ctx context.Context) error {
	return installCRDDefault(ctx, v.Config)
}
func (v *v1dot20) installCSIDriver(ctx context.Context) error {
	return installCSIDriverDefault(ctx, v.Config)
}
func (v *v1dot20) installStorageClass(ctx context.Context) error {
	return installStorageClassDefault(ctx, v.Config)
}
func (v *v1dot20) installService(ctx context.Context) error {
	return installServiceDefault(ctx, v.Config)
}
func (v *v1dot20) installDaemonset(ctx context.Context) error {
	return installDaemonsetDefault(ctx, v.Config)
}
func (v *v1dot20) installDeployment(ctx context.Context) error {
	return installDeploymentDefault(ctx, v.Config)
}
func (v *v1dot20) installValidationRules(ctx context.Context) error {
	return installValidationRulesDefault(ctx, v.Config)
}

// uninstallers
func (v *v1dot20) uninstallNS(ctx context.Context) error {
	return uninstallNSDefault(ctx, v.Config)
}
func (v *v1dot20) uninstallRBAC(ctx context.Context) error {
	return uninstallRBACDefault(ctx, v.Config)
}
func (v *v1dot20) uninstallPSP(ctx context.Context) error {
	return uninstallPSPDefault(ctx, v.Config)
}
func (v *v1dot20) uninstallConversionSecret(ctx context.Context) error {
	return uninstallConversionSecretDefault(ctx, v.Config)
}
func (v *v1dot20) uninstallCRD(ctx context.Context) error {
	return uninstallCRDDefault(ctx, v.Config)
}
func (v *v1dot20) uninstallCSIDriver(ctx context.Context) error {
	return uninstallCSIDriverDefault(ctx, v.Config)
}
func (v *v1dot20) uninstallStorageClass(ctx context.Context) error {
	return uninstallStorageClassDefault(ctx, v.Config)
}
func (v *v1dot20) uninstallService(ctx context.Context) error {
	return uninstallServiceDefault(ctx, v.Config)
}
func (v *v1dot20) uninstallDaemonset(ctx context.Context) error {
	return uninstallDaemonsetDefault(ctx, v.Config)
}
func (v *v1dot20) uninstallDeployment(ctx context.Context) error {
	return uninstallDeploymentDefault(ctx, v.Config)
}
func (v *v1dot20) uninstallValidationRules(ctx context.Context) error {
	return uninstallValidationRulesDefault(ctx, v.Config)
}

func (v *v1dot20) Install(ctx context.Context) error {
	if err := v.installNS(ctx); err != nil {
		return err
	}
	if err := v.installRBAC(ctx); err != nil {
		return err
	}
	if err := v.installPSP(ctx); err != nil {
		return err
	}
	if err := v.installConversionSecret(ctx); err != nil {
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
	return v.installValidationRules(ctx)
}

func (v *v1dot20) Uninstall(ctx context.Context) error {
	if err := v.uninstallCRD(ctx); err != nil {
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
	if err := v.uninstallValidationRules(ctx); err != nil {
		return err
	}
	if err := v.uninstallStorageClass(ctx); err != nil {
		return err
	}
	if err := v.uninstallCSIDriver(ctx); err != nil {
		return err
	}
	if err := v.uninstallConversionSecret(ctx); err != nil {
		return err
	}
	if err := v.uninstallPSP(ctx); err != nil {
		return err
	}
	if err := v.uninstallRBAC(ctx); err != nil {
		return err
	}
	return v.uninstallNS(ctx)
}
