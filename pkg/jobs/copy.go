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

package jobs

import (
	"context"
	"fmt"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CopyOpts defines the options for copying
type CopyOpts struct {
	SourceDriveID      directpvtypes.DriveID
	DestinationDriveID directpvtypes.DriveID
	VolumeID           string
	NodeID             directpvtypes.NodeID
}

// ContainerParams represents the container parameters
type ContainerParams struct {
	Image            string
	ImagePullSecrets []corev1.LocalObjectReference
	Tolerations      []corev1.Toleration
}

// CreateCopyJob creates a new job instance for copying the volume.
func CreateCopyJob(ctx context.Context, opts CopyOpts, params ContainerParams, overwrite bool) error {
	labels := map[string]string{
		string(directpvtypes.JobTypeLabelKey):          string(JobTypeCopy),
		string(directpvtypes.SourceDriveLabelKey):      string(opts.SourceDriveID),
		string(directpvtypes.DestinationDriveLabelKey): string(opts.DestinationDriveID),
		string(directpvtypes.NodeLabelKey):             string(opts.NodeID),
		string(directpvtypes.VolumeLabelKey):           opts.VolumeID,
	}
	for k, v := range defaultLabels {
		labels[k] = v
	}
	objectMeta := metav1.ObjectMeta{
		Name:      "copy-" + opts.VolumeID,
		Namespace: consts.AppNamespace,
		Labels:    labels,
		Finalizers: []string{
			consts.CopyProtectionFinalizer,
		},
	}
	privileged := true
	var backoffLimit int32 = 3
	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: objectMeta,
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector:       map[string]string{string(directpvtypes.NodeLabelKey): string(opts.NodeID)},
					ServiceAccountName: consts.Identity,
					Tolerations:        params.Tolerations,
					ImagePullSecrets:   params.ImagePullSecrets,
					Volumes: []corev1.Volume{
						k8s.NewHostPathVolume(consts.AppRootDirVolumeName, consts.AppRootDirVolumePath),
						k8s.NewHostPathVolume(consts.LegacyAppRootDirVolumeName, consts.LegacyAppRootDirVolumePath),
						k8s.NewHostPathVolume(consts.SysDirVolumeName, consts.SysDirVolumePath),
						k8s.NewHostPathVolume(consts.DevDirVolumeName, consts.DevDirVolumePath),
						k8s.NewHostPathVolume(consts.RunUdevDataVolumeName, consts.RunUdevDataVolumePath),
					},
					Containers: []corev1.Container{
						{
							Name:  "copy-job",
							Image: params.Image,
							Args: []string{
								"copy",
								string(opts.SourceDriveID),
								string(opts.DestinationDriveID),
								"--volume-id=" + opts.VolumeID,
								fmt.Sprintf("--kube-node-name=$(%s)", consts.KubeNodeNameEnvVarName),
							},
							Env: []corev1.EnvVar{
								{
									Name: consts.KubeNodeNameEnvVarName,
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "spec.nodeName",
										},
									},
								},
							},
							TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
							TerminationMessagePath:   "/var/log/copy-termination-log",
							VolumeMounts: []corev1.VolumeMount{
								k8s.NewVolumeMount(consts.AppRootDirVolumeName, consts.AppRootDirVolumePath, corev1.MountPropagationBidirectional, false),
								k8s.NewVolumeMount(consts.LegacyAppRootDirVolumeName, consts.LegacyAppRootDirVolumePath, corev1.MountPropagationBidirectional, false),
								k8s.NewVolumeMount(consts.SysDirVolumeName, consts.SysDirVolumePath, corev1.MountPropagationBidirectional, false),
								k8s.NewVolumeMount(consts.DevDirVolumeName, consts.DevDirVolumePath, corev1.MountPropagationHostToContainer, true),
								k8s.NewVolumeMount(consts.RunUdevDataVolumeName, consts.RunUdevDataVolumePath, corev1.MountPropagationBidirectional, true),
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged,
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	if _, err := k8s.KubeClient().BatchV1().Jobs(consts.AppNamespace).Create(ctx, job, metav1.CreateOptions{}); err != nil {
		if apierrors.IsAlreadyExists(err) && overwrite {
			return deleteAndCreate(ctx, job)
		}
		return err
	}
	return nil
}

func deleteAndCreate(ctx context.Context, job *batchv1.Job) error {
	if err := k8s.KubeClient().BatchV1().Jobs(consts.AppNamespace).Delete(ctx, job.Name, metav1.DeleteOptions{}); err != nil {
		return err
	}
	_, err := k8s.KubeClient().BatchV1().Jobs(consts.AppNamespace).Create(ctx, job, metav1.CreateOptions{})
	return err
}
