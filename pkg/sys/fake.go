// This file is part of MinIO DirectPV
// Copyright (c) 2025 MinIO, Inc.
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

package sys

// FakeMountInfo creates mount information with tesing mount entries
func FakeMountInfo(mountEntries ...MountEntry) *MountInfo {
	infoMap := map[uint64]*MountEntry{}
	mountPointMap := map[string][]uint64{}
	mountSourceMap := map[string][]uint64{}
	majorMinorMap := map[string][]uint64{}
	rootMap := map[string][]uint64{}

	for i, mountEntry := range mountEntries {
		xxHash := uint64(i)
		infoMap[xxHash] = &mountEntry
		mountPointMap[mountEntry.MountPoint] = append(mountPointMap[mountEntry.MountPoint], xxHash)
		mountSourceMap[mountEntry.MountSource] = append(mountSourceMap[mountEntry.MountSource], xxHash)
		majorMinorMap[mountEntry.MajorMinor] = append(majorMinorMap[mountEntry.MajorMinor], xxHash)
		rootMap[mountEntry.Root] = append(rootMap[mountEntry.Root], xxHash)
	}

	return &MountInfo{
		infoMap:        infoMap,
		mountPointMap:  mountPointMap,
		mountSourceMap: mountSourceMap,
		majorMinorMap:  majorMinorMap,
		rootMap:        rootMap,
	}
}
