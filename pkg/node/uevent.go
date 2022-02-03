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

package node

import (
	"context"
	"path"
	
	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/uevent"
	"github.com/minio/directpv/pkg/utils"
	"k8s.io/klog/v2"
)

func RunDynamicDriveHandler(ctx context.Context,
	identity, nodeID, rack, zone, region string,
	loopbackOnly bool) error {

	handler := &uEventHandler{
		nodeID: nodeID,
		topology: map[string]string{
			string(utils.TopologyDriverIdentity): identity,
			string(utils.TopologyDriverRack):     rack,
			string(utils.TopologyDriverZone):     zone,
			string(utils.TopologyDriverRegion):   region,
			string(utils.TopologyDriverNode):     nodeID,
		},
		loopbackOnly: loopbackOnly,
	}
	if loopbackOnly {
		if err := sys.CreateLoopDevices(); err != nil {
			return err
		}
	}

	listener, err := uevent.NewListener(&driveEventHandler{
		nodeID: nodeID,
	})
	if err != nil {
		return err
	}
	defer listener.Close()

	return listener.Run(ctx)
}

type driveEventHandler struct {
	nodeID string
}

func (d *driveEventHandler) remove(ctx context.Context,
	device *sys.Device,
	drives *directcsi.DirectCSIDrive) error {

	for _, drive := range drives {

	}

	return nil
}

func (d *driveEventHandler) update(ctx context.Context,
	device *sys.Device,
	drive *directcsi.DirectCSIDrive) error {

	// path - ?
	// ...
	
	
	return nil
}

func (d *driveEventHandler) add(ctx context.Context,
	device *sys.Device) error {

	// construct directcsiDrive object here and push it to etcd
	
}

func (d *driveEventHandler) findMatchingDrive() {
}

func (d *driveEventHandler) Handle(ctx context.Context, event map[string]string) error {

	if sys.LoopRegexp.MatchString(path.Base(event["DEVPATH"])) {
		klog.V(5).InfoS(
			"loopback device is ignored",
			"ACTION", event["ACTION"],
			"DEVPATH", event["DEVPATH"])
		return nil
	}

	device, err := sys.CreateDevice(event)
	if err != nil {
		klog.ErrorS(err, "ACTION", event["ACTION"], "DEVPATH", event["DEVPATH"])
		return nil
	}

	driveCh, err := client.ListDrives(
		ctx,
		[]utils.LabelValue{utils.NewLabelValue(d.nodeID)},
		[]utils.LabelValue{utils.NewLabelValue(device.Name)},
		nil,
		client.MaxThreadCount,
	)
	if err != nil {
		klog.ErrorS(err, "error while finding matching drive")
		return err
	}

	drives := []*directcsi.DirectCSIDrive{}
	for drive := range driveCh {
		drives = append(drives, drive)
	}
	
	if len(drives) == 0 {
		klog.V(5).Infof("no matching DirectPV drive found", "device", device.Name)
		// code to handle new drive
	} else {
		switch event["ACTION"] {
		case uevent.Remove:
			return d.remove(ctx, device, drive)
		default:
			return d.update(ctx, device, drive)
		}
	}
	return nil
}
