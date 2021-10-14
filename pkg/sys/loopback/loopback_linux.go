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
	"errors"
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

var (
	errNotALoopDevice      = errors.New("not a loop device")
	errAlreadyBackedByFile = errors.New("device already backed by a file")
	errNotBackedByFile     = errors.New("device not backed by a file")
	errDoesNotExist        = errors.New("no such file or directory")
	backFileSize           = 1024 * oneMB
)

type loopInfo struct {
	Device         uint64
	INode          uint64
	RDevice        uint64
	Offset         uint64
	SizeLimit      uint64
	Number         uint32
	EncryptType    uint32
	EncryptkeySize uint32
	Flags          uint32
	FileName       [nameSize]byte
	CryptName      [nameSize]byte
	EncryptKey     [keySize]byte
	Init           [2]uint64
}

func getFree() (uint64, error) {
	ctrl, err := os.OpenFile(loopControlPath, os.O_RDWR, 0660)
	if err != nil {
		return uint64(0), fmt.Errorf("could not open %v: %v", loopControlPath, err)
	}
	defer ctrl.Close()
	dev, _, errno := unix.Syscall(unix.SYS_IOCTL, ctrl.Fd(), ctlGetFree, 0)
	if int64(dev) < 0 {
		return uint64(0), fmt.Errorf("could not get free device (err: %d): %v", errno, errno)
	}

	return uint64(dev), nil
}

// CreateLoopbackDevice creates loopback device.
func CreateLoopbackDevice() (string, error) {
	if err := os.MkdirAll(DirectCSIBackFileRoot, 0755); err != nil {
		return "", err
	}

getFree:
	devNum, err := getFree()
	if err != nil {
		return "", err
	}

	backingFile, pErr := prepareBackingFile(devNum)
	if pErr != nil {
		return "", pErr
	}

	if err := addLoopDevice(devNum); err != nil {
		// cleanup the file
		os.Remove(backingFile)
		// Re-run the selection if already backed
		if err == errAlreadyBackedByFile {
			goto getFree
		}
		return "", err
	}

	devFile := getDeviceFileName(devNum)
	if err := attachLoopbackDeviceToFile(devFile, backingFile); err != nil {
		return "", err
	}

	return devFile, nil
}

func attachLoopbackDeviceToFile(devFile, backingFile string) error {
	// Open backing file
	back, err := os.OpenFile(backingFile, os.O_RDWR, 0660)
	if err != nil {
		return fmt.Errorf("could not open backing file: %v", err)
	}
	defer back.Close()

	// Open loop device file
	loopFile, err := os.OpenFile(devFile, os.O_RDWR, 0660)
	if err != nil {
		return fmt.Errorf("could not open loop device: %v", err)
	}
	defer loopFile.Close()

	// Attach backfile to loop device
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, loopFile.Fd(), setFD, back.Fd())
	if errno != 0 {
		return fmt.Errorf("could not attach backing file (%s) with loop device (%s): errno: %v", backingFile, devFile, errno)
	}

	// Setting the backing filename in the device info
	info := loopInfo{}
	copy(info.FileName[:], []byte(backingFile))
	info.Offset = uint64(0)
	if err := setInfo(loopFile.Fd(), info); err != nil {
		_, _, _ = unix.Syscall(unix.SYS_IOCTL, loopFile.Fd(), clrFD, 0)
		return fmt.Errorf("could not set info: %v", err)
	}

	return nil
}

// Add will add a loopback device if it does not exist already.
func addLoopDevice(ldNumber uint64) error {
	ctrl, err := os.OpenFile(loopControlPath, os.O_RDWR, 0660)
	if err != nil {
		return fmt.Errorf("could not open %v: %v", loopControlPath, err)
	}
	defer ctrl.Close()

	_, _, errno := unix.Syscall(unix.SYS_IOCTL, ctrl.Fd(), ctlAdd, uintptr(ldNumber))
	if errno == unix.EEXIST {
		hasFileNameAttached, err := checkIfBackingFileAttached(ldNumber)
		if err != nil {
			return err
		}
		if hasFileNameAttached {
			return errAlreadyBackedByFile
		}
		return nil
	}
	if errno != 0 {
		return fmt.Errorf("could not add device (err: %d): %v", errno, errno)
	}
	return nil
}

func setInfo(fd uintptr, info loopInfo) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, setStatus64, uintptr(unsafe.Pointer(&info)))
	if errno == unix.ENXIO {
		return errNotBackedByFile
	} else if errno != 0 {
		return fmt.Errorf("could set get info %v", errno)
	}
	return nil
}

func getInfo(fd uintptr) (loopInfo, error) {
	retInfo := loopInfo{}
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, getStatus64, uintptr(unsafe.Pointer(&retInfo)))
	if errno != 0 && errno != unix.ENXIO {
		if errno == unix.ENOENT {
			return retInfo, errDoesNotExist
		}
		// errno == unix.ENXIO indicates that the device is not backed by a file
		return retInfo, fmt.Errorf("could not get info: %v", errno)
	}
	return retInfo, nil
}

// RemoveLoopDevice removes given loopback device.
func RemoveLoopDevice(loopPath string) error {
	loopFile, err := os.OpenFile(loopPath, os.O_RDONLY, 0660)
	if err != nil {
		return err
	}
	defer loopFile.Close()

	// Getting the loopfile info
	info, err := getInfo(loopFile.Fd())
	if err != nil && err != errDoesNotExist {
		return fmt.Errorf("cannot get the loopfile info: %v", err)
	}
	backFileInB := bytes.Trim(info.FileName[:], "\x00")
	backFile := string(backFileInB)

	// Detaching the backing file
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, loopFile.Fd(), clrFD, 0)
	if errno != 0 && errno != unix.ENXIO {
		return fmt.Errorf("error clearing loopfile: %v", errno)
	}

	// Removing the backing file
	if backFile = backFile[:]; backFile != "" {
		if err := os.Remove(backFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove the backfile: %v", err)
		}
	}

	// Removing the loop device
	if err := removeLoopDevice(loopPath); err != nil && err != errDoesNotExist {
		return err
	}

	return nil
}

func removeLoopDevice(path string) error {
	ctrl, err := os.OpenFile(loopControlPath, os.O_RDWR, 0660)
	if err != nil {
		return fmt.Errorf("could not open %v: %v", loopControlPath, err)
	}
	defer ctrl.Close()

	devNumber, lErr := getLoopDeviceNumber(path)
	if lErr != nil {
		return lErr
	}
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, ctrl.Fd(), ctlRemove, uintptr(devNumber))
	if errno == unix.EBUSY {
		return nil
	}
	if errno == unix.ENOENT {
		return errDoesNotExist
	}
	if errno != 0 {
		return fmt.Errorf("could not remove (err: %d): %v", errno, errno)
	}

	return nil
}
