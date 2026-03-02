// This file is part of MinIO DirectPV
// Copyright (c) 2024 MinIO, Inc.
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

package admin

import (
	"context"
	"errors"
	"fmt"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	ttlSecondsAfterFinished = int32(5 * 60) // 5 Minutes
	backOffLimit            = int32(1)

	repairJobVolumes = []corev1.Volume{
		k8s.NewHostPathVolume(consts.AppRootDirVolumeName, consts.AppRootDirVolumePath),
		k8s.NewHostPathVolume(consts.LegacyAppRootDirVolumeName, consts.LegacyAppRootDirVolumePath),
		k8s.NewHostPathVolume(consts.SysDirVolumeName, consts.SysDirVolumePath),
		k8s.NewHostPathVolume(consts.DevDirVolumeName, consts.DevDirVolumePath),
		k8s.NewHostPathVolume(consts.RunUdevDataVolumeName, consts.RunUdevDataVolumePath),
	}

	repairJobVolumeMounts = []corev1.VolumeMount{
		k8s.NewVolumeMount(consts.AppRootDirVolumeName, consts.AppRootDirVolumePath, corev1.MountPropagationBidirectional, false),
		k8s.NewVolumeMount(consts.LegacyAppRootDirVolumeName, consts.LegacyAppRootDirVolumePath, corev1.MountPropagationBidirectional, false),
		k8s.NewVolumeMount(consts.SysDirVolumeName, consts.SysDirVolumePath, corev1.MountPropagationBidirectional, false),
		k8s.NewVolumeMount(consts.DevDirVolumeName, consts.DevDirVolumePath, corev1.MountPropagationHostToContainer, true),
		k8s.NewVolumeMount(consts.RunUdevDataVolumeName, consts.RunUdevDataVolumePath, corev1.MountPropagationBidirectional, true),
	}
)

// RepairArgs represents the arguments to repair a drive
type RepairArgs struct {
	DriveIDs            []directpvtypes.DriveID
	DryRun              bool
	ForceFlag           bool
	DisablePrefetchFlag bool
}

// RepairResult represents result of repaired drive
type RepairResult struct {
	JobName   string
	DriveName directpvtypes.DriveName
	DriveID   directpvtypes.DriveID
}

type repairContainerParams struct {
	containerImage   string
	imagePullSecrets []corev1.LocalObjectReference
	tolerations      []corev1.Toleration
	annotations      map[string]string
	securityContext  *corev1.SecurityContext
}

func (client *Client) getContainerParams(ctx context.Context) (params repairContainerParams, err error) {
	daemonSet, err := client.Kube().AppsV1().DaemonSets(consts.AppName).Get(
		ctx, consts.NodeServerName, metav1.GetOptions{},
	)

	if err != nil && !apierrors.IsNotFound(err) {
		return params, err
	}

	if daemonSet == nil || daemonSet.UID == "" {
		return params, errors.New("invalid daemonset found")
	}

	for _, container := range daemonSet.Spec.Template.Spec.Containers {
		if container.Name == consts.NodeServerName {
			params.containerImage = container.Image
			params.securityContext = container.SecurityContext
			break
		}
	}

	params.imagePullSecrets = daemonSet.Spec.Template.Spec.ImagePullSecrets
	params.tolerations = daemonSet.Spec.Template.Spec.Tolerations
	params.annotations = daemonSet.Spec.Template.Annotations

	return
}

// Repair repairs added drives
func (client *Client) Repair(ctx context.Context, args RepairArgs, log LogFunc) (results []RepairResult, err error) {
	if len(args.DriveIDs) == 0 {
		return
	}

	if log == nil {
		log = nullLogger
	}

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	params, err := client.getContainerParams(ctx)
	if err != nil {
		log(
			LogMessage{
				Type:             ErrorLogType,
				Err:              err,
				Message:          "unable to get container parameters from daemonset for drive repair",
				Values:           map[string]any{"namespace": consts.AppName, "daemonSet": consts.NodeServerName},
				FormattedMessage: fmt.Sprintf("unable to get container parameters from daemonset; %v\n", err),
			},
		)
		return nil, err
	}

	resultCh := client.NewDriveLister().
		DriveIDSelector(args.DriveIDs).
		IgnoreNotFound(true).
		List(ctx)
	for result := range resultCh {
		if result.Err != nil {
			return results, result.Err
		}

		jobName := "repair-" + result.Drive.Name
		if _, err := client.Kube().BatchV1().Jobs(consts.AppName).Get(ctx, jobName, metav1.GetOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				log(
					LogMessage{
						Type:             ErrorLogType,
						Err:              err,
						Message:          "unable to get repair job",
						Values:           map[string]any{"jobName": jobName},
						FormattedMessage: fmt.Sprintf("unable to get repair job %v; %v\n", jobName, err),
					},
				)
				continue
			}
		} else {
			log(
				LogMessage{
					Type:             ErrorLogType,
					Err:              err,
					Message:          "job already exists",
					Values:           map[string]any{"jobName": jobName},
					FormattedMessage: fmt.Sprintf("job %v already exists\n", jobName),
				},
			)
			continue
		}

		nodeID := string(result.Drive.GetNodeID())

		containerArgs := []string{"/directpv", "repair", result.Drive.Name, "--kube-node-name=" + nodeID}
		if args.ForceFlag {
			containerArgs = append(containerArgs, "--force")
		}
		if args.DisablePrefetchFlag {
			containerArgs = append(containerArgs, "--disable-prefetch")
		}
		if args.DryRun {
			containerArgs = append(containerArgs, "--dry-run")
		}

		job := batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:        jobName,
				Namespace:   consts.AppName,
				Annotations: params.annotations,
			},
			Spec: batchv1.JobSpec{
				BackoffLimit:            &backOffLimit,
				TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						NodeSelector:       map[string]string{string(directpvtypes.NodeLabelKey): nodeID},
						ServiceAccountName: consts.Identity,
						Tolerations:        params.tolerations,
						ImagePullSecrets:   params.imagePullSecrets,
						Volumes:            repairJobVolumes,
						Containers: []corev1.Container{
							{
								Name:                     jobName,
								Image:                    params.containerImage,
								Command:                  containerArgs,
								SecurityContext:          params.securityContext,
								VolumeMounts:             repairJobVolumeMounts,
								TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
								TerminationMessagePath:   "/var/log/repair-termination-log",
							},
						},
						RestartPolicy: corev1.RestartPolicyNever,
					},
				},
			},
		}

		if _, err := client.Kube().BatchV1().Jobs(consts.AppName).Create(ctx, &job, metav1.CreateOptions{}); err != nil {
			log(
				LogMessage{
					Type:             ErrorLogType,
					Err:              err,
					Message:          "unable to create repair job",
					Values:           map[string]any{"jobName": jobName, "driveName": result.Drive.Name},
					FormattedMessage: fmt.Sprintf("unable to create repair job %v; %v\n", jobName, err),
				},
			)
		} else {
			log(
				LogMessage{
					Type:             InfoLogType,
					Message:          "repair job created",
					Values:           map[string]any{"jobName": jobName, "driveName": result.Drive.Name},
					FormattedMessage: fmt.Sprintf("repair job %v for drive %v is created\n", jobName, result.Drive.Name),
				},
			)

			results = append(results, RepairResult{JobName: jobName, DriveName: result.Drive.GetDriveName(), DriveID: result.Drive.GetDriveID()})
		}
	}

	return
}
