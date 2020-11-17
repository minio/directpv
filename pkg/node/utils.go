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

package node

import (
	"bufio"
	"context"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"k8s.io/utils/mount"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/pborman/uuid"

	direct_csi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
)

func FindDrives(ctx context.Context, nodeID string) ([]*direct_csi.DirectCSIDrive, error) {
	drives := map[string]*direct_csi.DirectCSIDrive{}
	visited := map[string]struct{}{}
	if err := WalkWithFollow("/sys/block/", func(path string, info os.FileInfo, err error) error {
		if strings.Compare(path, "/sys/block/") == 0 {
			return nil
		}
		if err != nil {
			if os.IsPermission(err) {
				// skip
				return nil
			}
			return err
		}
		if strings.HasPrefix(info.Name(), "loop") {
			return filepath.SkipDir
		}
		if strings.HasPrefix(info.Name(), "driver") {
			return filepath.SkipDir
		}
		if strings.HasPrefix(info.Name(), "iommu") {
			return filepath.SkipDir
		}
		if strings.Compare(info.Name(), "firmware_node") == 0 {
			return filepath.SkipDir
		}
		if strings.Compare(info.Name(), "kernel") == 0 {
			return filepath.SkipDir
		}
		if strings.Compare(info.Name(), "pci") == 0 {
			return filepath.SkipDir
		}
		if strings.Compare(info.Name(), "devices") == 0 {
			return filepath.SkipDir
		}

		link, err := os.Readlink(path)
		if err != nil {
			link = path
		}
		if _, ok := visited[link]; ok {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		visited[link] = struct{}{}

		// Find drive specific info
		drive := getDriveForPath(drives, link)
		if drive == nil {
			return nil
		}
		if strings.Compare(info.Name(), "wwid") == 0 {
			if drive.SerialNumber == "" {
				data, err := ioutil.ReadFile(link)
				if err != nil {
					return err
				}
				drive.SerialNumber = strings.TrimSpace(string(data))
			}
		}
		if strings.Compare(info.Name(), "model") == 0 {
			if drive.ModelNumber == "" {
				data, err := ioutil.ReadFile(link)
				if err != nil {
					return err
				}
				drive.ModelNumber = strings.TrimSpace(string(data))
			}
		}
		if strings.Compare(info.Name(), "partition") == 0 {
			// not needed if this is a root partition
			if drive.Name == drive.RootPartition {
				return nil
			}
			data, err := ioutil.ReadFile(link)
			if err != nil {
				return err
			}
			partNum, err := strconv.Atoi(strings.TrimSpace(string(data)))
			if err != nil {
				return err
			}
			drive.PartitionNum = partNum
		}
		if strings.Compare(info.Name(), "size") == 0 {
			data, err := ioutil.ReadFile(link)
			if err != nil {
				return err
			}
			size, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
			if err != nil {
				return err
			}
			blockSize := int64(1)
			if drive.BlockSize != 0 {
				blockSize = drive.BlockSize
			}
			size = size * blockSize
			drive.TotalCapacity = size
		}
		if strings.Compare(info.Name(), "logical_block_size") == 0 {
			data, err := ioutil.ReadFile(link)
			if err != nil {
				return err
			}
			size, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
			if err != nil {
				return err
			}
			blocks := int64(1)
			if drive.TotalCapacity != 0 {
				blocks = drive.TotalCapacity
			}
			drive.BlockSize = size
			totalSize := size * blocks
			drive.TotalCapacity = totalSize
		}

		return nil
	}); err != nil {
		return nil, err
	}
	toRet := []*direct_csi.DirectCSIDrive{}
	for _, v := range drives {
		if v.Name != v.RootPartition {
			root := drives[v.RootPartition]
			v.ModelNumber = fmt.Sprintf("%s-part%d", root.ModelNumber, v.PartitionNum)
			v.SerialNumber = fmt.Sprintf("%s-part%d", root.SerialNumber, v.PartitionNum)
			if v.BlockSize == 0 {
				v.BlockSize = root.BlockSize
				v.TotalCapacity = v.TotalCapacity * root.BlockSize
			}
		}
		v.OwnerNode = nodeID
		driveName := uuid.NewUUID().String()
		v.ObjectMeta.Name, v.Name = driveName, driveName
		toRet = append(toRet, v)
	}
	if err := findMounts(toRet); err != nil {
		return nil, err
	}
	return toRet, nil
}

func findMounts(drives []*direct_csi.DirectCSIDrive) error {
	mounts, err := os.Open("/proc/mounts")
	if err != nil {
		return err
	}
	defer mounts.Close()
	scanner := bufio.NewScanner(mounts)

	index := map[string]string{}
	for scanner.Scan() {
		line := scanner.Text()
		words := strings.SplitN(line, " ", 2)
		index[words[0]] = line
	}

	for _, drive := range drives {
		if line, ok := index[drive.Path]; ok {
			words := strings.Split(line, " ")
			if len(words) < 6 {
				// unrecognized format
				continue
			}
			drive.Mountpoint = words[1]
			drive.Filesystem = words[2]
			drive.MountOptions = strings.Split(words[3], ",")
			drive.DriveStatus = direct_csi.New
			if drive.Filesystem == "" {
				drive.DriveStatus = direct_csi.Unformatted
			}
			stat := &syscall.Statfs_t{}
			if err := syscall.Statfs(drive.Mountpoint, stat); err != nil {
				return err
			}
			availBlocks := int64(stat.Bavail)
			drive.FreeCapacity = int64(stat.Bsize) * availBlocks
		}
	}
	return nil
}

func getDriveForPath(drives map[string]*direct_csi.DirectCSIDrive, path string) *direct_csi.DirectCSIDrive {
	driveName, isRootPartition := getPartition(path)
	if driveName == "" {
		return nil
	}
	if _, ok := drives[driveName]; !ok {
		drives[driveName] = new(direct_csi.DirectCSIDrive)
	}

	if drives[driveName].Path == "" {
		drives[driveName].Path = fmt.Sprintf("/dev/%s", driveName)
	}

	if drives[driveName].Name == "" {
		drives[driveName].Name = driveName
	}

	drives[driveName].RootPartition = driveName

	if !isRootPartition {
		drives[driveName].RootPartition = getRootPartition(path)
	}

	return drives[driveName]
}

func getRootPartition(path string) string {
	cleanPath := filepath.Clean(path)
	if !strings.HasPrefix(cleanPath, "/sys/block/") {
		return ""
	}

	cleanPath = cleanPath[len("/sys/block/"):]
	cleanPathComponents := strings.SplitN(cleanPath, "/", 2)
	return cleanPathComponents[0]
}

func getPartition(path string) (string, bool) {
	isRootPartition := true
	cleanPath := filepath.Clean(path)
	if !strings.HasPrefix(cleanPath, "/sys/block/") {
		return "", isRootPartition
	}

	cleanPath = cleanPath[len("/sys/block/"):]
	cleanPathComponents := strings.SplitN(cleanPath, "/", 2)
	driveName := cleanPathComponents[0]

	if len(cleanPathComponents) == 1 {
		return driveName, isRootPartition
	}

	// if it is a partition
	if strings.HasPrefix(cleanPathComponents[1], driveName) {
		isRootPartition = false
		return strings.SplitN(cleanPathComponents[1], "/", 2)[0], isRootPartition
	}
	return driveName, isRootPartition
}

func WalkWithFollow(path string, callback func(path string, info os.FileInfo, err error) error) error {
	f, err := os.Open(path)
	defer f.Close()

	if err != nil {
		err := callback(path, nil, err)
		if err != nil {
			if err != filepath.SkipDir {
				return err
			}
		}
		return nil
	}

	stat, err := f.Stat()
	if err != nil {
		err := callback(path, nil, err)
		if err != nil {
			if err != filepath.SkipDir {
				return err
			}
		}
		return nil
	}

	if err := callback(path, stat, nil); err != nil {
		if err != filepath.SkipDir {
			return err
		}
		return nil
	}

	if stat.IsDir() {
		dirs, err := f.Readdir(0)
		if err != nil {
			return err
		}
		for _, dir := range dirs {
			if err := WalkWithFollow(filepath.Join(path, dir.Name()), callback); err != nil {
				if err != filepath.SkipDir {
					return err
				}
				return nil
			}
		}
	}
	return nil
}

// MountDevice - Utility to mount a device in the given mountpoint
func MountDevice(devicePath, mountPoint, fsType string, options []string) error {
	if err := mount.New("").Mount(devicePath, mountPoint, fsType, options); err != nil {
		glog.V(5).Info(err)
		return err
	}
	return nil
}
