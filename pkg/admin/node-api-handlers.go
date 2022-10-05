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

package admin

import (
	"context"
	"errors"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/minio/directpv/pkg/device"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/xfs"
	losetup "gopkg.in/freddierice/go-losetup.v1"

	"k8s.io/klog/v2"
)

// nodeAPIHandlers provides HTTP handlers for DirectPV node API.
type nodeAPIHandler struct {
	nodeID                      string
	reflinkSupport              bool
	topology                    map[string]string
	mountDevice                 func(device, target string) error
	makeFS                      func(ctx context.Context, device, uuid string, force, reflink bool) error
	safeUnmount                 func(target string, force, detach, expire bool) error
	truncate                    func(name string, size int64) error
	attachLoopDevice            func(backingFile string, offset uint64, ro bool) (losetup.Device, error)
	readRunUdevDataByMajorMinor func(majorMinor string) (map[string]string, error)
	probeXFS                    func(path string) (fsuuid, label string, totalCapacity, freeCapacity uint64, err error)
	// locks
	formatLockerMutex sync.Mutex
	formatLocker      map[string]*sync.Mutex
}

// Unmount(target string, force, detach, expire bool) error {
func newNodeAPIHandler(ctx context.Context, identity, nodeID, rack, zone, region string) (*nodeAPIHandler, error) {
	var err error
	nodeAPIHandler := &nodeAPIHandler{
		nodeID:                      nodeID,
		mountDevice:                 xfs.Mount,
		makeFS:                      xfs.MakeFS,
		safeUnmount:                 sys.SafeUnmount,
		truncate:                    os.Truncate,
		attachLoopDevice:            losetup.Attach,
		readRunUdevDataByMajorMinor: device.ReadRunUdevDataByMajorMinor,
		probeXFS:                    xfs.Probe,
		formatLocker:                map[string]*sync.Mutex{},
		topology: map[string]string{
			string(types.TopologyDriverIdentity): identity,
			string(types.TopologyDriverRack):     rack,
			string(types.TopologyDriverZone):     zone,
			string(types.TopologyDriverRegion):   region,
			string(types.TopologyDriverNode):     nodeID,
		},
	}
	nodeAPIHandler.reflinkSupport, err = nodeAPIHandler.isReflinkSupported(ctx)
	if err != nil {
		return nil, err
	}
	return nodeAPIHandler, nil
}

func (n *nodeAPIHandler) getFormatLock(majorMinor string) *sync.Mutex {
	n.formatLockerMutex.Lock()
	defer n.formatLockerMutex.Unlock()

	if _, found := n.formatLocker[majorMinor]; !found {
		n.formatLocker[majorMinor] = &sync.Mutex{}
	}

	return n.formatLocker[majorMinor]
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

	if err = n.truncate(file.Name(), xfs.MinSupportedDeviceSize); err != nil {
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

	if err = n.mountDevice(loopDevice.Path(), mountPoint); err != nil {
		klog.V(3).ErrorS(err, "failed to mount", "reflink", reflinkSupport)
		return errMountFailure
	}

	return n.safeUnmount(mountPoint, true, true, false)
}
