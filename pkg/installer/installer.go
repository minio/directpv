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
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/utils"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/klog/v2"
)

const (
	// TotalTasks denotes the total number of tasks to be executed
	TotalTasks = 9
)

func getKubeVersion() (major, minor uint, err error) {
	versionInfo, err := k8s.DiscoveryClient().ServerVersion()
	if err != nil {
		return 0, 0, err
	}

	var u64 uint64
	if u64, err = strconv.ParseUint(versionInfo.Major, 10, 64); err != nil {
		return 0, 0, fmt.Errorf("unable to parse major version %v; %v", versionInfo.Major, err)
	}
	major = uint(u64)

	minorString := versionInfo.Minor
	if strings.Contains(versionInfo.GitVersion, "-eks-") {
		// Do trimming only for EKS.
		// Refer https://github.com/aws/containers-roadmap/issues/1404
		i := strings.IndexFunc(minorString, func(r rune) bool { return r < '0' || r > '9' })
		if i > -1 {
			minorString = minorString[:i]
		}
	}
	if u64, err = strconv.ParseUint(minorString, 10, 64); err != nil {
		return 0, 0, fmt.Errorf("unable to parse minor version %v; %v", minor, err)
	}
	minor = uint(u64)

	return major, minor, nil
}

// Install performs DirectPV installation on kubernetes.
func Install(ctx context.Context, args *Args) (err error) {
	defer func() {
		if !sendDoneMessage(ctx, args.ProgressCh, err) {
			err = errSendProgress
		}
	}()

	err = args.validate()
	if err != nil {
		return err
	}

	switch {
	case args.DryRun:
		if args.KubeVersion == nil {
			// default higher version
			if args.KubeVersion, err = version.ParseSemantic("1.25.0"); err != nil {
				klog.Fatalf("this should not happen; %v", err)
			}
		}
	default:
		major, minor, err := getKubeVersion()
		if err != nil {
			return err
		}
		args.KubeVersion, err = version.ParseSemantic(fmt.Sprintf("%v.%v.0", major, minor))
		if err != nil {
			klog.Fatalf("this should not happen; %v", err)
		}
	}

	if args.KubeVersion.Major() == 1 {
		if args.KubeVersion.Minor() < 20 {
			args.csiProvisionerImage = "csi-provisioner:v2.2.0-go1.18"
		}
		args.podSecurityAdmission = args.KubeVersion.Minor() > 24
	}

	if args.KubeVersion.Major() != 1 ||
		args.KubeVersion.Minor() < 18 ||
		args.KubeVersion.Minor() > 25 {
		if !args.DryRun {
			utils.Eprintf(
				args.Quiet,
				false,
				"%v\n",
				color.HiYellowString(
					"Installing on unsupported Kubernetes v%v.%v",
					args.KubeVersion.Major(),
					args.KubeVersion.Minor(),
				),
			)
		}
	}

	execute := func(fn func(context.Context, *Args) error, totalSteps int) (err error) {
		if !sendStartMessage(ctx, args.ProgressCh, totalSteps) {
			return errSendProgress
		}
		defer func() {
			if !sendEndMessage(ctx, args.ProgressCh, err) {
				err = errSendProgress
			}
		}()
		return fn(ctx, args)
	}

	if err := execute(createNamespace, totalNamespaceSteps); err != nil {
		return err
	}

	if err := execute(createRBAC, totalRBACSteps); err != nil {
		return err
	}

	if err := execute(createPSP, totalPSPSteps); err != nil {
		return err
	}

	if err := execute(createCRDs, totalCRDSteps); err != nil {
		return err
	}

	if err := execute(Migrate, totalMigrateSteps); err != nil {
		return err
	}

	if err := execute(createCSIDriver, totalCSIDriverSteps); err != nil {
		return err
	}

	if err := execute(createStorageClass, totalStorageClassSteps); err != nil {
		return err
	}

	if err := execute(createDaemonset, totalDaemonsetSteps); err != nil {
		return err
	}

	return execute(createDeployment, totalDeploymentSteps)
}

// Uninstall removes DirectPV from kubernetes.
func Uninstall(ctx context.Context, quiet, force bool) (err error) {
	major, minor, err := getKubeVersion()
	if err != nil {
		return err
	}

	podSecurityAdmission := (major == 1 && minor > 24)

	if err := deleteNamespace(ctx); err != nil {
		return err
	}

	if err := deleteRBAC(ctx); err != nil {
		return err
	}

	if !podSecurityAdmission {
		if err := deletePSP(ctx); err != nil {
			return err
		}
	}

	if err := deleteCRDs(ctx, force); err != nil {
		return err
	}

	if err := deleteCSIDriver(ctx); err != nil {
		return err
	}

	if err := deleteStorageClass(ctx); err != nil {
		return err
	}

	if err := deleteDaemonset(ctx); err != nil {
		return err
	}

	if err := deleteDeployment(ctx); err != nil {
		return err
	}

	return nil
}
