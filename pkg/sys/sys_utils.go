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
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"k8s.io/klog/v2"

	"github.com/minio/direct-csi/pkg/sys/loopback"
)

// GetDirectCSIPath returns Direct CSI path of given drive name.
func GetDirectCSIPath(driveName string) string {
	if strings.Contains(driveName, DirectCSIDevRoot) {
		return driveName
	}
	if strings.HasPrefix(driveName, HostDevRoot) {
		return GetDirectCSIPath(filepath.Base(driveName))
	}
	return filepath.Join(DirectCSIDevRoot, driveName)
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

// FlushLoopBackReservations does umount/detach/remove loopback devices and associated files.
func FlushLoopBackReservations() error {
	umountLoopDev := func(devPath string) error {
		if err := safeUnmountAll(devPath, []UnmountOption{
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

	files, err := os.ReadDir(loopback.DirectCSIBackFileRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("error while reading (%s): %v", loopback.DirectCSIBackFileRoot, err)
	}
	for _, file := range files {
		loopDevName := file.Name() // filebase
		if err := flushLoopDevice(loopDevName); err != nil {
			return err
		}
	}
	return nil
}

// ReserveLoopbackDevices creates loopback devices.
func ReserveLoopbackDevices(devCount int) error {
	for i := 1; i <= devCount; i++ {
		dev, err := loopback.CreateLoopbackDevice()
		if err != nil {
			return err
		}
		klog.V(2).Infof("Successfully created loopback device %v", dev)
	}
	return nil
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
