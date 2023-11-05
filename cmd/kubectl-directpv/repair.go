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

package main

import (
	"context"
	"errors"
	"os"
	"strings"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/drive"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	forceFlag           = false
	disablePrefetchFlag = false
)

var repairCmd = &cobra.Command{
	Use:           "repair DRIVE ...",
	Short:         "Repair filesystem of drives",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`1. Repair drives
   $ kubectl {PLUGIN_NAME} repair 3b562992-f752-4a41-8be4-4e688ae8cd4c`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		driveIDArgs = args
		if err := validateRepairCmd(); err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", err)
			os.Exit(-1)
		}

		repairMain(c.Context())
	},
}

func init() {
	setFlagOpts(repairCmd)

	addDryRunFlag(repairCmd, "Repair drives with no modify mode")
	repairCmd.PersistentFlags().BoolVar(&forceFlag, "force", forceFlag, "Force log zeroing")
	repairCmd.PersistentFlags().BoolVar(&disablePrefetchFlag, "disable-prefetch", disablePrefetchFlag, "Disable prefetching of inode and directory blocks")
}

func validateRepairCmd() error {
	if err := validateDriveIDArgs(); err != nil {
		return err
	}

	if len(driveIDArgs) == 0 {
		return errors.New("no drive provided to repair")
	}

	return nil
}

func repairMain(ctx context.Context) {
	var failed bool

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	containerImage, imagePullSecrets, tolerations, err := getContainerParams(ctx)
	if err != nil {
		utils.Eprintf(quietFlag, true, "unable to container parameters from daemonset; %v\n", err)
		os.Exit(1)
	}

	resultCh := drive.NewLister().
		DriveIDSelector(driveIDSelectors).
		IgnoreNotFound(true).
		List(ctx)
	for result := range resultCh {
		if result.Err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", result.Err)
			os.Exit(1)
		}

		jobName := "repair-" + result.Drive.Name
		if _, err := k8s.KubeClient().BatchV1().Jobs(consts.AppName).Get(ctx, jobName, metav1.GetOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				utils.Eprintf(quietFlag, true, "unable to get repair job %v; %v\n", jobName, err)
				failed = true
				continue
			}
		} else {
			utils.Eprintf(quietFlag, true, "job %v already exists\n", jobName)
			continue
		}

		nodeID := string(result.Drive.GetNodeID())

		containerArgs := []string{"/directpv", "repair", result.Drive.Name, "--kube-node-name=" + nodeID}
		if forceFlag {
			containerArgs = append(containerArgs, "--force")
		}
		if disablePrefetchFlag {
			containerArgs = append(containerArgs, "--disable-prefetch")
		}
		if dryRunFlag {
			containerArgs = append(containerArgs, "--dry-run")
		}

		backOffLimit := int32(1)

		volumes := []corev1.Volume{
			k8s.NewHostPathVolume(consts.AppRootDirVolumeName, consts.AppRootDirVolumePath),
			k8s.NewHostPathVolume(consts.LegacyAppRootDirVolumeName, consts.LegacyAppRootDirVolumePath),
			k8s.NewHostPathVolume(consts.SysDirVolumeName, consts.SysDirVolumePath),
			k8s.NewHostPathVolume(consts.DevDirVolumeName, consts.DevDirVolumePath),
			k8s.NewHostPathVolume(consts.RunUdevDataVolumeName, consts.RunUdevDataVolumePath),
		}

		volumeMounts := []corev1.VolumeMount{
			k8s.NewVolumeMount(consts.AppRootDirVolumeName, consts.AppRootDirVolumePath, corev1.MountPropagationBidirectional, false),
			k8s.NewVolumeMount(consts.LegacyAppRootDirVolumeName, consts.LegacyAppRootDirVolumePath, corev1.MountPropagationBidirectional, false),
			k8s.NewVolumeMount(consts.SysDirVolumeName, consts.SysDirVolumePath, corev1.MountPropagationBidirectional, false),
			k8s.NewVolumeMount(consts.DevDirVolumeName, consts.DevDirVolumePath, corev1.MountPropagationHostToContainer, true),
			k8s.NewVolumeMount(consts.RunUdevDataVolumeName, consts.RunUdevDataVolumePath, corev1.MountPropagationBidirectional, true),
		}

		privileged := true

		job := batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jobName,
				Namespace: consts.AppName,
			},
			Spec: batchv1.JobSpec{
				BackoffLimit: &backOffLimit,
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						NodeSelector:       map[string]string{string(directpvtypes.NodeLabelKey): nodeID},
						ServiceAccountName: consts.Identity,
						Tolerations:        tolerations,
						ImagePullSecrets:   imagePullSecrets,
						Volumes:            volumes,
						Containers: []corev1.Container{
							{
								Name:                     jobName,
								Image:                    containerImage,
								Command:                  containerArgs,
								SecurityContext:          &corev1.SecurityContext{Privileged: &privileged},
								VolumeMounts:             volumeMounts,
								TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
								TerminationMessagePath:   "/var/log/repair-termination-log",
							},
						},
						RestartPolicy: corev1.RestartPolicyNever,
					},
				},
			},
		}

		if _, err := k8s.KubeClient().BatchV1().Jobs(consts.AppName).Create(ctx, &job, metav1.CreateOptions{}); err != nil {
			utils.Eprintf(quietFlag, true, "unable to create repair job %v; %v\n", jobName, err)
		} else {
			utils.Eprintf(quietFlag, false, "repair job %v for drive %v is created\n", jobName, result.Drive.Name)
		}
	}

	if failed {
		os.Exit(1)
	}
}
