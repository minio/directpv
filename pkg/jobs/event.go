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
	"time"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/controller"
	"github.com/minio/directpv/pkg/k8s"
	v1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// JobStatus represents the status of the Job
type JobStatus string

// JobType represents the type of the Job
type JobType string

const (
	// JobTypeCopy represents the mirror job
	JobTypeCopy JobType = "copy"
	// JobTypeUnknown represents unknown job type
	JobTypeUnknown JobType = "unknown"
	// JobStatusActive represents the active job status
	JobStatusActive JobStatus = "active"
	// JobStatusFailed represents the failed job status
	JobStatusFailed JobStatus = "failed"
	// JobStatusSucceeded represents the succeeded job status
	JobStatusSucceeded JobStatus = "succeeded"
)

var defaultLabels = map[string]string{
	"application-name":                      consts.GroupName,
	"application-type":                      "CSIDriver",
	string(directpvtypes.CreatedByLabelKey): "controller",
	string(directpvtypes.VersionLabelKey):   consts.LatestAPIVersion,
}

const (
	workerThreads = 10
	resyncPeriod  = 10 * time.Minute
)

type jobsEventHandler struct{}

func newJobsEventHandler() *jobsEventHandler {
	return &jobsEventHandler{}
}

func (handler *jobsEventHandler) ListerWatcher() cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return k8s.KubeClient().BatchV1().Jobs(consts.AppNamespace).List(context.TODO(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return k8s.KubeClient().BatchV1().Jobs(consts.AppNamespace).Watch(context.TODO(), options)
		},
	}
}

func (handler *jobsEventHandler) ObjectType() runtime.Object {
	return &v1.Job{}
}

func (handler *jobsEventHandler) Handle(ctx context.Context, eventType controller.EventType, object runtime.Object) error {
	job := object.(*v1.Job)
	if !job.GetDeletionTimestamp().IsZero() {
		return handleDelete(ctx, job)
	}
	if eventType == controller.UpdateEvent {
		return handleUpdate(ctx, object.(*v1.Job))
	}
	return nil
}

func handleDelete(ctx context.Context, job *v1.Job) error {
	jobType, err := getJobType(job)
	if err != nil {
		return err
	}
	switch jobType {
	case JobTypeCopy:
		return handleCopyJobDeletion(ctx, job)
	default:
		return fmt.Errorf("Invalid jobType: %v", jobType)
	}
}

func handleCopyJobDeletion(ctx context.Context, job *v1.Job) error {
	if err := updateOnCopyJobCompletion(ctx, job); err != nil {
		return err
	}
	finalizers := []string{}
	for _, finalizer := range job.ObjectMeta.GetFinalizers() {
		if finalizer == consts.CopyProtectionFinalizer {
			continue
		}
		finalizers = append(finalizers, finalizer)
	}
	job.ObjectMeta.SetFinalizers(finalizers)
	_, err := k8s.KubeClient().BatchV1().Jobs(consts.AppNamespace).Update(ctx, job, metav1.UpdateOptions{})
	return err
}

func handleUpdate(ctx context.Context, job *v1.Job) error {
	if job.Status.CompletionTime == nil || job.Status.Succeeded == 0 {
		return nil
	}
	jobType, err := getJobType(job)
	if err != nil {
		return err
	}
	switch jobType {
	case JobTypeCopy:
		return updateOnCopyJobCompletion(ctx, job)
	default:
		return fmt.Errorf("Invalid jobType: %v", jobType)
	}
}

func getJobType(job *v1.Job) (JobType, error) {
	labels := job.ObjectMeta.GetLabels()
	if labels == nil {
		return JobTypeUnknown, fmt.Errorf("No labels present in the job: %v", job.Name)
	}
	value, ok := labels[string(directpvtypes.JobTypeLabelKey)]
	if !ok {
		return JobTypeUnknown, fmt.Errorf("Unable to identify the job: %v; Missing JobType", job.Name)
	}
	jobType, err := ToType(value)
	if err != nil {
		return JobTypeUnknown, err
	}
	return jobType, nil
}

func updateOnCopyJobCompletion(ctx context.Context, job *v1.Job) error {
	labels := job.ObjectMeta.GetLabels()

	// Update volume
	volumeName := labels[string(directpvtypes.VolumeLabelKey)]
	if volumeName == "" {
		return fmt.Errorf("No volumeID present in the copy job: %v", job.Name)
	}
	volume, err := client.VolumeClient().Get(ctx, volumeName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	volume.Status.Status = directpvtypes.VolumeStatusReady
	if !volume.IsStaged() {
		volume.Status.Status = directpvtypes.VolumeStatusPending
	}
	if _, err = client.VolumeClient().Update(ctx, volume, metav1.UpdateOptions{}); err != nil {
		return err
	}

	// Update source drive
	sourceDriveID := labels[string(directpvtypes.SourceDriveLabelKey)]
	if sourceDriveID == "" {
		return fmt.Errorf("No source drive ID present in the copy job: %v", job.Name)
	}
	sourceDrive, err := client.DriveClient().Get(ctx, sourceDriveID, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	sourceDrive.RemoveCopyProtectionFinalizer()
	_, err = client.DriveClient().Update(ctx, sourceDrive, metav1.UpdateOptions{})
	return err
}

// StartController starts volume controller.
func StartController(ctx context.Context) {
	ctrl := controller.New("jobs", newJobsEventHandler(), workerThreads, resyncPeriod)
	ctrl.Run(ctx)
}
