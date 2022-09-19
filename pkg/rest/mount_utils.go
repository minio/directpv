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

package rest

import (
	"os"

	"github.com/minio/directpv/pkg/sys"
)

const (
	mountFlagNoAtime = "noatime"
	mountOptPrjQuota = "prjquota"
)

func mountXFSDevice(device, target string, flags []string) error {
	if err := os.MkdirAll(target, 0o777); err != nil {
		return err
	}
	// mount with "noatime" by default
	flags = append(flags, mountFlagNoAtime)
	// safemounting with "prjquota" mountopt
	return sys.Mount(device, target, "xfs", flags, mountOptPrjQuota)
}
