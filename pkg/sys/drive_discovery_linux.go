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
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
	"k8s.io/klog"

	"github.com/minio/direct-csi/pkg/sys/loopback"
)

func readFirstLine(filename string, ignoreNotExist bool) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		if ignoreNotExist && errors.Is(err, os.ErrNotExist) {
			err = nil
		}
		return "", err
	}
	defer file.Close()
	s, err := bufio.NewReader(file).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(s), nil
}

type drive struct {
	name      string // from "/sys/class/block"
	major     int    // from "/sys/class/block/${name}/dev"
	minor     int    // from "/sys/class/block/${name}/dev"
	partition int    // from "/sys/class/block/${name}/partition"
	dmName    string // from "/sys/class/block/${name}/dm/name"
	dmUUID    string // from "/sys/class/block/${name}/dm/uuid"
	parent    string // computed
	master    string // computed
}

func getDevMajorMinor(name string) (major int, minor int, err error) {
	var dev string
	if dev, err = readFirstLine("/sys/class/block/"+name+"/dev", false); err != nil {
		return
	}

	tokens := strings.SplitN(dev, ":", 2)
	if len(tokens) != 2 {
		err = fmt.Errorf("unknown format of %v", dev)
		return
	}

	if major, err = strconv.Atoi(tokens[0]); err != nil {
		return
	}
	minor, err = strconv.Atoi(tokens[1])
	return
}

func getPartition(name string) (int, error) {
	s, err := readFirstLine("/sys/class/block/"+name+"/partition", true)
	if err != nil {
		return 0, err
	}
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}

func getDMName(name string) (string, error) {
	return readFirstLine("/sys/class/block/"+name+"/dm/name", true)
}

func getDMUUID(name string) (string, error) {
	return readFirstLine("/sys/class/block/"+name+"/dm/uuid", true)
}

func getDrive(name string) (*drive, error) {
	major, minor, err := getDevMajorMinor(name)
	if err != nil {
		return nil, err
	}
	partition, err := getPartition(name)
	if err != nil {
		return nil, err
	}
	dmName, err := getDMName(name)
	if err != nil {
		return nil, err
	}
	dmUUID, err := getDMUUID(name)
	if err != nil {
		return nil, err
	}
	return &drive{
		name:      name,
		major:     major,
		minor:     minor,
		partition: partition,
		dmName:    dmName,
		dmUUID:    dmUUID,
	}, nil
}

func getParttiions(name string) ([]string, error) {
	file, err := os.Open("/sys/block/" + name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	names, err := file.Readdirnames(-1)
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

func getSlaves(name string) ([]string, error) {
	file, err := os.Open("/sys/block/" + name + "/slaves")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = nil
		}
		return nil, err
	}
	defer file.Close()
	return file.Readdirnames(-1)
}

func readSysBlock() ([]string, error) {
	file, err := os.Open("/sys/block")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return file.Readdirnames(-1)
}

func readSysClassBlock() ([]string, error) {
	file, err := os.Open("/sys/class/block")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return file.Readdirnames(-1)
}

func probeDrives() (map[string]*drive, error) {
	names, err := readSysClassBlock()
	if err != nil {
		return nil, err
	}

	driveMap := map[string]*drive{}
	for _, name := range names {
		drive, err := getDrive(name)
		if err != nil {
			return nil, err
		}
		driveMap[name] = drive
	}

	if names, err = readSysBlock(); err != nil {
		return nil, err
	}

	for _, name := range names {
		partitions, err := getParttiions(name)
		if err != nil {
			return nil, err
		}
		for _, partition := range partitions {
			if _, found := driveMap[partition]; found {
				driveMap[partition].parent = name
			}
		}

		slaves, err := getSlaves(name)
		if err != nil {
			return nil, err
		}
		for _, slave := range slaves {
			if _, found := driveMap[slave]; found {
				driveMap[slave].master = name
			}
		}
	}

	return driveMap, nil
}

func FindDevices(ctx context.Context, loopBackOnly bool) ([]BlockDevice, error) {
	driveMap, err := probeDrives()
	if err != nil {
		return nil, err
	}

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
			klog.V(5).Info(err)
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
			klog.V(5).Info(err)
			return nil
		}
		if subsystem != "block" {
			return nil
		}
		if err := drive.probeBlockDev(ctx, driveMap); err != nil {
			klog.Errorf("Error while probing block device: %v", err)
		}

		drives = append(drives, *drive)
		return nil
	})
}

func (b *BlockDevice) GetPartitions() []Partition {
	return b.Partitions
}

func (b *BlockDevice) probeBlockDev(ctx context.Context, driveMap map[string]*drive) (err error) {
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

	b.DMName = driveMap[b.Devname].dmName
	b.DMUUID = driveMap[b.Devname].dmUUID
	b.Parent = driveMap[b.Devname].parent
	b.Master = driveMap[b.Devname].master
	for i := range parts {
		for name, drive := range driveMap {
			if strings.HasPrefix(name, b.Devname) && drive.parent == b.Devname && drive.partition == int(parts[i].PartitionNum) {
				parts[i].DMName = drive.dmName
				parts[i].DMUUID = drive.dmUUID
				parts[i].Parent = drive.parent
				parts[i].Master = drive.master
			}
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
		mounts, err = b.probeMountInfo(b.DriveInfo.Major, b.DriveInfo.Minor, driveMap)
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
		mounts, err = b.probeMountInfo(p.DriveInfo.Major, p.DriveInfo.Minor, driveMap)
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
		klog.Errorf("could not obtain logical block size for device: %s", b.Devname)
		return 0, 0, err
	}
	physicalBlockSize, err := unix.IoctlGetInt(int(fd), unix.BLKBSZGET)
	if err != nil {
		klog.Errorf("could not obtain physical block size for device: %s", b.Devname)
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
