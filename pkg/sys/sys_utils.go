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

package sys

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/minio/direct-csi/pkg/sys/loopback"
)

func (b *BlockDevice) DirectCSIDrivePath() string {
	return getBlockFile(b.Devname)
}

func (b *BlockDevice) HostDrivePath() string {
	return getRootBlockFile(b.Devname)
}

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
	return strings.ReplaceAll(devName, DirectCSIPartitionInfix, "")
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

func (b *BlockDevice) TagError(err error) {
	b.DeviceError = err
}

func (b *BlockDevice) Error() string {
	if b.DeviceError == nil {
		return ""
	}
	return b.DeviceError.Error()
}

func FlushLoopBackReservations() error {

	umountLoopDev := func(devPath string) error {
		if err := SafeUnmountAll(devPath, []UnmountOption{
			UnmountOptionDetach,
			UnmountOptionForce,
		}); err != nil {
			return err
		}
		return nil
	}

	flushLoopDevice := func(loopDevName string) error {
		// umount
		blockFile := getBlockFile(loopDevName)
		if err := umountLoopDev(blockFile); err != nil && !os.IsNotExist(err) {
			return err
		}
		// Remove loop device
		loopFilePath := getRootBlockFile(loopDevName)
		if err := loopback.RemoveLoopDevice(loopFilePath); err != nil && !os.IsNotExist(err) {
			return err
		}
		// Remove direct-csi (loop)device file
		if err := os.Remove(blockFile); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	files, err := ioutil.ReadDir(loopback.DirectCSIBackFileRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("Error while reading (%s): %v", loopback.DirectCSIBackFileRoot, err)
	}
	for _, file := range files {
		loopDevName := file.Name() // filebase
		if err := flushLoopDevice(loopDevName); err != nil {
			return err
		}
	}
	return nil
}

func ReserveLoopbackDevices(devCount int) error {
	for i := 1; i <= devCount; i++ {
		dev, err := loopback.CreateLoopbackDevice()
		if err != nil {
			return err
		}
		glog.V(2).Infof("Successfully created loopback device %v", dev)
	}
	return nil
}
