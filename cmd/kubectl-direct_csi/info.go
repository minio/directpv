// This file is part of MinIO Direct CSI
// Copyright (c) 2021 MinIO, Inc.
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
	"strings"
	"time"

	"github.com/minio/direct-csi/pkg/installer"
	"github.com/minio/direct-csi/pkg/utils"

	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"github.com/spf13/cobra"
	"k8s.io/klog"
)

var infoCmd = &cobra.Command{
	Use:           "info",
	Short:         "Info about direct-csi installation",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		return getInfo(c.Context(), args, false)
	},
}

func getInfo(ctx context.Context, args []string, quiet bool) error {
	crdclient := utils.GetCRDClient()

	if crds, err := crdclient.List(ctx, metav1.ListOptions{}); err != nil {
		if !quiet {
			klog.Errorf("error listing crds: %v", err)
		}
		return err
	} else {
		drivesFound := false
		volumesFound := false
		for _, crd := range crds.Items {
			if strings.Contains(crd.Name, "directcsidrives.direct.csi.min.io") {
				drivesFound = true
			}
			if strings.Contains(crd.Name, "directcsivolumes.direct.csi.min.io") {
				volumesFound = true
			}
		}
		if !(drivesFound && volumesFound) {
			if !quiet {
				return fmt.Errorf("%s: DirectCSI installation not found", bold("Error"))
			}
			return fmt.Errorf("%s: DirectCSI installation not found", bold("Error"))
		}
	}

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
			if !quiet {
				klog.Errorf("error getting csinodes: %v", err)
			}
			return err
		}
		for _, csiNode := range result.Items {
			for _, driver := range csiNode.Spec.Drivers {
				if driver.Name == strings.ReplaceAll(identity, ".", "-") {
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
			if !quiet {
				klog.Errorf("error getting storagev1beta1/csinodes: %v", err)
			}
			return err
		}
		for _, csiNode := range result.Items {
			for _, driver := range csiNode.Spec.Drivers {
				if driver.Name == strings.ReplaceAll(identity, ".", "-") {
					nodeList = append(nodeList, csiNode.Name)
					break
				}
			}
		}
	}

	if gvk.Version == "v1alpha1" {
		return installer.ErrKubeVersionNotSupported
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
		if !quiet {
			klog.Errorf("error getting drive list: %v", err)
		}
		return err
	}

	volumes, err := directCSIClient.DirectCSIVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		if !quiet {
			klog.Errorf("error getting volume list: %v", err)
		}
		return err
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"NODE", "CAPACITY", "ALLOCATED", "VOLUMES", "DRIVES"})

	var totalSize int64
	var allocatedSize int64
	totalOwnedDrives := 0
	totalVolumes := len(volumes.Items)
	for _, d := range drives.Items {
		if d.Spec.DirectCSIOwned {
			totalOwnedDrives++
			totalSize = totalSize + d.Status.TotalCapacity
		}
	}
	for _, n := range nodeList {
		driveList := []string{}
		numDrives := 0
		var nodeVolSize int64
		var nodeDriveSize int64
		status := red(dot)
		for _, d := range drives.Items {
			if d.Status.NodeName == n {
				numDrives++
				if d.Spec.DirectCSIOwned {
					status = green(dot)
					driveList = append(driveList, d.Name)
					nodeDriveSize = nodeDriveSize + d.Status.TotalCapacity
				}
			}
		}
		numVols := 0
		for _, v := range volumes.Items {
			if v.Status.NodeName == n {
				numVols++
				allocatedSize = allocatedSize + v.Status.TotalCapacity
				nodeVolSize = nodeVolSize + v.Status.TotalCapacity
			}
		}
		if len(driveList) == 0 {
			t.AppendRow([]interface{}{
				fmt.Sprintf("%s %s", status, n),
				"-",
				"-",
				"-",
				"-",
			})
			continue
		}
		t.AppendRow([]interface{}{
			fmt.Sprintf("%s %s", status, n),
			fmt.Sprintf("%s", humanize.IBytes(uint64(nodeDriveSize))),
			fmt.Sprintf("%s", humanize.IBytes(uint64(nodeVolSize))),
			fmt.Sprintf("%d", numVols),
			fmt.Sprintf("%d", len(driveList)),
		})
	}

	text.DisableColors()

	style := table.StyleColoredDark
	style.Color.IndexColumn = text.Colors{text.FgHiBlue, text.BgHiBlack}
	style.Color.Header = text.Colors{text.FgHiBlue, text.BgHiBlack}
	t.SetStyle(style)
	if !quiet {
		t.Render()
		if totalOwnedDrives > 0 {
			fmt.Println()
			fmt.Printf("%s/%s used, %s volumes, %s drives\n",
				bold(fmt.Sprintf("%s", humanize.IBytes(uint64(allocatedSize)))),
				bold(fmt.Sprintf("%s", humanize.IBytes(uint64(totalSize)))),
				bold(fmt.Sprintf("%d", totalVolumes)),
				bold(fmt.Sprintf("%d", totalOwnedDrives)))
		}
	}

	return nil
}
