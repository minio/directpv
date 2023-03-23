// This file is part of MinIO DirectPV
// Copyright (c) 2021, 2022, 2023 MinIO, Inc.
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
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
	"github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/drive"
	"github.com/minio/directpv/pkg/initrequest"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/utils"
	"github.com/minio/directpv/pkg/volume"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type drainFunc func(ctx context.Context, nodeName string) error

var drainCmd = &cobra.Command{
	Use:           "drain [NODE ...]",
	Short:         "Drain the " + consts.AppPrettyName + " resources from the node(s)",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`1. Drain all the DirectPV resources from the node 'node1'
   $ kubectl {PLUGIN_NAME} drain node1`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		nodesArgs = args
		if err := validateDrainCmd(c.Context()); err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", err)
			os.Exit(-1)
		}
		input := getInput(color.HiRedString("Draining will forcefully remove all the " + consts.AppPrettyName + " resources from the specified node(s). Do you really want to continue (Yes|No)? "))
		if input != "Yes" {
			utils.Eprintf(quietFlag, false, "Aborting...\n")
			os.Exit(1)
		}
		drainMain(c.Context())
	},
}

func validateDrainCmd(ctx context.Context) error {
	if len(nodesArgs) == 0 {
		return errors.New("no node selected to drain, please check the syntax")
	}

	for _, node := range nodesArgs {
		csiNode, err := k8s.KubeClient().StorageV1().CSINodes().Get(ctx, node, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("unable to fetch the node %s; %v", node, err)
		}
		if err == nil {
			for _, driver := range csiNode.Spec.Drivers {
				if driver.Name == consts.Identity {
					return fmt.Errorf("unable to drain; the node '%s' is still under use by the '%s' CSI Driver", node, consts.Identity)
				}
			}
		}
	}

	return nil
}

func drainVolumes(ctx context.Context, nodeName string) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	for result := range volume.NewLister().NodeSelector([]types.LabelValue{types.ToLabelValue(nodeName)}).List(ctx) {
		if result.Err != nil {
			if apierrors.IsNotFound(result.Err) {
				continue
			}
			return result.Err
		}
		result.Volume.RemovePVProtection()
		result.Volume.RemovePurgeProtection()
		_, err := client.VolumeClient().Update(ctx, &result.Volume, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		err = client.VolumeClient().Delete(ctx, result.Volume.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func drainDrives(ctx context.Context, nodeName string) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	for result := range drive.NewLister().NodeSelector([]types.LabelValue{types.ToLabelValue(nodeName)}).List(ctx) {
		if result.Err != nil {
			if apierrors.IsNotFound(result.Err) {
				continue
			}
			return result.Err
		}
		result.Drive.Finalizers = []string{}
		_, err := client.DriveClient().Update(ctx, &result.Drive, metav1.UpdateOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		err = client.DriveClient().Delete(ctx, result.Drive.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func drainInitRequests(ctx context.Context, nodeName string) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	for result := range initrequest.NewLister().NodeSelector([]types.LabelValue{types.ToLabelValue(nodeName)}).List(ctx) {
		if result.Err != nil {
			if apierrors.IsNotFound(result.Err) {
				continue
			}
			return result.Err
		}
		result.InitRequest.Finalizers = []string{}
		_, err := client.InitRequestClient().Update(ctx, &result.InitRequest, metav1.UpdateOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		err = client.InitRequestClient().Delete(ctx, result.InitRequest.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func deleteNode(ctx context.Context, nodeName string) error {
	err := client.NodeClient().Delete(ctx, nodeName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func startDraining(ctx context.Context, nodes []string, teaProgram *tea.Program) (err error) {
	var completedTasks int
	drainFuncs := map[string]drainFunc{
		"volumes":      drainVolumes,
		"drives":       drainDrives,
		"initrequests": drainInitRequests,
	}
	totalTasks := len(nodes) * len(drainFuncs)
	drainProgressMap := make(map[string]progressLog, totalTasks)
	for _, node := range nodes {
		for resource, drainFn := range drainFuncs {
			if teaProgram != nil {
				drainProgressMap[resource+node] = progressLog{
					log: fmt.Sprintf("Draining %s from the node '%s'", resource, node),
				}
				teaProgram.Send(progressNotification{
					progressLogs: toProgressLogs(drainProgressMap),
					percent:      float64(completedTasks) / float64(totalTasks),
				})
			}
			if err := drainFn(ctx, node); err != nil {
				return err
			}
			if teaProgram != nil {
				completedTasks++
				drainProgressMap[resource+node] = progressLog{
					log:  fmt.Sprintf("Drained %s from the node '%v'", resource, node),
					done: true,
				}
				teaProgram.Send(progressNotification{
					progressLogs: toProgressLogs(drainProgressMap),
					percent:      float64(completedTasks) / float64(totalTasks),
				})
			}
		}
		if err := deleteNode(ctx, node); err != nil {
			return err
		}
	}
	return
}

func drainMain(ctx context.Context) {
	var teaProgram *tea.Program
	var wg sync.WaitGroup
	if !quietFlag {
		m := newProgressModel(true)
		teaProgram = tea.NewProgram(m)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := teaProgram.Run(); err != nil {
				fmt.Println("error running program:", err)
				os.Exit(1)
			}
		}()
	}
	err := startDraining(ctx, nodesArgs, teaProgram)
	if err != nil && quietFlag {
		utils.Eprintf(quietFlag, true, "%v\n", err)
		os.Exit(1)
	}
	if teaProgram != nil {
		teaProgram.Send(progressNotification{
			log: func() string {
				if err == nil {
					return "successfully drained the node(s)"
				}
				return ""
			}(),
			done: true,
			err:  err,
		})
		wg.Wait()
	}
}
