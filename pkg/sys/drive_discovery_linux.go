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
)

func FindDevices(ctx context.Context) ([]BlockDevice, error) {
	const head = "/sys/devices"
	drives := []BlockDevice{}

	return drives, filepath.Walk(head, func(path string, info os.FileInfo, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if strings.HasPrefix(info.Name(), "loop") {
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
			glog.Errorf("could not get device information: %v", err)
			return nil
		}

		drives = append(drives, *drive)
		return nil
	})
}

func (b *BlockDevice) GetPartitions() []Partition {
	return b.Partitions
}

func (b *BlockDevice) probeBlockDev(ctx context.Context) error {
	if err := os.MkdirAll(DirectCSIDevRoot, 0755); err != nil {
		return err
	}

	devPath := getBlockFile(b.Devname)
	if err := makeBlockFile(devPath, b.Major, b.Minor); err != nil {
		return err
	}

	b.DriveInfo = &DriveInfo{}
	b.Path = devPath
	logicalBlockSize, physicalBlockSize, err := getBlockSizes(b.Devname)
	if err != nil {
		return err
	}
	b.LogicalBlockSize = logicalBlockSize
	b.PhysicalBlockSize = physicalBlockSize

	driveSize, err := getTotalCapacity(b.Devname)
	if err != nil {
		return err
	}
	b.TotalCapacity = driveSize

	numBlocks := driveSize / logicalBlockSize
	b.NumBlocks = numBlocks
	b.EndBlock = numBlocks

	parts, err := b.probePartitions(ctx)
	if err != nil {
		if err != ErrNotPartition {
			return err
		}
	}

	if len(parts) == 0 {
		offsetBlocks := uint64(0)
		fsInfo, err := b.probeFS(offsetBlocks)
		if err != nil {
			if err != ErrNoFS {
				return err
			}
		}
		if fsInfo == nil {
			fsInfo = &FSInfo{}
		}
		mounts, err := b.probeMountInfo(0)
		if err != nil {
			return err
		}
		fsInfo.Mounts = append(fsInfo.Mounts, mounts...)
		b.FSInfo = fsInfo
		return nil
	}
	for i, p := range parts {
		offsetBlocks := p.StartBlock
		fsInfo, err := b.probeFS(offsetBlocks)
		if err != nil {
			if err != ErrNoFS {
				return err
			}
		}

		if fsInfo == nil {
			fsInfo = &FSInfo{}
		}

		mounts, err := b.probeMountInfo(uint(i + 1))
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
		Major:   majorNum,
		Minor:   minorNum,
		Devname: devname,
		Devtype: devtype,
	}, nil
}

func getBlockSizes(devname string) (uint64, uint64, error) {
	devFile, err := os.OpenFile(getBlockFile(devname), os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return 0, 0, err
	}
	defer devFile.Close()

	fd := devFile.Fd()
	logicalBlockSize, err := unix.IoctlGetInt(int(fd), unix.BLKSSZGET)
	if err != nil {
		glog.Errorf("could not obtain logical block size for device: %s", devname)
		return 0, 0, err
	}
	physicalBlockSize, err := unix.IoctlGetInt(int(fd), unix.BLKBSZGET)
	if err != nil {
		glog.Errorf("could not obtain physical block size for device: %s", devname)
		return 0, 0, err
	}
	return uint64(logicalBlockSize), uint64(physicalBlockSize), nil
}

func getTotalCapacity(devname string) (uint64, error) {
	devFile, err := os.OpenFile(getBlockFile(devname), os.O_RDONLY, os.ModeDevice)
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
