//go:build linux

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

package sys

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/minio/directpv/pkg/blockdev"
	"github.com/minio/directpv/pkg/blockdev/parttable"
	"github.com/minio/directpv/pkg/fs"
	fserrors "github.com/minio/directpv/pkg/fs/errors"
	"github.com/minio/directpv/pkg/mount"
	"github.com/minio/directpv/pkg/sys/smart"
	"golang.org/x/sys/unix"
	"k8s.io/klog/v2"
)

const (
	defaultBlockSize = 512
)

func getDeviceMajorMinor(device string) (major, minor uint32, err error) {
	stat := syscall.Stat_t{}
	if err = syscall.Stat(device, &stat); err == nil {
		major, minor = uint32(unix.Major(stat.Rdev)), uint32(unix.Minor(stat.Rdev))
	}
	return
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

func getMajorMinor(name string) (major int, minor int, err error) {
	var majorMinor string
	if majorMinor, err = readFirstLine("/sys/class/block/"+name+"/dev", true); err != nil {
		return
	}

	tokens := strings.SplitN(majorMinor, ":", 2)
	if len(tokens) != 2 {
		err = fmt.Errorf("unknown format of %v", majorMinor)
		return
	}

	if major, err = strconv.Atoi(tokens[0]); err != nil {
		return
	}
	minor, err = strconv.Atoi(tokens[1])
	return
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

func getPartition(name string) (int, error) {
	s, err := readFirstLine("/sys/class/block/"+name+"/partition", false)
	if err != nil {
		return 0, err
	}
	if s != "" {
		return strconv.Atoi(s)
	}
	return 0, nil
}

func getRemovable(name string) (bool, error) {
	s, err := readFirstLine("/sys/class/block/"+name+"/removable", false)
	return s != "" && s != "0", err
}

func getReadOnly(name string) (bool, error) {
	s, err := readFirstLine("/sys/class/block/"+name+"/ro", false)
	return s != "" && s != "0", err
}

func getWWID(name string) (wwid string, err error) {
	if wwid, err = readFirstLine("/sys/class/block/"+name+"/wwid", false); err == nil && wwid == "" {
		wwid, err = readFirstLine("/sys/class/block/"+name+"/device/wwid", false)
	}
	return wwid, err
}

func getModel(name string) (string, error) {
	return readFirstLine("/sys/class/block/"+name+"/device/model", false)
}

func getSerial(name string) (string, error) {
	serial, _ := smart.GetSerialNumber("/dev/" + name)
	return serial, nil
}

func getVendor(name string) (string, error) {
	return readFirstLine("/sys/class/block/"+name+"/device/vendor", false)
}

func getDMName(name string) (string, error) {
	return readFirstLine("/sys/class/block/"+name+"/dm/name", false)
}

func getDMUUID(name string) (string, error) {
	return readFirstLine("/sys/class/block/"+name+"/dm/uuid", false)
}

func getMDUUID(name string) (string, error) {
	uuid, err := readFirstLine("/sys/class/block/"+name+"/md/uuid", false)
	return NormalizeUUID(uuid), err
}

func getVirtual(name string) (bool, error) {
	absPath, err := filepath.EvalSymlinks("/sys/class/block/" + name)
	if err != nil {
		return false, err
	}
	return strings.HasPrefix(absPath, "/sys/devices/virtual/block/"), nil
}

func getHidden(name string) bool {
	// errors ignored since real devices do not have <sys>/hidden
	// borrow idea from 'lsblk'
	// https://github.com/util-linux/util-linux/commit/c8487d854ba5cf5bfcae78d8e5af5587e7622351
	v, _ := readFirstLine("/sys/class/block/"+name+"/hidden", false)
	return v == "1"
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

func getSlaves(name string) ([]string, error) {
	return readdirnames("/sys/block/"+name+"/slaves", false)
}

func readSysBlock() ([]string, error) {
	return readdirnames("/sys/block", true)
}

func readSysClassBlock() ([]string, error) {
	return readdirnames("/sys/class/block", true)
}

func probeDevice(name string) (device *Device, err error) {
	device = &Device{Name: name}
	device.Hidden = getHidden(name)
	// hidden devices do not have major,minor value.
	if !device.Hidden {
		device.Major, device.Minor, err = getMajorMinor(name)
		if err != nil {
			return nil, err
		}
	}
	if device.Size, err = getSize(name); err != nil {
		return nil, err
	}
	// hidden devices do not have parititions.
	if !device.Hidden {
		if device.Partition, err = getPartition(name); err != nil {
			return nil, err
		}
	}
	if device.Removable, err = getRemovable(name); err != nil {
		return nil, err
	}
	if device.ReadOnly, err = getReadOnly(name); err != nil {
		return nil, err
	}
	if device.WWID, err = getWWID(name); err != nil {
		return nil, err
	}
	if device.Model, err = getModel(name); err != nil {
		return nil, err
	}
	if device.Serial, err = getSerial(name); err != nil {
		return nil, err
	}
	if device.Vendor, err = getVendor(name); err != nil {
		return nil, err
	}
	// hidden devices do not have dmname, dmuuid, mduuid and virtual links.
	if !device.Hidden {
		if device.DMName, err = getDMName(name); err != nil {
			return nil, err
		}
		if device.DMUUID, err = getDMUUID(name); err != nil {
			return nil, err
		}
		if device.MDUUID, err = getMDUUID(name); err != nil {
			return nil, err
		}
		if device.Virtual, err = getVirtual(name); err != nil {
			return nil, err
		}
	}
	return device, nil
}

func getAllDevices() (devices map[string]*Device, err error) {
	var names []string
	if names, err = readSysClassBlock(); err != nil {
		return nil, err
	}

	var device *Device
	devices = map[string]*Device{}
	for _, name := range names {
		if device, err = probeDevice(name); err != nil {
			return nil, err
		}
		devices[name] = device
	}

	return devices, nil
}

func probePartTable(name string) (parttable.PartTable, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	return blockdev.Probe(ctx, "/dev/"+name)
}

func updatePartTableInfo(devices map[string]*Device) error {
	names, err := readSysBlock()
	if err != nil {
		return err
	}

	for _, name := range names {
		device, ok := devices[name]
		if !ok {
			klog.V(3).Infof("device name %s present in /sys/block missing from /sys/class/block probe, ignoring", name)
			continue
		}

		if device.Hidden {
			// No partitions for hidden devices.
			continue
		}

		partTable, err := probePartTable(name)
		if devices[name].Size > 0 && err != nil {
			switch {
			case errors.Is(err, parttable.ErrPartTableNotFound):
			case strings.Contains(strings.ToLower(err.Error()), "no medium found"):
			default:
				return err
			}
		}

		var partitionMap map[int]*parttable.Partition
		if partTable != nil {
			devices[name].PTUUID = partTable.UUID()
			devices[name].PTType = partTable.Type()
			partitionMap = partTable.Partitions()
		}

		partitions, err := getPartitions(name)
		if err != nil {
			return err
		}
		devices[name].Partitioned = len(partitions) > 0
		for _, partition := range partitions {
			devices[partition].Parent = name

			partNumber := devices[partition].Partition
			if partitionMap != nil {
				if _, found := partitionMap[partNumber]; found {
					devices[name].PartUUID = partitionMap[partNumber].UUID
				}
			}
		}

		slaves, err := getSlaves(name)
		if err != nil {
			return err
		}
		for _, slave := range slaves {
			devices[slave].Master = name
		}
	}

	return nil
}

func probeDevicesFromSysfs() (devices map[string]*Device, err error) {
	if devices, err = getAllDevices(); err != nil {
		return nil, err
	}

	if err = updatePartTableInfo(devices); err != nil {
		return nil, err
	}

	return devices, nil
}

func NewDevice(udevData *UDevData) (device *Device, err error) {
	name := filepath.Base(udevData.Path)
	if name == "" {
		return nil, fmt.Errorf("udevData does not have valid DEVPATH %v", udevData.Path)
	}
	device = &Device{
		Name:         name,
		Major:        udevData.Major,
		Minor:        udevData.Minor,
		Virtual:      strings.Contains(udevData.Path, "/virtual/"),
		Partition:    udevData.Partition,
		WWID:         udevData.WWID,
		Model:        udevData.Model,
		UeventSerial: udevData.UeventSerial,
		Vendor:       udevData.Vendor,
		DMName:       udevData.DMName,
		DMUUID:       udevData.DMUUID,
		MDUUID:       udevData.MDUUID,
		PTUUID:       udevData.PTUUID,
		PTType:       udevData.PTUUID,
		PartUUID:     udevData.PartUUID,
		UeventFSUUID: udevData.UeventFSUUID,
		FSType:       udevData.FSType,
		FSUUID:       udevData.FSUUID,
	}

	return device, nil
}

func updateRelationship(devices map[string]*Device) error {
	names, err := readSysBlock()
	if err != nil {
		return err
	}

	for _, name := range names {
		device, ok := devices[name]
		if !ok {
			klog.V(3).Infof("device name %s present in /sys/block missing from udev probe, ignoring", name)
			continue
		}

		if device.Hidden {
			// No partitions for hidden devices.
			continue
		}

		partitions, err := getPartitions(name)
		if err != nil {
			return err
		}

		devices[name].Partitioned = len(partitions) > 0
		for _, partition := range partitions {
			devices[partition].Parent = name
		}

		slaves, err := getSlaves(name)
		if err != nil {
			return err
		}
		for _, slave := range slaves {
			devices[slave].Master = name
		}
	}

	return nil
}

func probeDevicesFromUdev() (devices map[string]*Device, err error) {
	var names []string
	if names, err = readSysClassBlock(); err != nil {
		return nil, err
	}

	devices = map[string]*Device{}
	for _, name := range names {
		hidden := getHidden(name)
		var device *Device
		if !hidden {
			major, minor, err := getMajorMinor(name)
			if err != nil {
				return nil, err
			}

		udevData, err := readRunUdevData(major, minor)
		if err != nil {
			return nil, err
		}

		device, err := NewDevice(udevData)
		if err != nil {
			return nil, err
		}

		if device.Virtual, err = getVirtual(name); err != nil {
			return nil, err
		}

		if device.Serial, err = getSerial(device.Name); err != nil {
			return nil, err
		}

		if device.Size, err = getSize(name); err != nil {
			return nil, err
		}

		if device.Removable, err = getRemovable(name); err != nil {
			return nil, err
		}
		if device.ReadOnly, err = getReadOnly(name); err != nil {
			return nil, err
		}

		devices[name] = device
	}

	if err = updateRelationship(devices); err != nil {
		return nil, err
	}

	return devices, nil
}

func getBlockSizes(device string) (physicalBlockSize, logicalBlockSize uint64, err error) {
	devFile, err := os.OpenFile(device, os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return
	}
	defer devFile.Close()
	fd := devFile.Fd()

	var blockSize int
	if blockSize, err = unix.IoctlGetInt(int(fd), unix.BLKBSZGET); err != nil {
		klog.Errorf("could not obtain physical block size for device: %s", device)
		return
	}
	physicalBlockSize = uint64(blockSize)

	if blockSize, err = unix.IoctlGetInt(int(fd), unix.BLKSSZGET); err != nil {
		klog.Errorf("could not obtain logical block size for device: %s", device)
		return
	}
	logicalBlockSize = uint64(blockSize)

	return
}

func probeFS(device *Device) (fs.FS, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	fsInfo, err := fs.Probe(ctx, "/dev/"+device.Name)
	if err != nil && device.Size > 0 {
		switch {
		case errors.Is(err, fserrors.ErrFSNotFound), errors.Is(err, fserrors.ErrCanceled), errors.Is(err, io.ErrUnexpectedEOF):
		default:
			return nil, err
		}
	}
	return fsInfo, nil
}

func getCapacity(device *Device) (totalCapacity, freeCapacity uint64) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	var err error
	totalCapacity, freeCapacity, err = fs.GetCapacity(ctx, "/dev/"+device.Name, device.FSType)
	if err != nil {
		klog.V(5).InfoS("unable to get device capacity", "err", err, "Device", device.Name, "FSType", device.FSType)
	}
	return
}

func updateFSInfo(device *Device, CDROMs, swaps map[string]struct{}, mountInfos map[string][]mount.MountInfo, mountPointsMap map[string][]string) error {
	if _, found := CDROMs[device.Name]; found {
		device.ReadOnly = true
		device.Removable = true
		return nil
	}

	majorMinor := fmt.Sprintf("%v:%v", device.Major, device.Minor)
	if _, found := swaps[majorMinor]; !found {
		device.MountPoints = mountPointsMap[majorMinor]
		if len(device.MountPoints) > 0 {
			device.FirstMountPoint = mountInfos[majorMinor][0].MountPoint
			device.FirstMountOptions = mountInfos[majorMinor][0].MountOptions
		}

		var err error
		if device.PhysicalBlockSize, device.LogicalBlockSize, err = getBlockSizes("/dev/" + device.Name); device.Size > 0 && err != nil {
			return err
		}
	} else {
		device.SwapOn = true
	}

	if device.FSType == "" {
		fsInfo, err := probeFS(device)
		if err != nil {
			return err
		}
		if fsInfo != nil {
			device.FSUUID = fsInfo.ID()
			if device.FSType != "" && !FSTypeEqual(device.FSType, fsInfo.Type()) {
				klog.Errorf("%v: FSType %v from Uevent does not match probed FSType %v", "/dev/"+device.Name, device.FSType, fsInfo.Type())
				device.TotalCapacity, device.FreeCapacity = getCapacity(device)
			} else {
				device.FSType = fsInfo.Type()
				device.TotalCapacity = fsInfo.TotalCapacity()
				device.FreeCapacity = fsInfo.FreeCapacity()
			}
		}
	}
	return nil
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

func getMountPoints(mountInfos map[string][]mount.MountInfo) (map[string][]string, error) {
	mountPointsMap := map[string][]string{}
	for _, mounts := range mountInfos {
		for _, mount := range mounts {
			mountPointsMap[mount.MajorMinor] = append(mountPointsMap[mount.MajorMinor], mount.MountPoint)
		}
	}

	return mountPointsMap, nil
}

func probeDevices() (devices map[string]*Device, err error) {
	if isUdevDataReadable() {
		devices, err = probeDevicesFromUdev()
	} else {
		devices, err = probeDevicesFromSysfs()
	}
	if err != nil {
		return nil, err
	}

	CDROMs, err := getCDROMs()
	if err != nil {
		return nil, err
	}

	swaps, err := getSwaps()
	if err != nil {
		return nil, err
	}

	mountInfos, err := mount.Probe()
	if err != nil {
		return nil, err
	}

	mountPointsMap, err := getMountPoints(mountInfos)
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		if device.Hidden {
			// No FS information needed for hidden devices
			continue
		}
		if err = updateFSInfo(device, CDROMs, swaps, mountInfos, mountPointsMap); err != nil {
			return nil, err
		}
	}

	return devices, nil
}

func (device *Device) ProbeHostInfo() (err error) {
	if device.Serial, err = getSerial(device.Name); err != nil {
		return err
	}

	if device.Removable, err = getRemovable(device.Name); err != nil {
		return err
	}

	if device.ReadOnly, err = getReadOnly(device.Name); err != nil {
		return err
	}

	if device.Size, err = getSize(device.Name); err != nil {
		return err
	}

	if device.Partition <= 0 {
		names, err := getPartitions(device.Name)
		if err != nil {
			return err
		}
		device.Partitioned = len(names) > 0
	}

	CDROMs, err := getCDROMs()
	if err != nil {
		return err
	}

	mountInfos, err := mount.Probe()
	if err != nil {
		return err
	}

	mountPointsMap, err := getMountPoints(mountInfos)
	if err != nil {
		return err
	}

	swaps, err := getSwaps()
	if err != nil {
		return err
	}

	if err = updateFSInfo(device, CDROMs, swaps, mountInfos, mountPointsMap); err != nil {
		return err
	}

	return nil
}

func createDevice(udevData *UDevData) (device *Device, err error) {
	if device, err = NewDevice(udevData); err != nil {
		return nil, err
	}
	if err := device.ProbeHostInfo(); err != nil {
		return nil, err
	}
	return device, nil
}

func getDeviceName(major, minor uint32) (string, error) {
	filename := fmt.Sprintf("/sys/dev/block/%v:%v/uevent", major, minor)
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

		if !strings.HasPrefix(s, "DEVNAME=") {
			continue
		}

		switch tokens := strings.SplitN(s, "=", 2); len(tokens) {
		case 2:
			return strings.TrimSpace(tokens[1]), nil
		default:
			return "", fmt.Errorf("filename %v contains invalid DEVNAME value", filename)
		}
	}
}
