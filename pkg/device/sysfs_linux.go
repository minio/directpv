//go:build linux

// This file is part of MinIO DirectPV
// Copyright (c) 2022 MinIO, Inc.
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
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"k8s.io/klog/v2"
)

const defaultBlockSize = 512

func getDeviceName(majorMinor string) (string, error) {
	filename := fmt.Sprintf("/sys/dev/block/%v/uevent", majorMinor)
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		if strings.HasPrefix(s, "DEVNAME=") {
			name := strings.TrimSpace(s[8:])
			if name == "" {
				return "", fmt.Errorf("%v contains empty device name for DEVNAME key", filename)
			}
			return name, nil
		}
	}
}

func getHidden(name string) bool {
	// errors ignored since real devices do not have <sys>/hidden
	// borrow idea from 'lsblk'
	// https://github.com/util-linux/util-linux/commit/c8487d854ba5cf5bfcae78d8e5af5587e7622351
	v, _ := readFirstLine("/sys/class/block/" + name + "/hidden")
	return v == "1"
}

func getRemovable(name string) (bool, error) {
	s, err := readFirstLine("/sys/class/block/" + name + "/removable")
	return s != "" && s != "0", err
}

func getReadOnly(name string) (bool, error) {
	s, err := readFirstLine("/sys/class/block/" + name + "/ro")
	return s != "" && s != "0", err
}

func getSize(name string) (uint64, error) {
	s, err := readFirstLine("/sys/class/block/" + name + "/size")
	if err != nil || s == "" {
		return 0, err
	}
	ui64, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return ui64 * defaultBlockSize, nil
}

func getPartitions(name string) ([]string, error) {
	names, err := readdirnames("/sys/block/" + name)
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
	return readdirnames("/sys/class/block/" + name + "/holders")
}

func getDMName(name string) (string, error) {
	return readFirstLine("/sys/class/block/" + name + "/dm/name")
}

// GetStat retrieves and returns statistics for a given device name.
func GetStat(name string) ([]uint64, int, error) {
	filePath := path.Join("/sys/class/block/", name, "/stat")

	driveStatus := 1
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		klog.Warningf("Sysfs directory for drive %s does not exist", name)
		driveStatus = 0
		return nil, driveStatus, nil
	}

	klog.Infof("Reading drive statistics from: %s", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, driveStatus, fmt.Errorf("failed to open file %s: %v", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, driveStatus, fmt.Errorf("error reading file %s: %v", filePath, err)
		}
		return nil, driveStatus, fmt.Errorf("file %s is empty", filePath)
	}

	line := scanner.Text()

	var stats []uint64
	for _, token := range strings.Fields(line) {
		if token == "" {
			continue // Skip empty tokens
		}

		ui64, err := strconv.ParseUint(token, 10, 64)
		if err != nil {
			klog.Warningf("Failed to parse token '%s': %v", token, err)
			continue // Skip this token and continue with the next
		}
		stats = append(stats, ui64)
	}

	if len(stats) == 0 {
		return nil, driveStatus, fmt.Errorf("no valid statistics found in file %s", filePath)
	}

	return stats, driveStatus, nil
}

// GetHardwareSectorSize retrieves the hardware sector size for a given device.
// It works for both whole disks and partitions.
// The device name should be without the "/dev/" prefix (e.g., "sda" or "sda1").
func GetHardwareSectorSize(deviceName string) (uint64, error) {
	basePath := "/sys/class/block"
	devicePath := path.Join(basePath, deviceName)

	// Check if it's a partition
	isPartition := false
	if _, err := os.Stat(path.Join(devicePath, "partition")); err == nil {
		isPartition = true
	}

	// If it's a partition, find the parent disk
	if isPartition {
		parentPath, err := os.Readlink(devicePath)
		if err != nil {
			return 0, fmt.Errorf("failed to read link for partition %s: %v", deviceName, err)
		}
		devicePath = path.Join(basePath, path.Base(path.Dir(parentPath)))
		klog.Infof("Partition %s, using parent disk: %s", deviceName, path.Base(devicePath))
	}

	// The file containing the hardware sector size
	sectorSizePath := path.Join(devicePath, "queue/hw_sector_size")

	// Read the contents of the file
	content, err := os.ReadFile(sectorSizePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read hardware sector size for %s: %v", deviceName, err)
	}

	// Parse the sector size as an integer
	sectorSize, err := strconv.ParseUint(strings.TrimSpace(string(content)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse hardware sector size for %s: %v", deviceName, err)
	}

	klog.Infof("Hardware sector size for %s: %d bytes", deviceName, sectorSize)
	return sectorSize, nil
}
