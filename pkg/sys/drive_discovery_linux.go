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
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"golang.org/x/sys/unix"

	"github.com/minio/direct-csi/pkg/sys/loopback"
)

func FindDevices(ctx context.Context, loopBackOnly bool) ([]BlockDevice, error) {
	var head = func() string {
		var deviceHead = "/sys/devices"
		if loopBackOnly {
			return filepath.Join(deviceHead, "virtual", "block")
		}
		return deviceHead
	}()

	drives := []BlockDevice{}
	var attachedLoopDeviceNames []string
	if loopBackOnly {
		var err error
		attachedLoopDeviceNames, err = loopback.GetAttachedDeviceNames()
		if err != nil {
			return drives, err
		}
		if len(attachedLoopDeviceNames) == 0 {
			return drives, fmt.Errorf("No loop devices attached")
		}
	}

	return drives, filepath.Walk(head, func(path string, info os.FileInfo, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if !loopBackOnly && strings.HasPrefix(info.Name(), "loop") {
			return filepath.SkipDir
		}

		if info.Name() != "uevent" {
			return nil
		}
		drive, err := parseUevent(path)
		if err != nil {
			glog.V(5).Info(err)
			return nil
		}

		if loopBackOnly {
			if isAttachedDev := func() bool {
				for _, ldName := range attachedLoopDeviceNames {
					if ldName == drive.Devname {
						return true
					}
				}
				return false
			}(); isAttachedDev == false {
				return nil
			}
		}

		if drive.Devtype != "disk" {
			return nil
		}
		subsystem, err := subsystem(path)
		if err != nil {
			glog.V(5).Info(err)
			return nil
		}
		if subsystem != "block" {
			return nil
		}
		if err := drive.probeBlockDev(ctx); err != nil {
			glog.Errorf("Error while probing block device: %v", err)
		}

		drives = append(drives, *drive)
		return nil
	})
}

func (b *BlockDevice) GetPartitions() []Partition {
	return b.Partitions
}

func (b *BlockDevice) probeBlockDev(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			b.TagError(err)
		}
	}()

	err = os.MkdirAll(DirectCSIDevRoot, 0755)
	if err != nil {
		return err
	}

	if b.DriveInfo == nil {
		return fmt.Errorf("Invalid drive info for %s", b.Devname)
	}

	devPath := b.DirectCSIDrivePath()
	err = makeBlockFile(devPath, b.DriveInfo.Major, b.DriveInfo.Minor)
	if err != nil {
		return err
	}

	b.Path = devPath
	var logicalBlockSize, physicalBlockSize uint64
	logicalBlockSize, physicalBlockSize, err = b.getBlockSizes()
	if err != nil {
		return err
	}
	b.LogicalBlockSize = logicalBlockSize
	b.PhysicalBlockSize = physicalBlockSize

	var driveSize uint64
	driveSize, err = b.getTotalCapacity()
	if err != nil {
		return err
	}
	b.TotalCapacity = driveSize

	numBlocks := driveSize / logicalBlockSize
	b.NumBlocks = numBlocks
	b.EndBlock = numBlocks

	var parts []Partition
	parts, err = b.probePartitions(ctx)
	if err != nil {
		if err != ErrNotPartition {
			return err
		}
	}

	if len(parts) == 0 {
		offsetBlocks := uint64(0)
		var fsInfo *FSInfo
		fsInfo, err = b.probeFS(offsetBlocks)
		if err != nil {
			if err != ErrNoFS {
				return err
			}
		}
		if fsInfo == nil {
			fsInfo = &FSInfo{
				TotalCapacity: b.TotalCapacity,
				FSBlockSize:   b.LogicalBlockSize,
			}
		}
		var mounts []MountInfo
		mounts, err = b.probeMountInfo(b.DriveInfo.Major, b.DriveInfo.Minor)
		if err != nil {
			return err
		}
		fsInfo.Mounts = append(fsInfo.Mounts, mounts...)
		b.FSInfo = fsInfo
		return nil
	}
	for _, p := range parts {
		offsetBlocks := p.StartBlock
		var fsInfo *FSInfo
		fsInfo, err = b.probeFS(offsetBlocks)
		if err != nil {
			if err != ErrNoFS {
				return err
			}
		}

		if fsInfo == nil {
			fsInfo = &FSInfo{
				TotalCapacity: p.TotalCapacity,
				FSBlockSize:   p.LogicalBlockSize,
			}
		}

		var mounts []MountInfo
		mounts, err = b.probeMountInfo(p.DriveInfo.Major, p.DriveInfo.Minor)
		if err != nil {
			return err
		}
		fsInfo.Mounts = append(fsInfo.Mounts, mounts...)
		p.FSInfo = fsInfo
		b.Partitions = append(b.Partitions, p)
	}
	return nil
}

func subsystem(path string) (string, error) {
	dir := filepath.Dir(path)
	link, err := os.Readlink(filepath.Join(dir, "subsystem"))
	if err != nil {
		return "", err
	}
	return filepath.Base(link), nil
}

func parseUevent(path string) (*BlockDevice, error) {
	if filepath.Base(path) != "uevent" {
		return nil, fmt.Errorf("not a uevent file")
	}

	uevent, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	uev := string(uevent)
	var major, minor, devname, devtype string

	lines := strings.Split(uev, "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		cleanLine := strings.TrimSpace(line)
		parts := strings.Split(cleanLine, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("uevent file format not supported: %s", path)
		}
		key := parts[0]
		value := parts[1]
		switch key {
		case "MAJOR":
			major = value
		case "MINOR":
			minor = value
		case "DEVNAME":
			devname = value
		case "DEVTYPE":
			devtype = value
		default:
			return nil, fmt.Errorf("uevent file format not supported: %s", path)
		}
	}
	majorNum64, err := strconv.ParseUint(major, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid major num: %s", major)
	}
	minorNum64, err := strconv.ParseUint(minor, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid minor num: %s", minor)
	}
	majorNum := uint32(majorNum64)
	minorNum := uint32(minorNum64)

	return &BlockDevice{
		Devname: devname,
		Devtype: devtype,
		DriveInfo: &DriveInfo{
			Major: majorNum,
			Minor: minorNum,
		},
	}, nil
}

func (b *BlockDevice) getBlockSizes() (uint64, uint64, error) {
	devFile, err := os.OpenFile(b.DirectCSIDrivePath(), os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return 0, 0, err
	}
	defer devFile.Close()

	fd := devFile.Fd()
	logicalBlockSize, err := unix.IoctlGetInt(int(fd), unix.BLKSSZGET)
	if err != nil {
		glog.Errorf("could not obtain logical block size for device: %s", b.Devname)
		return 0, 0, err
	}
	physicalBlockSize, err := unix.IoctlGetInt(int(fd), unix.BLKBSZGET)
	if err != nil {
		glog.Errorf("could not obtain physical block size for device: %s", b.Devname)
		return 0, 0, err
	}
	return uint64(logicalBlockSize), uint64(physicalBlockSize), nil
}

func (b *BlockDevice) getTotalCapacity() (uint64, error) {
	devFile, err := os.OpenFile(b.DirectCSIDrivePath(), os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return 0, err
	}
	defer devFile.Close()

	driveSize, err := devFile.Seek(0, os.SEEK_END)
	if err != nil {
		return 0, err
	}
	return uint64(driveSize), nil
}
