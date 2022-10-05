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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/manifoldco/promptui"
	"github.com/minio/directpv/pkg/admin"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/ellipsis"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	"github.com/minio/directpv/pkg/volume"
	"github.com/mitchellh/go-homedir"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

const dot = "•"

var (
	globRegexp                = regexp.MustCompile(`(^|[^\\])[\*\?\[]`)
	errGlobPatternUnsupported = errors.New("glob patterns are unsupported")
	bold                      = color.New(color.Bold).SprintFunc()
	italic                    = color.New(color.Italic).SprintFunc()
	// prompt
	answerYes = "YES"
	answerNo  = "NO"
)

func printYAML(obj interface{}) error {
	y, err := utils.ToYAML(obj)
	if err != nil {
		return err
	}
	fmt.Println(y)
	return nil
}

func printJSON(obj interface{}) error {
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to marshal object; %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func parseNodeSelector(values []string) (map[string]string, error) {
	nodeSelector := map[string]string{}
	for _, value := range values {
		tokens := strings.Split(value, "=")
		if len(tokens) != 2 {
			return nil, fmt.Errorf("invalid node selector value %v", value)
		}
		if tokens[0] == "" {
			return nil, fmt.Errorf("invalid key in node selector value %v", value)
		}
		nodeSelector[tokens[0]] = tokens[1]
	}
	return nodeSelector, nil
}

func parseTolerations(values []string) ([]corev1.Toleration, error) {
	tolerations := []corev1.Toleration{}
	for _, value := range values {
		var k, v, e string
		tokens := strings.SplitN(value, "=", 2)
		switch len(tokens) {
		case 1:
			k = tokens[0]
			tokens = strings.Split(k, ":")
			switch len(tokens) {
			case 1:
			case 2:
				k, e = tokens[0], tokens[1]
			default:
				if len(tokens) != 2 {
					return nil, fmt.Errorf("invalid toleration %v", value)
				}
			}
		case 2:
			k, v = tokens[0], tokens[1]
		default:
			if len(tokens) != 2 {
				return nil, fmt.Errorf("invalid toleration %v", value)
			}
		}
		if k == "" {
			return nil, fmt.Errorf("invalid key in toleration %v", value)
		}
		if v != "" {
			if tokens = strings.Split(v, ":"); len(tokens) != 2 {
				return nil, fmt.Errorf("invalid value in toleration %v", value)
			}
			v, e = tokens[0], tokens[1]
		}
		effect := corev1.TaintEffect(e)
		switch effect {
		case corev1.TaintEffectNoSchedule, corev1.TaintEffectPreferNoSchedule, corev1.TaintEffectNoExecute:
		default:
			return nil, fmt.Errorf("invalid toleration effect in toleration %v", value)
		}
		operator := corev1.TolerationOpExists
		if v != "" {
			operator = corev1.TolerationOpEqual
		}
		tolerations = append(tolerations, corev1.Toleration{
			Key:      k,
			Operator: operator,
			Value:    v,
			Effect:   effect,
		})
	}

	return tolerations, nil
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
	return utils.NewSafeFile(path.Join(defaultAuditDir, fmt.Sprintf("%v.%v", auditFile, time.Now().UnixNano())))
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

func getLabelValue(obj metav1.Object, key string) string {
	if labels := obj.GetLabels(); labels != nil {
		return labels[key]
	}
	return ""
}

func getVolumesByNames(ctx context.Context, names []string) <-chan volume.ListVolumeResult {
	resultCh := make(chan volume.ListVolumeResult)
	go func() {
		defer close(resultCh)
		for _, name := range names {
			volumeName := strings.TrimSpace(name)
			vol, err := client.VolumeClient().Get(ctx, volumeName, metav1.GetOptions{})
			switch {
			case err == nil:
				resultCh <- volume.ListVolumeResult{Volume: *vol}
			case apierrors.IsNotFound(err):
				klog.V(5).Infof("No volume found by name %v", volumeName)
			default:
				klog.ErrorS(err, "unable to get volume", "volumeName", volumeName)
				return
			}
		}
	}()
	return resultCh
}

func processFilteredVolumes(
	ctx context.Context,
	names []string,
	matchFunc func(*types.Volume) bool,
	applyFunc func(*types.Volume) error,
	processFunc func(context.Context, *types.Volume) error,
	auditFile string,
) error {
	var resultCh <-chan volume.ListVolumeResult
	var err error

	if applyFunc == nil || processFunc == nil {
		klog.Fatalf("Either applyFunc or processFunc must be provided. This should not happen.")
	}

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	if len(names) == 0 {
		resultCh, err = volume.ListVolumes(ctx,
			nodeSelectors,
			driveSelectors,
			podNameSelectors,
			podNSSelectors,
			k8s.MaxThreadCount)
		if err != nil {
			return err
		}
	} else {
		resultCh = getVolumesByNames(ctx, names)
	}

	file, err := openAuditFile(auditFile)
	if err != nil {
		klog.ErrorS(err, "unable to open audit file", "auditFile", auditFile)
	}

	defer func() {
		if file != nil {
			if err := file.Close(); err != nil {
				klog.ErrorS(err, "unable to close audit file")
			}
		}
	}()

	if matchFunc == nil {
		matchFunc = func(volume *types.Volume) bool { return true }
	}

	return volume.ProcessVolumes(
		ctx,
		resultCh,
		matchFunc,
		applyFunc,
		processFunc,
		file,
		dryRun,
	)
}

func getSelectorValues(selectors []string) (values []types.LabelValue, err error) {
	for _, selector := range selectors {
		if globRegexp.MatchString(selector) {
			return nil, errGlobPatternUnsupported
		}

		result, err := ellipsis.Expand(selector)
		if err != nil {
			return nil, err
		}

		for _, value := range result {
			values = append(values, types.NewLabelValue(value))
		}
	}

	return values, nil
}

func getDriveSelectors() ([]types.LabelValue, error) {
	var values []string
	for i := range driveArgs {
		if utils.TrimDevPrefix(driveArgs[i]) == "" {
			return nil, fmt.Errorf("empty device name %v", driveArgs[i])
		}
		values = append(values, utils.TrimDevPrefix(driveArgs[i]))
	}
	return getSelectorValues(values)
}

func expandDriveArgs() ([]string, error) {
	var values []string
	for i := range driveArgs {
		trimmed := utils.TrimDevPrefix(strings.TrimSpace(driveArgs[i]))
		if trimmed == "" {
			return nil, fmt.Errorf("empty device name %v", driveArgs[i])
		}
		result, err := ellipsis.Expand(trimmed)
		if err != nil {
			return nil, err
		}
		values = append(values, result...)
	}
	return values, nil
}

func expandNodeArgs() ([]string, error) {
	var values []string
	for i := range nodeArgs {
		result, err := ellipsis.Expand(strings.TrimSpace(nodeArgs[i]))
		if err != nil {
			return nil, err
		}
		values = append(values, result...)
	}
	return values, nil
}

func getNodeSelectors() ([]types.LabelValue, error) {
	for i := range nodeArgs {
		if nodeArgs[i] == "" {
			return nil, fmt.Errorf("empty node name %v", nodeArgs[i])
		}
	}
	return getSelectorValues(nodeArgs)
}

func printTable(deviceInfo map[admin.NodeName][]admin.Device) {
	fmt.Println()
	defer func() {
		fmt.Println()
	}()

	if len(deviceInfo) == 0 {
		return
	}

	getStatusAndDesc := func(device admin.Device) (string, string) {
		switch device.Status {
		case admin.DeviceStatusAvailable:
			return color.GreenString(string(device.Status)), color.GreenString(italic(device.Description))
		case admin.DeviceStatusUnavailable:
			return color.RedString(string(device.Status)), color.RedString(italic(device.Description))
		default:
			return string(device.Status), device.Description
		}
	}

	headers := table.Row{
		"NAME",
		"SIZE",
		"FILESYSTEM",
		"NODE",
		"MODEL",
		"VENDOR",
		"STATUS",
		"DESCRIPTION",
	}
	text.DisableColors()
	writer := table.NewWriter()
	writer.SetOutputMirror(os.Stdout)
	writer.AppendHeader(headers)

	style := table.StyleColoredDark
	style.Options = table.OptionsDefault
	style.Color.IndexColumn = text.Colors{text.FgHiBlue, text.BgHiBlack}
	style.Color.Header = text.Colors{text.FgHiBlue, text.BgHiBlack}

	for nodeName, deviceList := range deviceInfo {
		for _, device := range deviceList {
			status, desc := getStatusAndDesc(device)
			row := []interface{}{
				device.Name,
				printableBytes(int64(device.Size)),
				printableString(device.Filesystem),
				nodeName,
				printableString(device.Model),
				printableString(device.Vendor),
				status,
				desc,
			}
			writer.AppendRow(row)
		}
	}

	writer.SortBy(
		[]table.SortBy{
			{
				Name: "NAME",
				Mode: table.Asc,
			},
			{
				Name: "NODE",
				Mode: table.Asc,
			},
			{
				Name: "SIZE",
				Mode: table.Asc,
			},
		},
	)

	writer.SetStyle(style)
	writer.Render()
}

// expandFormatArgs expand the format device args
func expandFormatArgs() (err error) {
	driveArgs, err = expandDriveArgs()
	if err != nil {
		return err
	}
	nodeArgs, err = expandNodeArgs()
	return
}

// AskQuestion user for generic input."
func askQuestion(question string, validate func(string) error) string {
	fmt.Println()
	prompt := promptui.Prompt{
		Label: question,
		Validate: func(input string) error {
			if validate != nil {
				return validate(input)
			}
			return nil
		},
	}
	result, err := prompt.Run()
	if err == promptui.ErrInterrupt {
		os.Exit(-1)
	}
	return result
}

// Ask user for Y/N input. Return true if response is "y"
func ask(label string) bool {
	return askQuestion(label, nil) == answerYes
}
