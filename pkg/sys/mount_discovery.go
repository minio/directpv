// This file is part of MinIO Direct CSI
// Copyright (c) 2020 MinIO, Inc.
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

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (b *BlockDevice) probeMountInfo(partitionNum uint) ([]Mount, error) {
	mounts, err := ProbeMountInfo()
	if err != nil {
		return nil, err
	}
	devName := b.Devname
	newDevName := getBlockFile(devName)
	rootDevName := getRootBlockFile(devName)
	toRet := []Mount{}
	for _, m := range mounts {
		if m.DevName != rootDevName && m.DevName != newDevName {
			continue
		}
		if partitionNum > 0 {
			if m.PartitionNum != partitionNum {
				continue
			}
		}
		toRet = append(toRet, m)
	}
	return toRet, nil
}

// ProbeMountInfo - fetches the list of mounted filesystems on particular node
func ProbeMountInfo() ([]Mount, error) {
	mountinfoFile := filepath.Join(DefaultProcFS, "1", "mountinfo")
	f, err := os.Open(mountinfoFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	mounts := []Mount{}
	fbuf := bufio.NewReader(f)

	for {
		line, err := fbuf.ReadString(byte('\n'))
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}
		parts := strings.SplitN(line, " - ", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format of %s", mountinfoFile)
		}
		firstParts := strings.Fields(strings.TrimSpace(parts[0]))
		if len(firstParts) < 7 {
			return nil, fmt.Errorf("invalid format of %s", mountinfoFile)
		}
		mID, err := strconv.ParseUint(firstParts[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid format of %s", mountinfoFile)
		}
		mountID := uint32(mID)

		pID, err := strconv.ParseUint(firstParts[1], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid format of %s", mountinfoFile)
		}
		parentID := uint32(pID)

		mountRoot := firstParts[3]
		mountPoint := firstParts[4]
		mountOptions := firstParts[5]
		optionalFields := firstParts[6:]
		mountFlags := strings.Split(mountOptions, ",")

		secondParts := strings.Fields(strings.TrimSpace(parts[1]))
		if len(secondParts) < 3 {
			return nil, fmt.Errorf("invalid format of %s", mountinfoFile)
		}

		fsType := secondParts[0]
		mountSource := secondParts[1]
		superblockOptions := strings.Split(secondParts[2], ",")

		devName, partNum := func(s string) (string, int) {
			possibleNum := strings.Builder{}
			toRet := strings.Builder{}

			//finds number at the end of a string
			for _, r := range s {
				if r >= '0' && r <= '9' {
					possibleNum.WriteRune(r)
					continue
				}
				toRet.WriteRune(r)
				possibleNum.Reset()
			}
			num := possibleNum.String()
			str := toRet.String()
			if len(num) > 0 {
				numVal, err := strconv.Atoi(num)
				if err != nil {
					// return full input string in this case
					return s, 0
				}
				return str, numVal
			}
			return str, 0
		}(mountSource)

		mounts = append(mounts, Mount{
			Mountpoint:        mountPoint,
			MountFlags:        mountFlags,
			MountRoot:         mountRoot,
			MountID:           mountID,
			ParentID:          parentID,
			MountSource:       mountSource,
			SuperblockOptions: superblockOptions,
			FSType:            fsType,
			OptionalFields:    optionalFields,
			DevName:           devName,
			PartitionNum:      uint(partNum),
		})
	}
	return mounts, nil
}
