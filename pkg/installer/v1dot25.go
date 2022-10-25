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

type v1dot25 struct {
	*Config
}

func newv1dot25(config *Config) *v1dot25 {
	config.enablePodSecurityAdmission = true
	return &v1dot25{
		Config: config,
	}
}

// installers
func (v *v1dot25) installNS(ctx context.Context) error {
	return installNSDefault(ctx, v.Config)
}

func (v *v1dot25) installRBAC(ctx context.Context) error {
	return installRBACDefault(ctx, v.Config)
}

func (v *v1dot25) installCRD(ctx context.Context) error {
	return installCRDDefault(ctx, v.Config)
}

func (v *v1dot25) installCSIDriver(ctx context.Context) error {
	return installCSIDriverDefault(ctx, v.Config)
}

func (v *v1dot25) installStorageClass(ctx context.Context) error {
	return installStorageClassDefault(ctx, v.Config)
}

func (v *v1dot25) installService(ctx context.Context) error {
	return installServiceDefault(ctx, v.Config)
}

func (v *v1dot25) installDaemonset(ctx context.Context) error {
	return installDaemonsetDefault(ctx, v.Config)
}

func (v *v1dot25) installDeployment(ctx context.Context) error {
	return installDeploymentDefault(ctx, v.Config)
}

// uninstallers
func (v *v1dot25) uninstallNS(ctx context.Context) error {
	return uninstallNSDefault(ctx, v.Config)
}

func (v *v1dot25) uninstallRBAC(ctx context.Context) error {
	return uninstallRBACDefault(ctx, v.Config)
}

func (v *v1dot25) uninstallCRD(ctx context.Context) error {
	return uninstallCRDDefault(ctx, v.Config)
}

func (v *v1dot25) uninstallCSIDriver(ctx context.Context) error {
	return uninstallCSIDriverDefault(ctx, v.Config)
}

func (v *v1dot25) uninstallStorageClass(ctx context.Context) error {
	return uninstallStorageClassDefault(ctx, v.Config)
}

func (v *v1dot25) uninstallService(ctx context.Context) error {
	return uninstallServiceDefault(ctx, v.Config)
}

func (v *v1dot25) uninstallDaemonset(ctx context.Context) error {
	return uninstallDaemonsetDefault(ctx, v.Config)
}

func (v *v1dot25) uninstallDeployment(ctx context.Context) error {
	return uninstallDeploymentDefault(ctx, v.Config)
}

func (v *v1dot25) Install(ctx context.Context) error {
	if err := v.installNS(ctx); err != nil {
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
	return v.installDeployment(ctx)
}

func (v *v1dot25) Uninstall(ctx context.Context) error {
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
	if err := v.uninstallStorageClass(ctx); err != nil {
		return err
	}
	if err := v.uninstallCSIDriver(ctx); err != nil {
		return err
	}
	if err := v.uninstallRBAC(ctx); err != nil {
		return err
	}
	return v.uninstallNS(ctx)
}
