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
	"path"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/table"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/utils"
	"github.com/mitchellh/go-homedir"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
)

const dot = "•"

func printYAML(obj interface{}) {
	data, err := utils.ToYAML(obj)
	if err != nil {
		klog.Fatalf("unable to marshal object to YAML; %v", err)
	}

	fmt.Print(string(data))
}

func printJSON(obj interface{}) {
	data, err := utils.ToJSON(obj)
	if err != nil {
		klog.Fatalf("unable to marshal object to JSON; %v", err)
	}

	fmt.Print(string(data))
}

func getDefaultAuditDir() (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return path.Join(homeDir, "."+consts.AppName, "audit"), nil
}

func openAuditFile(auditFile string) (*utils.SafeFile, error) {
	defaultAuditDir, err := getDefaultAuditDir()
	if err != nil {
		return nil, fmt.Errorf("unable to get default audit directory; %w", err)
	}
	if err := os.MkdirAll(defaultAuditDir, 0o700); err != nil {
		return nil, fmt.Errorf("unable to create default audit directory; %w", err)
	}
	return utils.NewSafeFile(path.Join(defaultAuditDir, auditFile))
}

func printableString(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func printableBytes(value int64) string {
	if value == 0 {
		return "-"
	}

	return humanize.IBytes(uint64(value))
}

func newTableWriter(header table.Row, sortBy []table.SortBy, noHeader bool) table.Writer {
	writer := table.NewWriter()
	writer.SetOutputMirror(os.Stdout)
	writer.AppendHeader(header)
	writer.SortBy(sortBy)
	if noHeader {
		writer.ResetHeaders()
	}

	style := table.StyleLight
	writer.SetStyle(style)

	return writer
}

func toLabelValues(slice []string) (values []directpvtypes.LabelValue) {
	for _, s := range slice {
		values = append(values, directpvtypes.ToLabelValue(s))
	}
	return
}

func validateOutputFormat(isWideSupported bool) error {
	switch outputFormat {
	case "":
	case "wide":
		if !isWideSupported {
			return errors.New("wide option is not supported by this command")
		}
		wideOutput = true
	case "yaml":
		dryRunPrinter = printYAML
	case "json":
		dryRunPrinter = printJSON
	default:
		if isWideSupported {
			return errors.New("--output flag value must be one of wide|json|yaml or empty")
		}
		return errors.New("--output flag value must be one of yaml|json")
	}
	return nil
}

func getCSINodes(ctx context.Context) (nodes []string, err error) {
	storageClient, gvk, err := k8s.GetClientForNonCoreGroupVersionKind("storage.k8s.io", "CSINode", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return nil, err
	}

	switch gvk.Version {
	case "v1apha1":
		err = fmt.Errorf("unsupported CSINode storage.k8s.io/v1alpha1")
	case "v1":
		result := &storagev1.CSINodeList{}
		if err = storageClient.Get().
			Resource("csinodes").
			VersionedParams(&metav1.ListOptions{}, scheme.ParameterCodec).
			Timeout(10 * time.Second).
			Do(ctx).
			Into(result); err != nil {
			err = fmt.Errorf("unable to get csinodes; %w", err)
			break
		}
		for _, csiNode := range result.Items {
			for _, driver := range csiNode.Spec.Drivers {
				if driver.Name == consts.Identity {
					nodes = append(nodes, csiNode.Name)
					break
				}
			}
		}
	case "v1beta1":
		result := &storagev1beta1.CSINodeList{}
		if err = storageClient.Get().
			Resource(gvk.Kind).
			VersionedParams(&metav1.ListOptions{}, scheme.ParameterCodec).
			Timeout(10 * time.Second).
			Do(ctx).
			Into(result); err != nil {
			err = fmt.Errorf("unable to get csinodes; %w", err)
			break
		}
		for _, csiNode := range result.Items {
			for _, driver := range csiNode.Spec.Drivers {
				if driver.Name == consts.Identity {
					nodes = append(nodes, csiNode.Name)
					break
				}
			}
		}
	}

	return nodes, err
}
