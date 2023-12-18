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
	"strings"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/utils"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListJobResult denotes list of job result.
type ListJobResult struct {
	Job batchv1.Job
	Err error
}

// Lister is job lister.
type Lister struct {
	nodes          []directpvtypes.LabelValue
	statusList     []JobStatus
	typeList       []JobType
	jobNames       []string
	labels         map[directpvtypes.LabelKey]directpvtypes.LabelValue
	maxObjects     int64
	ignoreNotFound bool
}

// NewLister creates new job lister.
func NewLister() *Lister {
	return &Lister{
		maxObjects: k8s.MaxThreadCount,
	}
}

// ToStatus converts a value to job status.
func ToStatus(value string) (JobStatus, error) {
	status := JobStatus(strings.ToLower(value))
	switch status {
	case JobStatusActive, JobStatusFailed, JobStatusSucceeded:
	default:
		return status, fmt.Errorf("invalid job status: %v", value)
	}
	return status, nil
}

// ToType converts a value to job type.
func ToType(value string) (JobType, error) {
	status := JobType(strings.ToLower(value))
	switch status {
	case JobTypeCopy:
	default:
		return status, fmt.Errorf("invalid job type: %v", value)
	}
	return status, nil
}

// NodeSelector adds filter listing by nodes.
func (lister *Lister) NodeSelector(nodes []directpvtypes.LabelValue) *Lister {
	lister.nodes = nodes
	return lister
}

// StatusSelector adds filter listing by job status.
func (lister *Lister) StatusSelector(statusList []JobStatus) *Lister {
	lister.statusList = statusList
	return lister
}

// TypeSelector adds filter listing by job status.
func (lister *Lister) TypeSelector(typeList []JobType) *Lister {
	lister.typeList = typeList
	return lister
}

// JobNameSelector adds filter listing by job names.
func (lister *Lister) JobNameSelector(jobNames []string) *Lister {
	lister.jobNames = jobNames
	return lister
}

// LabelSelector adds filter listing by labels.
func (lister *Lister) LabelSelector(labels map[directpvtypes.LabelKey]directpvtypes.LabelValue) *Lister {
	lister.labels = labels
	return lister
}

// MaxObjects controls number of items to be fetched in every iteration.
func (lister *Lister) MaxObjects(n int64) *Lister {
	lister.maxObjects = n
	return lister
}

// IgnoreNotFound controls listing to ignore job not found error.
func (lister *Lister) IgnoreNotFound(b bool) *Lister {
	lister.ignoreNotFound = b
	return lister
}

// GetStatus gets the job status.
func GetStatus(job batchv1.Job) JobStatus {
	if job.Status.Active > 0 {
		return JobStatusActive
	}
	if job.Status.CompletionTime != nil && job.Status.Succeeded > 0 {
		return JobStatusSucceeded
	}
	return JobStatusFailed
}

// GetType gets the job type
func GetType(job batchv1.Job) JobType {
	labels := job.GetLabels()
	if v, ok := labels[string(directpvtypes.JobTypeLabelKey)]; ok {
		jobType, err := ToType(v)
		if err == nil {
			return jobType
		}
	}
	return JobTypeUnknown
}

// GetNode returns the node name of the job
func GetNode(job batchv1.Job) string {
	labels := job.GetLabels()
	if v, ok := labels[string(directpvtypes.NodeLabelKey)]; ok {
		return v
	}
	return ""
}

// List returns channel to loop through job items.
func (lister *Lister) List(ctx context.Context) <-chan ListJobResult {
	getOnly := len(lister.nodes) == 0 &&
		len(lister.statusList) == 0 &&
		len(lister.labels) == 0 &&
		len(lister.typeList) == 0 &&
		len(lister.jobNames) != 0

	labelMap := map[directpvtypes.LabelKey][]directpvtypes.LabelValue{
		directpvtypes.NodeLabelKey: lister.nodes,
	}
	for k, v := range lister.labels {
		labelMap[k] = []directpvtypes.LabelValue{v}
	}
	labelSelector := directpvtypes.ToLabelSelector(labelMap)

	resultCh := make(chan ListJobResult)
	go func() {
		defer close(resultCh)

		send := func(result ListJobResult) bool {
			select {
			case <-ctx.Done():
				return false
			case resultCh <- result:
				return true
			}
		}

		if !getOnly {
			options := metav1.ListOptions{
				Limit:         lister.maxObjects,
				LabelSelector: labelSelector,
			}

			for {
				result, err := k8s.KubeClient().BatchV1().Jobs(consts.AppNamespace).List(ctx, options)
				if err != nil {
					if apierrors.IsNotFound(err) && lister.ignoreNotFound {
						break
					}

					send(ListJobResult{Err: err})
					return
				}

				for _, item := range result.Items {
					var found bool
					var values []string
					for i := range lister.jobNames {
						if lister.jobNames[i] == item.Name {
							found = true
						} else {
							values = append(values, lister.jobNames[i])
						}
					}
					lister.jobNames = values

					switch {
					case found || (len(lister.statusList) == 0 && len(lister.typeList) == 0):
					case len(lister.statusList) > 0 && utils.Contains(lister.statusList, GetStatus(item)):
					case len(lister.typeList) > 0 && utils.Contains(lister.typeList, GetType(item)):
					default:
						continue
					}

					if !send(ListJobResult{Job: item}) {
						return
					}
				}

				if result.Continue == "" {
					break
				}

				options.Continue = result.Continue
			}
		}

		for _, jobName := range lister.jobNames {
			job, err := k8s.KubeClient().BatchV1().Jobs(consts.AppNamespace).Get(ctx, jobName, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) && lister.ignoreNotFound {
					continue
				}
				send(ListJobResult{Err: err})
				return
			}
			if !send(ListJobResult{Job: *job}) {
				return
			}
		}
	}()

	return resultCh
}

// Get returns list of jobs.
func (lister *Lister) Get(ctx context.Context) ([]batchv1.Job, error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	jobList := []batchv1.Job{}
	for result := range lister.List(ctx) {
		if result.Err != nil {
			return jobList, result.Err
		}
		jobList = append(jobList, result.Job)
	}

	return jobList, nil
}
