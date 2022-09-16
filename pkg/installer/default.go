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
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
)

type defaultInstaller struct {
	*Config
}

func newDefaultInstaller(config *Config) *defaultInstaller {
	return &defaultInstaller{
		Config: config,
	}
}

// installers
func (v *defaultInstaller) installNS(ctx context.Context) error {
	timer := time.AfterFunc(
		3*time.Second,
		func() { fmt.Fprintln(os.Stderr, color.HiYellowString("WARNING: too long to create Namespace")) },
	)
	defer timer.Stop()

	err := installNSDefault(ctx, v.Config)
	if err != nil && !v.DryRun {
		fmt.Fprintf(os.Stderr, "%v unable to create Namespace; %v", color.HiRedString("ERROR"), err)
	}
	return err
}

func (v *defaultInstaller) installRBAC(ctx context.Context) error {
	timer := time.AfterFunc(
		3*time.Second,
		func() { fmt.Fprintln(os.Stderr, color.HiYellowString("WARNING: too long to create RBAC")) },
	)
	defer timer.Stop()

	err := installRBACDefault(ctx, v.Config)
	if err != nil && !v.DryRun {
		fmt.Fprintf(os.Stderr, "%v unable to create RBAC; %v", color.HiRedString("ERROR"), err)
	}
	return err
}

func (v *defaultInstaller) installPSP(ctx context.Context) error {
	timer := time.AfterFunc(
		3*time.Second,
		func() {
			fmt.Fprintln(os.Stderr, color.HiYellowString("WARNING: too long to create PodSecurityPolicies"))
		},
	)
	defer timer.Stop()

	err := installPSPDefault(ctx, v.Config)
	if err != nil && !v.DryRun {
		fmt.Fprintf(os.Stderr, "%v unable to create PodSecurityPolicies; %v", color.HiRedString("ERROR"), err)
	}
	return err
}

func (v *defaultInstaller) installCRD(ctx context.Context) error {
	timer := time.AfterFunc(
		3*time.Second,
		func() { fmt.Fprintln(os.Stderr, color.HiYellowString("WARNING: too long to create CRDs")) },
	)
	defer timer.Stop()

	err := installCRDDefault(ctx, v.Config)
	if err != nil && !v.DryRun {
		fmt.Fprintf(os.Stderr, "%v unable to create CRDs; %v", color.HiRedString("ERROR"), err)
	}
	return err
}

func (v *defaultInstaller) installCSIDriver(ctx context.Context) error {
	timer := time.AfterFunc(
		3*time.Second,
		func() { fmt.Fprintln(os.Stderr, color.HiYellowString("WARNING: too long to create CSI driver")) },
	)
	defer timer.Stop()

	err := installCSIDriverDefault(ctx, v.Config)
	if err != nil && !v.DryRun {
		fmt.Fprintf(os.Stderr, "%v unable to create CSI driver; %v", color.HiRedString("ERROR"), err)
	}
	return err
}

func (v *defaultInstaller) installStorageClass(ctx context.Context) error {
	timer := time.AfterFunc(
		3*time.Second,
		func() { fmt.Fprintln(os.Stderr, color.HiYellowString("WARNING: too long to create Storage class")) },
	)
	defer timer.Stop()

	err := installStorageClassDefault(ctx, v.Config)
	if err != nil && !v.DryRun {
		fmt.Fprintf(os.Stderr, "%v unable to create Storage class; %v", color.HiRedString("ERROR"), err)
	}
	return err
}

func (v *defaultInstaller) installService(ctx context.Context) error {
	timer := time.AfterFunc(
		3*time.Second,
		func() { fmt.Fprintln(os.Stderr, color.HiYellowString("WARNING: too long to create Service")) },
	)
	defer timer.Stop()

	err := installServiceDefault(ctx, v.Config)
	if err != nil && !v.DryRun {
		fmt.Fprintf(os.Stderr, "%v unable to create Service; %v", color.HiRedString("ERROR"), err)
	}
	return err
}

func (v *defaultInstaller) installDaemonset(ctx context.Context) error {
	timer := time.AfterFunc(
		3*time.Second,
		func() { fmt.Fprintln(os.Stderr, color.HiYellowString("WARNING: too long to create Daemon set")) },
	)
	defer timer.Stop()

	err := installDaemonsetDefault(ctx, v.Config)
	if err != nil && !v.DryRun {
		fmt.Fprintf(os.Stderr, "%v unable to create Daemon set; %v", color.HiRedString("ERROR"), err)
	}
	return err
}

func (v *defaultInstaller) installDeployment(ctx context.Context) error {
	timer := time.AfterFunc(
		3*time.Second,
		func() { fmt.Fprintln(os.Stderr, color.HiYellowString("WARNING: too long to create Deployment")) },
	)
	defer timer.Stop()

	err := installDeploymentDefault(ctx, v.Config)
	if err != nil && !v.DryRun {
		fmt.Fprintf(os.Stderr, "%v unable to create Deployment; %v", color.HiRedString("ERROR"), err)
	}
	return err
}

func (v *defaultInstaller) installValidationRules(ctx context.Context) error {
	timer := time.AfterFunc(
		3*time.Second,
		func() { fmt.Fprintln(os.Stderr, color.HiYellowString("WARNING: too long to create Validation rules")) },
	)
	defer timer.Stop()

	err := installValidationRulesDefault(ctx, v.Config)
	if err != nil && !v.DryRun {
		fmt.Fprintf(os.Stderr, "%v unable to create Validation rules; %v", color.HiRedString("ERROR"), err)
	}
	return err
}

// uninstallers
func (v *defaultInstaller) uninstallNS(ctx context.Context) error {
	timer := time.AfterFunc(
		3*time.Second,
		func() { fmt.Fprintln(os.Stderr, color.HiYellowString("WARNING: too long to delete Namespace")) },
	)
	defer timer.Stop()

	err := uninstallNSDefault(ctx, v.Config)
	if err != nil && !v.DryRun {
		fmt.Fprintf(os.Stderr, "%v unable to delete Namespace; %v", color.HiRedString("ERROR"), err)
	}
	return err
}

func (v *defaultInstaller) uninstallRBAC(ctx context.Context) error {
	timer := time.AfterFunc(
		3*time.Second,
		func() { fmt.Fprintln(os.Stderr, color.HiYellowString("WARNING: too long to delete RBAC")) },
	)
	defer timer.Stop()

	err := uninstallRBACDefault(ctx, v.Config)
	if err != nil && !v.DryRun {
		fmt.Fprintf(os.Stderr, "%v unable to delete RBAC; %v", color.HiRedString("ERROR"), err)
	}
	return err
}

func (v *defaultInstaller) uninstallPSP(ctx context.Context) error {
	timer := time.AfterFunc(
		3*time.Second,
		func() {
			fmt.Fprintln(os.Stderr, color.HiYellowString("WARNING: too long to delete PodSecurityPolicies"))
		},
	)
	defer timer.Stop()

	err := uninstallPSPDefault(ctx, v.Config)
	if err != nil && !v.DryRun {
		fmt.Fprintf(os.Stderr, "%v unable to delete PodSecurityPolicies; %v", color.HiRedString("ERROR"), err)
	}
	return err
}

func (v *defaultInstaller) uninstallCRD(ctx context.Context) error {
	timer := time.AfterFunc(
		3*time.Second,
		func() { fmt.Fprintln(os.Stderr, color.HiYellowString("WARNING: too long to delete CRDs")) },
	)
	defer timer.Stop()

	err := uninstallCRDDefault(ctx, v.Config)
	if err != nil && !v.DryRun {
		fmt.Fprintf(os.Stderr, "%v unable to delete CRDs; %v", color.HiRedString("ERROR"), err)
	}
	return err
}

func (v *defaultInstaller) uninstallCSIDriver(ctx context.Context) error {
	timer := time.AfterFunc(
		3*time.Second,
		func() { fmt.Fprintln(os.Stderr, color.HiYellowString("WARNING: too long to delete CSI driver")) },
	)
	defer timer.Stop()

	err := uninstallCSIDriverDefault(ctx, v.Config)
	if err != nil && !v.DryRun {
		fmt.Fprintf(os.Stderr, "%v unable to delete CSI driver; %v", color.HiRedString("ERROR"), err)
	}
	return err
}

func (v *defaultInstaller) uninstallStorageClass(ctx context.Context) error {
	timer := time.AfterFunc(
		3*time.Second,
		func() { fmt.Fprintln(os.Stderr, color.HiYellowString("WARNING: too long to delete Storage class")) },
	)
	defer timer.Stop()

	err := uninstallStorageClassDefault(ctx, v.Config)
	if err != nil && !v.DryRun {
		fmt.Fprintf(os.Stderr, "%v unable to delete Storage class; %v", color.HiRedString("ERROR"), err)
	}
	return err
}

func (v *defaultInstaller) uninstallService(ctx context.Context) error {
	timer := time.AfterFunc(
		3*time.Second,
		func() { fmt.Fprintln(os.Stderr, color.HiYellowString("WARNING: too long to delete Service")) },
	)
	defer timer.Stop()

	err := uninstallServiceDefault(ctx, v.Config)
	if err != nil && !v.DryRun {
		fmt.Fprintf(os.Stderr, "%v unable to delete Service; %v", color.HiRedString("ERROR"), err)
	}
	return err
}

func (v *defaultInstaller) uninstallDaemonset(ctx context.Context) error {
	timer := time.AfterFunc(
		3*time.Second,
		func() { fmt.Fprintln(os.Stderr, color.HiYellowString("WARNING: too long to delete Daemon set")) },
	)
	defer timer.Stop()

	err := uninstallDaemonsetDefault(ctx, v.Config)
	if err != nil && !v.DryRun {
		fmt.Fprintf(os.Stderr, "%v unable to delete Daemon set; %v", color.HiRedString("ERROR"), err)
	}
	return err
}

func (v *defaultInstaller) uninstallDeployment(ctx context.Context) error {
	timer := time.AfterFunc(
		3*time.Second,
		func() { fmt.Fprintln(os.Stderr, color.HiYellowString("WARNING: too long to delete Deployment")) },
	)
	defer timer.Stop()

	err := uninstallDeploymentDefault(ctx, v.Config)
	if err != nil && !v.DryRun {
		fmt.Fprintf(os.Stderr, "%v unable to delete Deployment; %v", color.HiRedString("ERROR"), err)
	}
	return err
}

func (v *defaultInstaller) uninstallValidationRules(ctx context.Context) error {
	timer := time.AfterFunc(
		3*time.Second,
		func() { fmt.Fprintln(os.Stderr, color.HiYellowString("WARNING: too long to delete Validation rules")) },
	)
	defer timer.Stop()

	err := uninstallValidationRulesDefault(ctx, v.Config)
	if err != nil && !v.DryRun {
		fmt.Fprintf(os.Stderr, "%v unable to delete Validation rules; %v", color.HiRedString("ERROR"), err)
	}
	return err
}

func (v *defaultInstaller) Install(ctx context.Context) error {
	if err := v.installNS(ctx); err != nil {
		return err
	}
	if err := v.installRBAC(ctx); err != nil {
		return err
	}
	if err := v.installPSP(ctx); err != nil {
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

func (v *defaultInstaller) Uninstall(ctx context.Context) error {
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
	if err := v.uninstallPSP(ctx); err != nil {
		return err
	}
	if err := v.uninstallRBAC(ctx); err != nil {
		return err
	}
	return v.uninstallNS(ctx)
}
