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

// Tasks is a list of tasks to performed during installation and uninstallation
var Tasks = []Task{
	namespaceTask{},
	rbacTask{},
	pspTask{},
	crdTask{},
	migrateTask{},
	csiDriverTask{},
	storageClassTask{},
	daemonsetTask{},
	deploymentTask{},
}

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
	case args.dryRun():
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
		if !args.dryRun() {
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

	for _, task := range Tasks {
		if err = task.Start(ctx, args); err != nil {
			break
		}
		taskErr := task.Execute(ctx, args)
		if err = task.End(ctx, args, taskErr); err != nil {
			break
		}
	}

	if err == nil {
		return nil
	}

	var errs []string
	if err != nil {
		// TODO: revert upgraded components if any
		if rerr := revertNamespace(ctx); rerr != nil {
			errs = append(errs, fmt.Sprintf("Namespace: %v", rerr))
		}

		if rerr := revertRBAC(ctx); rerr != nil {
			errs = append(errs, fmt.Sprintf("RBAC: %v", rerr))
		}

		if rerr := revertPSP(ctx, args); rerr != nil {
			errs = append(errs, fmt.Sprintf("PSP: %v", rerr))
		}

		if rerr := revertCSIDriver(ctx, args); rerr != nil {
			errs = append(errs, fmt.Sprintf("CSIDriver: %v", rerr))
		}

		if rerr := revertStorageClass(ctx, args); rerr != nil {
			errs = append(errs, fmt.Sprintf("StorageClass: %v", rerr))
		}

		if rerr := revertDaemonSet(ctx, args); rerr != nil {
			errs = append(errs, fmt.Sprintf("DaemonSet: %v", rerr))
		}

		if rerr := revertDeployment(ctx); rerr != nil {
			errs = append(errs, fmt.Sprintf("Deployment: %v", rerr))
		}
	}

	if len(errs) == 0 {
		return err
	}

	return fmt.Errorf("%w; %v", err, strings.Join(errs, "; "))
}

// Uninstall removes DirectPV from kubernetes.
func Uninstall(ctx context.Context, quiet, force bool) (err error) {
	args := &Args{
		ForceUninstall: force,
		Quiet:          quiet,
	}
	for _, task := range Tasks {
		if err := task.Delete(ctx, args); err != nil {
			return err
		}
	}
	return nil
}
