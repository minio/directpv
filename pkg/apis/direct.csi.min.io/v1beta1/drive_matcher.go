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

package v1beta1

import (
	"github.com/mb0/glob"
	"github.com/minio/direct-csi/pkg/sys"
	"path/filepath"
	"strings"
)

func (drive *DirectCSIDrive) MatchGlob(nodes, drives, status []string) bool {

	getBasePath := func(in string) string {
		path := strings.ReplaceAll(in, sys.DirectCSIPartitionInfix, "")
		path = strings.ReplaceAll(path, sys.DirectCSIDevRoot+"/", "")
		path = strings.ReplaceAll(path, sys.HostDevRoot+"/", "")
		return filepath.Base(path)
	}

	matchGlob := func(patternList []string, name string, transformF transformFunc) bool {
		name = transformF(name)
		for _, p := range patternList {
			if ok, _ := glob.Match(p, name); ok {
				return true
			}
		}
		return false
	}

	nodeList, driveList, statusesList := checkWildcards(nodes),
		fmap(checkWildcards(drives), getBasePath),
		fmap(checkWildcards(status), strings.ToLower)

	var noOp transformFunc
	noOp = func(a string) string {
		return a
	}

	return matchGlob(nodeList, drive.Status.NodeName, noOp) &&
		matchGlob(driveList, drive.Status.Path, getBasePath) &&
		matchGlob(statusesList, string(drive.Status.DriveStatus), strings.ToLower)
}

func (drive *DirectCSIDrive) MatchAccessTier(accessTierList []AccessTier) bool {
	if len(accessTierList) == 0 {
		return true
	}
	for _, at := range accessTierList {
		if drive.Status.AccessTier == at {
			return true
		}
	}
	return false
}
