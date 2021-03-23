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

package loopback

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func prepareBackingFile(loopDevNum uint64) (string, error) {
	backingFile := filepath.Join(DirectCSIBackFileRoot, fmt.Sprintf("loop%d", uint64(loopDevNum)))
	bytesToWrite := make([]byte, backFileSize)
	return backingFile, ioutil.WriteFile(backingFile, bytesToWrite, 0666)
}

func getDeviceFileName(ldNumber uint64) string {
	fileName := fmt.Sprintf(LoopDeviceFormat, ldNumber)
	return fileName
}

func checkIfBackingFileAttached(ldNumber uint64) (bool, error) {
	info, iErr := getInfoFromLDNumber(ldNumber)
	if iErr != nil {
		return false, fmt.Errorf("could not get existing loop device info (loop device number: %d): %v", ldNumber, iErr)
	}

	hasFileNameAttached := func() bool {
		emptyFileNameinB := [NameSize]byte{}
		return bytes.Compare(emptyFileNameinB[:], info.FileName[:]) != 0
	}()

	return hasFileNameAttached, nil
}

func getInfoFromLDNumber(ldNumber uint64) (LoopInfo, error) {
	devFile := getDeviceFileName(ldNumber)
	loopFile, err := os.OpenFile(devFile, os.O_RDWR, 0660)
	if err != nil {
		return LoopInfo{}, fmt.Errorf("could not open loop device: %v", err)
	}
	defer loopFile.Close()

	info, iErr := getInfo(loopFile.Fd())
	if iErr != nil {
		return info, iErr
	}

	return info, nil
}

func getLoopDeviceNumber(devName string) (uint64, error) {
	if !strings.HasPrefix(devName, "/dev/loop") {
		return uint64(0), errNotALoopDevice
	}
	numStr := strings.TrimPrefix(devName, "/dev/loop")
	num, err := strconv.ParseUint(numStr, 10, 64)
	if err != nil {
		return uint64(0), err
	}
	return num, nil
}
