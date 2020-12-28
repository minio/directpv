// This file is part of MinIO Direct CSI
// Copyright (c) 2020 MinIO, Inc.
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
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"github.com/spf13/cobra"

	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/minio/direct-csi/pkg/utils"
)

var infoCmd = &cobra.Command{
	Use:           "info",
	Short:         "Info about direct-csi installation",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		return info(c.Context(), args, false)
	},
}

func info(ctx context.Context, args []string, quiet bool) error {
	utils.Init()

	bold := color.New(color.Bold).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	client, gvk, err := utils.GetClientForNonCoreGroupKindVersions("storage.k8s.io", "CSINode", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return err
	}

	nodeList := []string{}

	if gvk.Version == "v1" {
		result := &storagev1.CSINodeList{}
		if err := client.Get().
			Resource("csinodes").
			VersionedParams(&metav1.ListOptions{}, scheme.ParameterCodec).
			Timeout(10 * time.Second).
			Do(ctx).
			Into(result); err != nil {
			return err
		}
		for _, csiNode := range result.Items {
			for _, driver := range csiNode.Spec.Drivers {
				if driver.Name == identity {
					nodeList = append(nodeList, csiNode.Name)
					break
				}
			}
		}
	}
	if gvk.Version == "v1beta1" {
		result := &storagev1beta1.CSINodeList{}
		if err := client.Get().
			Resource(gvk.Kind).
			VersionedParams(&metav1.ListOptions{}, scheme.ParameterCodec).
			Timeout(10 * time.Second).
			Do(ctx).
			Into(result); err != nil {
			return err
		}
		for _, csiNode := range result.Items {
			for _, driver := range csiNode.Spec.Drivers {
				if driver.Name == identity {
					nodeList = append(nodeList, csiNode.Name)
					break
				}
			}
		}
	}

	if gvk.Version == "v1alpha1" {
		return utils.ErrKubeVersionNotSupported
	}

	if len(nodeList) == 0 {
		if !quiet {
			fmt.Printf("%s: DirectCSI installation %s found\n", red(bold("ERR")), "NOT")
			fmt.Println()
			fmt.Printf("run '%s' to get started\n", bold("kubectl direct-csi install"))
		}
		return fmt.Errorf("DirectCSI installation not found")
	}

	directCSIClient := utils.GetDirectCSIClient()
	drives, err := directCSIClient.DirectCSIDrives().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("could not list all drives: %v", err)
	}

	volumes, err := directCSIClient.DirectCSIVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("could not list all drives: %v", err)
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"#", "NodeName", "", "#Drives", "", "#Volumes"})

	totalOwnedDrives := 0
	totalDrives := len(drives.Items)
	totalVolumes := len(volumes.Items)
	for _, d := range drives.Items {
		if d.Spec.DirectCSIOwned {
			totalOwnedDrives++
		}
	}
	for i, n := range nodeList {
		driveList := []string{}
		numDrives := 0
		for _, d := range drives.Items {
			if d.Status.NodeName == n {
				numDrives++
				if d.Spec.DirectCSIOwned {
					driveList = append(driveList, d.Name)
				}
			}
		}
		numVols := 0
		for _, v := range volumes.Items {
			if v.Status.OwnerNode == n {
				numVols++
			}
		}
		t.AppendRow([]interface{}{
			fmt.Sprintf("%d", i+1),
			n,
			"",
			fmt.Sprintf("(%d/%d)", len(driveList), numDrives),
			"",
			fmt.Sprintf(" %d", numVols),
		})
	}

	style := table.StyleColoredDark
	style.Color.IndexColumn = text.Colors{text.FgHiBlue, text.BgHiBlack}
	style.Color.Header = text.Colors{text.FgHiBlue, text.BgHiBlack}
	t.SetStyle(style)
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, Align: text.AlignJustify},
		{Number: 2, Align: text.AlignJustify},
		{Number: 3, Align: text.AlignJustify},
		{Number: 4, Align: text.AlignJustify},
		{Number: 5, Align: text.AlignJustify},
	})
	if !quiet {
		t.Render()
		fmt.Println()
		fmt.Printf("(%s/%s) Drives managed by direct-csi, %s Total Volumes\n", bold(fmt.Sprintf("%d", totalOwnedDrives)), bold(fmt.Sprintf("%d", totalDrives)), bold(fmt.Sprintf("%d", totalVolumes)))
	}

	return nil
}
