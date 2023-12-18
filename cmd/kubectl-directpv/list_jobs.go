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
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/jobs"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var jobNameArgs []string

var listJobsCmd = &cobra.Command{
	Use:           "jobs [JOB ...]",
	Short:         "List jobs",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`1. List all jobs
   $ kubectl {PLUGIN_NAME} list jobs

2. List jobs by a node
   $ kubectl {PLUGIN_NAME} list jobs --nodes=node1

3. List jobs by type
   $ kubectl {PLUGIN_NAME} list jobs --type=copy

3. List jobs filtered by labels
   $ kubectl {PLUGIN_NAME} list jobs --labels type=copy`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		jobNameArgs = args
		if err := validateListJobsArgs(); err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", err)
			os.Exit(-1)
		}

		listJobsMain(c.Context())
	},
}

func init() {
	setFlagOpts(listJobsCmd)

	addJobsTypeFlag(listJobsCmd, "Filter output by job type")
	addJobsStatusFlag(listJobsCmd, "Filter output by job status")
	addShowLabelsFlag(listJobsCmd)
	addLabelsFlag(listJobsCmd, "Filter output by job labels")
}

func validateListJobsArgs() error {
	if err := validateJobNameArgs(); err != nil {
		return err
	}

	if err := validateJobStatusArgs(); err != nil {
		return err
	}

	return validateJobTypeArgs()
}

func listJobsMain(ctx context.Context) {
	jobObjects, err := jobs.NewLister().
		JobNameSelector(jobNameArgs).
		NodeSelector(toLabelValues(nodesArgs)).
		StatusSelector(jobStatusSelectors).
		TypeSelector(jobTypeSelectors).
		LabelSelector(labelSelectors).
		Get(ctx)
	if err != nil {
		utils.Eprintf(quietFlag, true, "%v\n", err)
		os.Exit(1)
	}

	if dryRunPrinter != nil {
		jobList := batchv1.JobList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "List",
				APIVersion: "v1",
			},
			Items: jobObjects,
		}
		dryRunPrinter(jobList)
		return
	}

	headers := table.Row{
		"JOB",
		"TYPE",
		"NODE",
		"STATUS",
	}
	if showLabels {
		headers = append(headers, "LABELS")
	}
	writer := newTableWriter(
		headers,
		[]table.SortBy{
			{
				Name: "JOB",
				Mode: table.Asc,
			},
			{
				Name: "NODE",
				Mode: table.Asc,
			},
			{
				Name: "STATUS",
				Mode: table.Asc,
			},
			{
				Name: "TYPE",
				Mode: table.Asc,
			},
		},
		noHeaders)

	for _, job := range jobObjects {
		row := []interface{}{
			job.Name,
			jobs.GetType(job),
			printableString(jobs.GetNode(job)),
			jobs.GetStatus(job),
		}
		if showLabels {
			row = append(row, labelsToString(job.GetLabels()))
		}
		writer.AppendRow(row)
	}

	if writer.Length() > 0 {
		writer.Render()
		return
	}

	if allFlag {
		utils.Eprintf(quietFlag, false, "No resources found\n")
	} else {
		utils.Eprintf(quietFlag, false, "No matching resources found\n")
	}

	os.Exit(1)
}
