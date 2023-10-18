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
	"fmt"
	"os"
	"strings"

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/jobs"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var purgeJobsCmd = &cobra.Command{
	Use:           "jobs [JOB ...]",
	Short:         "Purge jobs",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`1. Purge all jobs
   $ kubectl {PLUGIN_NAME} purge jobs --all

2. Purge jobs from a node
   $ kubectl {PLUGIN_NAME} purge jobs --nodes=node1

3. Purge jobs by type
   $ kubectl {PLUGIN_NAME} purge jobs --type=copy

3. Purge jobs filtered by labels
   $ kubectl {PLUGIN_NAME} purge jobs --labels type=copy`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		jobNameArgs = args
		if err := validatePurgeJobsArgs(); err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", err)
			os.Exit(-1)
		}

		purgeJobsMain(c.Context())
	},
}

func init() {
	setFlagOpts(purgeJobsCmd)

	addJobsTypeFlag(purgeJobsCmd, "Filter output by job type")
	addJobsStatusFlag(purgeJobsCmd, "Filter output by job status")
	addLabelsFlag(purgeJobsCmd, "Filter output by job labels")
	addDangerousFlag(purgeJobsCmd, "Set dangerous flag to forcefully purge active jobs")
}

func validatePurgeJobsArgs() error {
	if err := validateListJobsArgs(); err != nil {
		return err
	}
	if err := validateLabelArgs(); err != nil {
		return err
	}
	switch {
	case allFlag:
	case len(nodesArgs) != 0:
	case len(jobNameArgs) != 0:
	case len(jobStatusArgs) != 0:
	case len(jobTypeArgs) != 0:
	case len(labelArgs) != 0:
	default:
		return errors.New("no jobs selected to purge")
	}
	if allFlag {
		nodesArgs = nil
		jobNameArgs = nil
		jobStatusSelectors = nil
		jobTypeSelectors = nil
		labelSelectors = nil
	}
	return nil
}

func purgeJobsMain(ctx context.Context) {
	resultCh := jobs.NewLister().
		JobNameSelector(jobNameArgs).
		NodeSelector(toLabelValues(nodesArgs)).
		StatusSelector(jobStatusSelectors).
		TypeSelector(jobTypeSelectors).
		LabelSelector(labelSelectors).
		List(ctx)
	for result := range resultCh {
		if result.Err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", result.Err)
			os.Exit(1)
		}
		switch jobs.GetStatus(result.Job) {
		case jobs.JobStatusActive:
			if !dangerousFlag {
				utils.Eprintf(quietFlag, true, "Purging the active job may lead to partial data; Please use `--dangerous` to purge the job %v", result.Job.Name)
				continue
			}
		case jobs.JobStatusSucceeded, jobs.JobStatusFailed:
		}
		if !dryRunFlag {
			if err := k8s.KubeClient().BatchV1().Jobs(consts.AppNamespace).Delete(ctx, result.Job.Name, metav1.DeleteOptions{}); err != nil {
				utils.Eprintf(quietFlag, true, "unable to delete job %v: %v\n", result.Job.Name, err)
			}
		}
		if !quietFlag {
			fmt.Printf("Job '%s' purged successfully \n", result.Job.Name)
		}
	}
}
