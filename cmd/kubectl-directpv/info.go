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

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/drive"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/volume"
	"github.com/spf13/cobra"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
)

var errInstallationNotFound = errors.New("installation not found")

var infoCmd = &cobra.Command{
	Use:           "info",
	Short:         "Show information about " + consts.AppPrettyName + " installation.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		return getInfo(c.Context(), args)
	},
}

func getInfo(ctx context.Context, args []string) error {
	crds, err := k8s.CRDClient().List(ctx, metav1.ListOptions{})
	if err != nil {
		if !quiet {
			klog.ErrorS(err, "unable to list CRDs")
		}
		return err
	}

	drivesFound := false
	volumesFound := false
	for _, crd := range crds.Items {
		if strings.Contains(crd.Name, consts.DriveResource+"."+consts.GroupName) {
			drivesFound = true
		}
		if strings.Contains(crd.Name, consts.VolumeResource+"."+consts.GroupName) {
			volumesFound = true
		}
	}
	if !drivesFound || !volumesFound {
		return fmt.Errorf(consts.AppPrettyName + " installation not found")
	}

	storageClient, gvk, err := k8s.GetClientForNonCoreGroupVersionKind("storage.k8s.io", "CSINode", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return err
	}

	nodeList := []string{}
	switch gvk.Version {
	case "v1":
		result := &storagev1.CSINodeList{}
		if err := storageClient.Get().
			Resource("csinodes").
			VersionedParams(&metav1.ListOptions{}, scheme.ParameterCodec).
			Timeout(10 * time.Second).
			Do(ctx).
			Into(result); err != nil {
			if !quiet {
				klog.ErrorS(err, "unable to get CSI nodes")
			}
			return err
		}
		for _, csiNode := range result.Items {
			for _, driver := range csiNode.Spec.Drivers {
				if driver.Name == consts.Identity {
					nodeList = append(nodeList, csiNode.Name)
					break
				}
			}
		}
	case "v1beta1":
		result := &storagev1beta1.CSINodeList{}
		if err := storageClient.Get().
			Resource(gvk.Kind).
			VersionedParams(&metav1.ListOptions{}, scheme.ParameterCodec).
			Timeout(10 * time.Second).
			Do(ctx).
			Into(result); err != nil {
			if !quiet {
				klog.ErrorS(err, "unable to get CSI nodes")
			}
			return err
		}
		for _, csiNode := range result.Items {
			for _, driver := range csiNode.Spec.Drivers {
				if driver.Name == consts.Identity {
					nodeList = append(nodeList, csiNode.Name)
					break
				}
			}
		}
	case "v1apha1":
		return errors.New("storage.k8s.io/v1alpha1 is not supported")
	}

	if len(nodeList) == 0 {
		return errInstallationNotFound
	}

	drives, err := drive.GetDriveList(ctx, nil, nil, nil)
	if err != nil {
		if !quiet {
			klog.ErrorS(err, "unable to get drive list")
		}
		return err
	}

	volumes, err := volume.GetVolumeList(ctx, nil, nil, nil, nil)
	if err != nil {
		if !quiet {
			klog.ErrorS(err, "unable to get volume list")
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
				if k8s.IsConditionStatus(v.Status.Conditions, string(types.VolumeConditionTypeReady), metav1.ConditionTrue) {
					volumeCount++
					volumeSize += uint64(v.Status.TotalCapacity)
				}
			}
		}
		totalVolumeSize += volumeSize

		if driveCount == 0 {
			t.AppendRow([]interface{}{
				fmt.Sprintf("%s %s", color.YellowString(dot), n),
				"-",
				"-",
				"-",
				"-",
			})
		} else {
			t.AppendRow([]interface{}{
				fmt.Sprintf("%s %s", color.GreenString(dot), n),
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
			fmt.Printf(
				"\n%s/%s used, %s volumes, %s drives\n",
				humanize.IBytes(totalVolumeSize),
				humanize.IBytes(totalDriveSize),
				color.HiWhiteString("%d", len(volumes)),
				color.HiWhiteString("%d", len(drives)),
			)
		}
	}

	return nil
}
