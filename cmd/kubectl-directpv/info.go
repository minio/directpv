// This file is part of MinIO DirectPV
// Copyright (c) 2021, 2022 MinIO, Inc.
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
	"time"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/utils"

	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var infoCmd = &cobra.Command{
	Use:           "info",
	Short:         utils.BinaryNameTransform("Info about {{ . }} installation"),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		return getInfo(c.Context(), args, false)
	},
}

func getInfo(ctx context.Context, args []string, quiet bool) error {
	crdclient := client.GetCRDClient()

	crds, err := crdclient.List(ctx, metav1.ListOptions{})
	if err != nil {
		if !quiet {
			klog.Errorf("error listing crds: %v", err)
		}
		return err
	}

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
		formatter := utils.BinaryNameTransform("%s: {{ . }} installation not found")
		if !quiet {
			return fmt.Errorf(formatter, bold("Error"))
		}
		return fmt.Errorf(formatter, bold("Error"))
	}

	cln, gvk, err := client.GetClientForNonCoreGroupKindVersions("storage.k8s.io", "CSINode", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return err
	}

	nodeList := []string{}

	if gvk.Version == "v1" {
		result := &storagev1.CSINodeList{}
		if err := cln.Get().
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
		if err := cln.Get().
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
		return errors.New(utils.BinaryNameTransform("this version of CSINode is not supported by {{ . }}"))
	}

	if len(nodeList) == 0 {
		if !quiet {
			fmt.Printf(utils.BinaryNameTransform("%s: {{ . }} installation %s found\n\n"),
				red(bold("ERR")), "NOT")
			fmt.Printf("run '%s' to get started\n",
				bold(utils.BinaryNameTransform("kubectl {{ . }} install")))
		}
		return fmt.Errorf(utils.BinaryNameTransform("{{ . }} installation not found"))
	}

	drives, err := getFilteredDriveList(
		ctx,
		func(drive directcsi.DirectCSIDrive) bool {
			return drive.Status.DriveStatus == directcsi.DriveStatusInUse || drive.Status.DriveStatus == directcsi.DriveStatusReady
		},
	)
	if err != nil {
		if !quiet {
			klog.Errorf("error getting drive list: %v", err)
		}
		return err
	}

	volumes, err := client.GetVolumeList(ctx, nil, nil, nil, nil)
	if err != nil {
		if !quiet {
			klog.Errorf("error getting volume list: %v", err)
		}
		return err
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"NODE", "CAPACITY", "ALLOCATED", "VOLUMES", "DRIVES"})

	var totalDriveSize uint64
	var totalVolumeSize uint64
	for _, n := range nodeList {
		driveCount := 0
		driveSize := uint64(0)
		for _, d := range drives {
			if d.Status.NodeName == n {
				driveCount++
				driveSize += uint64(d.Status.TotalCapacity)
			}
		}
		totalDriveSize += driveSize

		volumeCount := 0
		volumeSize := uint64(0)
		for _, v := range volumes {
			if v.Status.NodeName == n {
				if utils.IsConditionStatus(v.Status.Conditions, string(directcsi.DirectCSIVolumeConditionReady), metav1.ConditionTrue) {
					volumeCount++
					volumeSize += uint64(v.Status.TotalCapacity)
				}
			}
		}
		totalVolumeSize += volumeSize

		if driveCount == 0 {
			t.AppendRow([]interface{}{
				fmt.Sprintf("%s %s", red(dot), n),
				"-",
				"-",
				"-",
				"-",
			})
		} else {
			t.AppendRow([]interface{}{
				fmt.Sprintf("%s %s", green(dot), n),
				humanize.IBytes(driveSize),
				humanize.IBytes(volumeSize),
				fmt.Sprintf("%d", volumeCount),
				fmt.Sprintf("%d", driveCount),
			})
		}
	}

	text.DisableColors()

	style := table.StyleColoredDark
	style.Color.IndexColumn = text.Colors{text.FgHiBlue, text.BgHiBlack}
	style.Color.Header = text.Colors{text.FgHiBlue, text.BgHiBlack}
	t.SetStyle(style)
	if !quiet {
		t.Render()
		if len(drives) > 0 {
			fmt.Println()
			fmt.Printf("%s/%s used, %s volumes, %s drives\n",
				humanize.IBytes(totalVolumeSize),
				humanize.IBytes(totalDriveSize),
				bold(fmt.Sprintf("%d", len(volumes))),
				bold(fmt.Sprintf("%d", len(drives))))
		}
	}

	return nil
}
