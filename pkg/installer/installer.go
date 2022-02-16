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

package installer

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/minio/directpv/pkg/client"
)

func trimMinorVersion(minor string) (string, error) {
	i := strings.IndexFunc(minor, func(r rune) bool { return r < '0' || r > '9' })
	if i < 0 {
		return minor, nil
	}

	m := minor[:i]
	_, err := strconv.Atoi(m)
	if err != nil {
		return "", err
	}

	return m, nil
}

func getInstaller(config *Config) (installer, error) {
	versionInfo, err := client.GetDiscoveryClient().ServerVersion()
	if err != nil {
		return nil, err
	}

	minor := versionInfo.Minor
	if strings.Contains(versionInfo.GitVersion, "-eks-") {
		// Do trimming only for EKS.
		// Refer https://github.com/aws/containers-roadmap/issues/1404
		minor, err = trimMinorVersion(versionInfo.Minor)
		if err != nil {
			return nil, err
		}
	}

	if versionInfo.Major == "1" {
		switch minor {
		case "18":
			return newV1Dot18(config), nil
		case "19":
			return newV1Dot19(config), nil
		case "20":
			return newV1Dot20(config), nil
		case "21":
			return newV1Dot21(config), nil
		case "22":
			return newV1Dot22(config), nil
		}
	}

	return nil, fmt.Errorf("unsupported kubernetes version %s.%s", versionInfo.Major, versionInfo.Minor)
}

func Install(ctx context.Context, config *Config) error {
	if config == nil {
		return errors.New("bad arguments: empty configuration")
	}
	if err := config.validate(); err != nil {
		return err
	}
	installer, err := getInstaller(config)
	if err != nil {
		return err
	}
	if !config.DryRun {
		if err := deleteLegacyConversionDeployment(ctx, config.Identity); err != nil {
			return err
		}
	}
	return installer.Install(ctx)
}

func Uninstall(ctx context.Context, config *Config) error {
	if config == nil {
		return errors.New("bad arguments: empty configuration")
	}
	if err := config.validate(); err != nil {
		return err
	}
	installer, err := getInstaller(config)
	if err != nil {
		return err
	}
	if !config.DryRun {
		if err := deleteLegacyConversionDeployment(ctx, config.Identity); err != nil {
			return err
		}
	}
	return installer.Uninstall(ctx)
}
