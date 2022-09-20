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
	"fmt"
	"os"
	"path/filepath"
	directcsiclient "github.com/minio/directpv/pkg/client"
	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

const (
	MaxThreadCount = 200
)

func getKubeConfig() (*rest.Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	kubeConfig := filepath.Join(home, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		if config, err = rest.InClusterConfig(); err != nil {
			return nil, err
		}
	}
	config.QPS = float32(MaxThreadCount / 2)
	config.Burst = MaxThreadCount
	return config, nil
}

func main() {

	// Get the kubernetes configuration file
	kubeConfig, err := getKubeConfig()
	if err != nil {
		fmt.Printf("%s: Could not connect to kubernetes. %s=%s\n", "Error", "KUBECONFIG", kubeConfig)
		os.Exit(1)
	}

	// Get the latest interface available
	latestDirectCSIDriveInterface, err := directcsiclient.DirectCSIDriveInterfaceForConfig(kubeConfig)
	if err != nil {
		fmt.Printf("%s: could not initialize drive adapter client: err=%v\n", "Error", err)
		os.Exit(1)
	}

	// Then we list the drives
	// This is getting all the info from all the drives
	ctx, _ := context.WithCancel(context.Background())
	driveList, err := latestDirectCSIDriveInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Println("error")
	}

	// I want to print all info but human readable:
	filteredDrives := []directcsi.DirectCSIDrive{}
	for _, drive := range driveList.Items {
		filteredDrives = append(filteredDrives, drive)
	}
	headers := []interface{}{
		"DRIVE",
		"STATUS",
	}
	text.DisableColors()
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row(headers))
	style := table.StyleColoredDark
	style.Color.IndexColumn = text.Colors{text.FgHiBlue, text.BgHiBlack}
	style.Color.Header = text.Colors{text.FgHiBlue, text.BgHiBlack}
	t.SetStyle(style)

	for _, d := range filteredDrives {
		drive := d.Status.Path
		output := []interface{}{
			drive,
			d.Status.DriveStatus,
		}
		t.AppendRow(output)
	}
	t.Render()

	// TODO: I need to list only those drives that are InUse & Ready, the rest should be discarded.

	// TODO: We need to convert from directCSI to directPV the CRDs.

}
