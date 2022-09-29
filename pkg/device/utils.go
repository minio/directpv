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

package device

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"

	"github.com/minio/directpv/pkg/xfs"
	"golang.org/x/sys/unix"
)

const (
	defaultBlockSize = 512
)

func getHidden(name string) bool {
	// errors ignored since real devices do not have <sys>/hidden
	// borrow idea from 'lsblk'
	// https://github.com/util-linux/util-linux/commit/c8487d854ba5cf5bfcae78d8e5af5587e7622351
	v, _ := readFirstLine("/sys/class/block/"+name+"/hidden", false)
	return v == "1"
}

func getRemovable(name string) (bool, error) {
	s, err := readFirstLine("/sys/class/block/"+name+"/removable", false)
	return s != "" && s != "0", err
}

func getReadOnly(name string) (bool, error) {
	s, err := readFirstLine("/sys/class/block/"+name+"/ro", false)
	return s != "" && s != "0", err
}

func getSize(name string) (uint64, error) {
	s, err := readFirstLine("/sys/class/block/"+name+"/size", true)
	if err != nil {
		return 0, err
	}
	ui64, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return ui64 * defaultBlockSize, nil
}

func getPartitions(name string) ([]string, error) {
	names, err := readdirnames("/sys/block/"+name, false)
	if err != nil {
		return nil, err
	}

	partitions := []string{}
	for _, n := range names {
		if strings.HasPrefix(n, name) {
			partitions = append(partitions, n)
		}
	}

	return partitions, nil
}

func getHolders(name string) ([]string, error) {
	return readdirnames("/sys/block/"+name+"/holders", false)
}

func getCDROMs() (map[string]struct{}, error) {
	file, err := os.Open("/proc/sys/dev/cdrom/info")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]struct{}{}, nil
		}
		return nil, err
	}
	defer file.Close()
	return parseCDROMs(file)
}

func parseCDROMs(r io.Reader) (map[string]struct{}, error) {
	reader := bufio.NewReader(r)
	names := map[string]struct{}{}
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		if tokens := strings.SplitAfterN(s, "drive name:", 2); len(tokens) == 2 {
			for _, token := range strings.Fields(tokens[1]) {
				if token != "" {
					names[token] = struct{}{}
				}
			}
			break
		}
	}
	return names, nil
}

func getSwaps() (map[string]struct{}, error) {
	file, err := os.Open("/proc/swaps")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]struct{}{}, nil
		}
		return nil, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	filenames := []string{}
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		filenames = append(filenames, strings.Fields(s)[0])
	}

	devices := map[string]struct{}{}
	for _, filename := range filenames[1:] {
		major, minor, err := getDeviceMajorMinor(filename)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}

		devices[fmt.Sprintf("%v:%v", major, minor)] = struct{}{}
	}
	return devices, nil
}

func updateFSInfo(device *Device) error {
	// Probe only for "xfs" devices
	// UDev may have empty ID_FS_TYPE for xfs devices (ref: https://github.com/minio/directpv/issues/602)
	udevFSType := device.FSType()
	if udevFSType == "" || strings.EqualFold(udevFSType, "xfs") {
		fsuuid, _, totalCapacity, freeCapacity, err := xfs.Probe(device.Path())
		if err != nil {
			if device.Size > 0 {
				switch {
				case errors.Is(err, xfs.ErrFSNotFound), errors.Is(err, xfs.ErrCanceled), errors.Is(err, io.ErrUnexpectedEOF):
				default:
					return err
				}
			}
		} else {
			device.FSUUID = fsuuid
			device.TotalCapacity = totalCapacity
			device.FreeCapacity = freeCapacity
		}
	}
	return nil
}

func readFirstLine(filename string, errorIfNotExist bool) (string, error) {
	getError := func(err error) error {
		if errorIfNotExist {
			return err
		}
		switch {
		case errors.Is(err, os.ErrNotExist), errors.Is(err, os.ErrInvalid):
			return nil
		case strings.Contains(strings.ToLower(err.Error()), "no such device"):
			return nil
		case strings.Contains(strings.ToLower(err.Error()), "invalid argument"):
			return nil
		}
		return err
	}

	file, err := os.Open(filename)
	if err != nil {
		return "", getError(err)
	}
	defer file.Close()
	s, err := bufio.NewReader(file).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", getError(err)
	}
	return strings.TrimSpace(s), nil
}

func readdirnames(dirname string, errorIfNotExist bool) ([]string, error) {
	dir, err := os.Open(dirname)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !errorIfNotExist {
			err = nil
		}
		return nil, err
	}
	defer dir.Close()
	return dir.Readdirnames(-1)
}

func getDeviceMajorMinor(device string) (major, minor uint32, err error) {
	stat := syscall.Stat_t{}
	if err = syscall.Stat(device, &stat); err == nil {
		major, minor = uint32(unix.Major(stat.Rdev)), uint32(unix.Minor(stat.Rdev))
	}
	return
}

// getMajorMinorFromStr parses the maj:min string and extracts major and minor
func getMajorMinorFromStr(majMin string) (major, minor uint32, err error) {
	tokens := strings.SplitN(majMin, ":", 2)
	if len(tokens) != 2 {
		err = fmt.Errorf("unknown format of %v", majMin)
		return
	}

	var major64, minor64 uint64
	major64, err = strconv.ParseUint(tokens[0], 10, 32)
	if err != nil {
		return
	}
	major = uint32(major64)

	minor64, err = strconv.ParseUint(tokens[1], 10, 32)
	minor = uint32(minor64)
	return
}

// isLoopBackDevice checks if the device is a loopback or not
func isLoopBackDevice(devPath string) bool {
	return strings.HasPrefix(path.Base(devPath), "loop")
}
