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

package v1beta4

import (
	"github.com/minio/directpv/pkg/matcher"
)

// MatchStatus does volume's condition status with given status list.
func (volume *DirectCSIVolume) MatchStatus(statusList []string) bool {
	return matcher.MatchTrueConditions(
		volume.Status.Conditions,
		[]string{string(DirectCSIVolumeConditionPublished), string(DirectCSIVolumeConditionStaged)},
		statusList,
	)
}

// MatchPodName matches pod name of this volume with atleast of the patterns.
func (volume *DirectCSIVolume) MatchPodName(patterns []string) bool {
	return matcher.GlobMatch(volume.Labels[Group+"/pod.name"], patterns)
}

// MatchPodNamespace matches pod namespace of this volume with atleast of the patterns.
func (volume *DirectCSIVolume) MatchPodNamespace(patterns []string) bool {
	return matcher.GlobMatch(volume.Labels[Group+"/pod.namespace"], patterns)
}

func (volume *DirectCSIVolume) MatchNodeDrives(nodes, drives []string) bool {
	return matcher.GlobMatchNodesDrivesStatuses(nodes, drives, nil, volume.Labels[Group+"/node"], volume.Labels[Group+"/drive-path"], "")
}
