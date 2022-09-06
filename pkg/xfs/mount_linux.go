//go:build linux

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

package xfs

import (
	"errors"
	"os"

	"github.com/minio/directpv/pkg/sys"
)

func mount(device, target string) error {
	if err := os.Mkdir(target, 0o777); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}

	return sys.Mount(device, target, "xfs", []string{"noatime"}, "prjquota")
}

func bindMount(source, target string, readOnly bool) error {
	return sys.BindMount(source, target, "xfs", false, readOnly, "prjquota")
}
