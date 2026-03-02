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

	"github.com/fatih/color"
	"github.com/minio/directpv/pkg/client"
	legacyclient "github.com/minio/directpv/pkg/legacy/client"
	"github.com/minio/directpv/pkg/utils"
)

// GetDefaultTasks returns the installer tasks to be run
func GetDefaultTasks(client *client.Client, legacyClient *legacyclient.Client) []Task {
	return []Task{
		namespaceTask{client},
		rbacTask{client},
		crdTask{client},
		migrateTask{client, legacyClient},
		csiDriverTask{client},
		storageClassTask{client},
		daemonsetTask{client},
		deploymentTask{client},
	}
}

// Install performs DirectPV installation on kubernetes.
func Install(ctx context.Context, args *Args, tasks []Task) (err error) {
	defer func() {
		if !sendDoneMessage(ctx, args.ProgressCh, err) {
			err = errSendProgress
		}
	}()

	err = args.validate()
	if err != nil {
		return err
	}

	if args.KubeVersion.Major() == 1 {
		if args.KubeVersion.Minor() < 20 {
			args.csiProvisionerImage = csiProvisionerImageV2_2_0
		}
		args.podSecurityAdmission = args.KubeVersion.Minor() > 24
	}

	if args.KubeVersion.Major() != 1 ||
		args.KubeVersion.Minor() < 18 ||
		args.KubeVersion.Minor() > 35 {
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

	for _, task := range tasks {
		if err := task.Start(ctx, args); err != nil {
			return err
		}
		taskErr := task.Execute(ctx, args)
		if err := task.End(ctx, args, taskErr); err != nil {
			return err
		}
	}

	return nil
}

// Uninstall removes DirectPV from kubernetes.
func Uninstall(ctx context.Context, quiet, force bool, tasks []Task) (err error) {
	args := &Args{
		ForceUninstall: force,
		Quiet:          quiet,
	}
	for _, task := range tasks {
		if err := task.Delete(ctx, args); err != nil {
			return err
		}
	}
	return nil
}
