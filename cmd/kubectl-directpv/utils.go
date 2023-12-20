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
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/table"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/utils"
	"github.com/mitchellh/go-homedir"
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
