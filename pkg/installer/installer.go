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

package installer

import (
	"context"
	"fmt"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	defaultLabels = map[string]string{ // labels
		AppNameLabel: DirectCSI,
		AppTypeLabel: CSIDriver,

		utils.CreatedByLabel: DirectCSIPluginName,
		utils.VersionLabel:   directcsi.Version,
	}

	defaultAnnotations = map[string]string{}
)

// Installer is installer interface
type Installer interface {
	Install(context.Context) error
	Uninstall(context.Context) error
}

type installConfig struct {
	Identity string

	// DirectCSIContainerImage properties
	DirectCSIContainerImage    string
	DirectCSIContainerOrg      string
	DirectCSIContainerRegistry string

	// CSIImage properties
	CSIProvisionerImage      string
	NodeDriverRegistrarImage string
	LivenessProbeImage       string

	// Mode switches
	LoopBackMode bool

	// dry-run properties
	DryRun       bool
	DryRunFormat DryRunFormat
}

func (i *installConfig) GetIdentity() string {
	return i.Identity
}

func (i *installConfig) GetDryRunFormat() DryRunFormat {
	if i.DryRunFormat == "" {
		return DryRunFormatYAML
	}
	return i.DryRunFormat
}

func (i *installConfig) PostProc(obj interface{}) error {
	if i.DryRun {
		var format func(interface{}) string
		dryRunFormat := i.GetDryRunFormat()
		if dryRunFormat == DryRunFormatJSON {
			format = utils.MustJSON
		} else {
			format = func(obj interface{}) string {
				return fmt.Sprintf("%s\n---\n", utils.MustYAML(obj))
			}
		}
		fmt.Println(format(obj))
	}

	return nil
}

func (i *installConfig) getDryRunDirectives() []string {
	if i.DryRun {
		return []string{
			metav1.DryRunAll,
		}
	}
	return []string{}
}
