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

package rest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/minio/directpv/pkg/fs"
	"github.com/minio/directpv/pkg/fs/xfs"
	"github.com/minio/directpv/pkg/mount"
	"github.com/minio/directpv/pkg/sys"
	losetup "gopkg.in/freddierice/go-losetup.v1"

	"k8s.io/klog/v2"
)

var errMountFailure = errors.New("could not mount the drive")

// nodeAPIHandlers provides HTTP handlers for DirectPV node API.
type nodeAPIHandler struct {
	nodeID                      string
	reflinkSupport              bool
	getDevice                   func(major, minor uint32) (string, error)
	stat                        func(name string) (os.FileInfo, error)
	mountDevice                 func(device, target string, flags []string) error
	unmountDevice               func(device string) error
	makeFS                      func(ctx context.Context, device, uuid string, force, reflink bool) error
	isMounted                   func(target string) (bool, error)
	safeUnmount                 func(target string, force, detach, expire bool) error
	truncate                    func(name string, size int64) error
	attachLoopDevice            func(backingFile string, offset uint64, ro bool) (losetup.Device, error)
	readRunUdevDataByMajorMinor func(major, minor int) (map[string]string, error)
	probeFS                     func(ctx context.Context, device string) (fs fs.FS, err error)
	// locks
	formatLockerMutex sync.Mutex
	formatLocker      map[string]*sync.Mutex
}

func newNodeAPIHandler(ctx context.Context, nodeID string) (*nodeAPIHandler, error) {
	var err error
	getDevice := func(major, minor uint32) (string, error) {
		name, err := sys.GetDeviceName(major, minor)
		if err != nil {
			return "", err
		}
		return "/dev/" + name, nil
	}
	nodeAPIHandler := &nodeAPIHandler{
		nodeID:                      nodeID,
		getDevice:                   getDevice,
		stat:                        os.Stat,
		mountDevice:                 mount.MountXFSDevice,
		unmountDevice:               mount.UnmountDevice,
		makeFS:                      xfs.MakeFS,
		isMounted:                   mount.IsMounted,
		safeUnmount:                 mount.SafeUnmount,
		truncate:                    os.Truncate,
		attachLoopDevice:            losetup.Attach,
		readRunUdevDataByMajorMinor: sys.ReadRunUdevDataByMajorMinor,
		probeFS:                     fs.Probe,
		formatLocker:                map[string]*sync.Mutex{},
	}
	nodeAPIHandler.reflinkSupport, err = nodeAPIHandler.isReflinkSupported(ctx)
	if err != nil {
		return nil, err
	}
	return nodeAPIHandler, nil
}

func (n *nodeAPIHandler) getFormatLock(major, minor int) *sync.Mutex {
	n.formatLockerMutex.Lock()
	defer n.formatLockerMutex.Unlock()

	key := fmt.Sprintf("%d:%d", major, minor)
	if _, found := n.formatLocker[key]; !found {
		n.formatLocker[key] = &sync.Mutex{}
	}

	return n.formatLocker[key]
}

func (n *nodeAPIHandler) isReflinkSupported(ctx context.Context) (bool, error) {
	var reflinkSupport bool
	// trying with reflink enabled
	if err := n.checkXFS(ctx, true); err == nil {
		reflinkSupport = true
		klog.V(3).Infof("enabled reflink while formatting")
	} else {
		if !errors.Is(err, errMountFailure) {
			return reflinkSupport, err
		}
		// trying with reflink disabled
		if err := n.checkXFS(ctx, false); err != nil {
			return reflinkSupport, err
		}
		reflinkSupport = false
		klog.V(3).Infof("disabled reflink while formatting")
	}
	return reflinkSupport, nil
}

func (n *nodeAPIHandler) checkXFS(ctx context.Context, reflinkSupport bool) error {
	mountPoint, err := os.MkdirTemp("", "xfs.check.mnt.")
	if err != nil {
		return err
	}
	defer os.Remove(mountPoint)

	file, err := os.CreateTemp("", "xfs.check.file.")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	file.Close()

	if err = n.truncate(file.Name(), sys.MinSupportedDeviceSize); err != nil {
		return err
	}

	if err = n.makeFS(ctx, file.Name(), uuid.New().String(), false, reflinkSupport); err != nil {
		klog.V(3).ErrorS(err, "failed to format", "reflink", reflinkSupport)
		return err
	}

	loopDevice, err := n.attachLoopDevice(file.Name(), 0, false)
	if err != nil {
		return err
	}

	defer func() {
		if err := loopDevice.Detach(); err != nil {
			klog.Error(err)
		}
	}()

	if err = n.mountDevice(loopDevice.Path(), mountPoint, []string{}); err != nil {
		klog.V(3).ErrorS(err, "failed to mount", "reflink", reflinkSupport)
		return errMountFailure
	}

	return n.unmountDevice(mountPoint)
}
