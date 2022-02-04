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

package sys

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

func getBlockFile(devName string) string {
	if strings.Contains(devName, DirectCSIDevRoot) {
		return devName
	}
	if strings.HasPrefix(devName, HostDevRoot) {
		return getBlockFile(filepath.Base(devName))
	}
	return filepath.Join(DirectCSIDevRoot, makeBlockDeviceName(devName))
}

func makeBlockDeviceName(devName string) string {
	dName, partNum := splitDevAndPartNum(devName)

	partNumStr := func() string {
		if partNum == 0 {
			return ""
		}
		return strconv.Itoa(partNum)
	}()

	if partNumStr == "" {
		return devName
	}

	return strings.Join([]string{dName, partNumStr}, DirectCSIPartitionInfix)
}

func getRootBlockFile(devName string) string {
	if strings.Contains(devName, DirectCSIDevRoot) {
		return getRootBlockFile(filepath.Base(devName))
	}
	if strings.HasPrefix(devName, HostDevRoot) {
		return devName
	}
	return filepath.Join(HostDevRoot, makeRootDeviceName(devName))
}

func makeRootDeviceName(devName string) string {
	cleanPrefix := strings.Replace(devName, DirectCSIPartitionInfix, "", 1)
	return strings.ReplaceAll(cleanPrefix, DirectCSIPartitionInfix, HostPartitionInfix)
}

func splitDevAndPartNum(s string) (string, int) {
	possibleNum := strings.Builder{}
	toRet := strings.Builder{}

	// finds number at the end of a string
	for _, r := range s {
		if r >= '0' && r <= '9' {
			possibleNum.WriteRune(r)
			continue
		}
		toRet.WriteString(possibleNum.String())
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
}

func isFATFSType(fsType string) bool {
	switch fsType {
	case "fat", "vfat", "fat12", "fat16", "fat32":
		return true
	default:
		return false
	}
}

func isSwapFSType(fsType string) bool {
	switch fsType {
	case "linux-swap", "swap":
		return true
	default:
		return false
	}
}

func FSTypeEqual(fsType1, fsType2 string) bool {
	fsType1, fsType2 = strings.ToLower(fsType1), strings.ToLower(fsType2)
	switch {
	case fsType1 == fsType2:
		return true
	case isFATFSType(fsType1) && isFATFSType(fsType2):
		return true
	case isSwapFSType(fsType1) && isSwapFSType(fsType2):
		return true
	default:
		return false
	}
}

func NormalizeUUID(uuid string) string {
	if u := strings.ReplaceAll(strings.ReplaceAll(uuid, ":", ""), "-", ""); len(u) > 20 {
		uuid = fmt.Sprintf("%v-%v-%v-%v-%v", u[:8], u[8:12], u[12:16], u[16:20], u[20:])
	}
	return uuid
}
