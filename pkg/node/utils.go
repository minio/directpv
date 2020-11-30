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
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"

	direct_csi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	simd "github.com/minio/sha256-simd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sExec "k8s.io/utils/exec"
)

func FindDrives(ctx context.Context, nodeID string, procfs string) ([]*direct_csi.DirectCSIDrive, error) {
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
			if drive.Status.SerialNumber == "" {
				data, err := ioutil.ReadFile(link)
				if err != nil {
					return err
				}
				drive.Status.SerialNumber = strings.TrimSpace(string(data))
			}
		}
		if strings.Compare(info.Name(), "model") == 0 {
			if drive.Status.ModelNumber == "" {
				data, err := ioutil.ReadFile(link)
				if err != nil {
					return err
				}
				drive.Status.ModelNumber = strings.TrimSpace(string(data))
			}
		}
		if strings.Compare(info.Name(), "partition") == 0 {
			// not needed if this is a root partition
			if drive.Name == drive.Status.RootPartition {
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
			drive.Status.PartitionNum = partNum
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
			if drive.Status.BlockSize != 0 {
				blockSize = drive.Status.BlockSize
			}
			size = size * blockSize
			drive.Status.TotalCapacity = size
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
			if drive.Status.TotalCapacity != 0 {
				blocks = drive.Status.TotalCapacity
			}
			drive.Status.BlockSize = size
			totalSize := size * blocks
			drive.Status.TotalCapacity = totalSize
		}

		return nil
	}); err != nil {
		return nil, err
	}
	toRet := []*direct_csi.DirectCSIDrive{}
	for _, v := range drives {
		if v.Name != v.Status.RootPartition {
			root := drives[v.Status.RootPartition]
			if root.Status.ModelNumber != "" {
				v.Status.ModelNumber = fmt.Sprintf("%s-part%d", root.Status.ModelNumber, v.Status.PartitionNum)
			}
			if root.Status.SerialNumber != "" {
				v.Status.SerialNumber = fmt.Sprintf("%s-part%d", root.Status.SerialNumber, v.Status.PartitionNum)
			}
			if v.Status.BlockSize == 0 {
				v.Status.BlockSize = root.Status.BlockSize
				v.Status.TotalCapacity = v.Status.TotalCapacity * root.Status.BlockSize
			}
		}
		v.Status.NodeName = nodeID
		driveName := strings.Join([]string{nodeID, v.Status.Path}, "-")
		driveName = fmt.Sprintf("%x", simd.Sum256([]byte(driveName)))

		v.ObjectMeta.Name, v.Name = driveName, driveName
		toRet = append(toRet, v)
	}
	if err := findMounts(toRet, procfs); err != nil {
		return nil, err
	}
	return toRet, nil
}

func findMounts(drives []*direct_csi.DirectCSIDrive, procfs string) error {
	procMounts := filepath.Join(procfs, "mounts")
	mounts, err := os.Open(procMounts)
	if err != nil {
		return err
	}
	defer mounts.Close()
	scanner := bufio.NewScanner(mounts)

	index := map[string]string{}
	for scanner.Scan() {
		line := scanner.Text()
		words := strings.SplitN(line, " ", 2)
		if _, ok := index[words[0]]; !ok {
			index[words[0]] = line
		}
	}

	for _, drive := range drives {
		if line, ok := index[drive.Status.Path]; ok {
			words := strings.Split(line, " ")
			if len(words) < 6 {
				// unrecognized format
				continue
			}
			drive.Status.Mountpoint = words[1]
			drive.Status.Filesystem = words[2]
			drive.Status.MountOptions = strings.Split(words[3], ",")
			drive.Status.DriveStatus = direct_csi.New
			if drive.Status.Filesystem == "" {
				drive.Status.DriveStatus = direct_csi.Unformatted
			}
			stat := &syscall.Statfs_t{}
			if err := syscall.Statfs(drive.Status.Mountpoint, stat); err != nil {
				return err
			}
			availBlocks := int64(stat.Bavail)
			drive.Status.FreeCapacity = int64(stat.Bsize) * availBlocks
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

	if drives[driveName].Status.Path == "" {
		drives[driveName].Status.Path = fmt.Sprintf("/dev/%s", driveName)
	}

	if drives[driveName].Name == "" {
		drives[driveName].Name = driveName
	}

	drives[driveName].Status.RootPartition = driveName

	if !isRootPartition {
		drives[driveName].Status.RootPartition = getRootPartition(path)
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
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return err
	}
	if err := mount.New("").Mount(devicePath, mountPoint, fsType, options); err != nil {
		glog.V(5).Info(err)
		return err
	}
	return nil
}

// FormatDevice - Formats the given device
func FormatDevice(ctx context.Context, source, fsType string, force bool) error {
	args := []string{source}
	forceFlag := "-F"
	if fsType == "xfs" {
		forceFlag = "-f"
	}
	if force {
		args = []string{
			forceFlag, // Force flag
			source,
		}
	}
	glog.V(5).Infof("args: %v", args)
	output, err := exec.CommandContext(ctx, "mkfs."+fsType, args...).CombinedOutput()
	if err != nil {
		glog.V(5).Infof("Failed to format the device: err: (%s) output: (%s)", err.Error(), string(output))
	}
	return err
}

// UnmountIfMounted - Idempotent function to unmount a target
func UnmountIfMounted(mountPoint string) error {
	shouldUmount := false
	mountPoints, mntErr := mount.New("").List()
	if mntErr != nil {
		return mntErr
	}
	for _, mp := range mountPoints {
		abPath, _ := filepath.Abs(mp.Path)
		if mountPoint == abPath {
			shouldUmount = true
			break
		}
	}
	if shouldUmount {
		if mErr := mount.New("").Unmount(mountPoint); mErr != nil {
			return mErr
		}
	}
	return nil
}

// GetLatestStatus gets the latest condition by time
func GetLatestStatus(statusXs []metav1.Condition) metav1.Condition {
	// Sort the drives by LastTransitionTime [Descending]
	sort.SliceStable(statusXs, func(i, j int) bool {
		return (&statusXs[j].LastTransitionTime).Before(&statusXs[i].LastTransitionTime)
	})
	return statusXs[0]
}

// GetDiskFS - To get the filesystem of a block device
func GetDiskFS(devicePath string) (string, error) {
	diskMounter := &mount.SafeFormatAndMount{Interface: mount.New(""), Exec: k8sExec.New()}
	// Internally uses 'blkid' to see if the given disk is unformatted
	fs, err := diskMounter.GetDiskFormat(devicePath)
	if err != nil {
		glog.V(5).Infof("Error while reading the disk format: (%s)", err.Error())
	}
	return fs, err
}
