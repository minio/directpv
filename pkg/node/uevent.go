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
	"errors"
	"path"

	"github.com/google/uuid"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/uevent"
	"github.com/minio/directpv/pkg/utils"
	"k8s.io/klog/v2"
)

var (
	errNoMatchFound = errors.New("no matching drive found")
)

func RunDynamicDriveHandler(ctx context.Context,
	identity, nodeID, rack, zone, region string,
	loopbackOnly bool) error {

	handler := &driveEventHandler{
		nodeID: nodeID,
		topology: map[string]string{
			string(utils.TopologyDriverIdentity): identity,
			string(utils.TopologyDriverRack):     rack,
			string(utils.TopologyDriverZone):     zone,
			string(utils.TopologyDriverRegion):   region,
			string(utils.TopologyDriverNode):     nodeID,
		},
		createDevice: sys.CreateDevice,
	}

	listener, err := uevent.NewListener(handler)
	if err != nil {
		return err
	}
	defer listener.Close()

	return listener.Run(ctx)
}

type driveEventHandler struct {
	nodeID       string
	topology     map[string]string
	createDevice func(event map[string]string) (device *sys.Device, err error)
}

func (d *driveEventHandler) remove(ctx context.Context,
	device *sys.Device,
	drive *directcsi.DirectCSIDrive) error {

	return nil
}

func (d *driveEventHandler) update(ctx context.Context,
	device *sys.Device,
	drive *directcsi.DirectCSIDrive) error {

	// path - ?
	// ...

	return nil
}

func (d *driveEventHandler) add(
	ctx context.Context,
	device *sys.Device) error {
	drive := client.NewDirectCSIDrive(
		uuid.New().String(),
		client.NewDirectCSIDriveStatus(device, d.nodeID, d.topology),
	)
	err := client.CreateDrive(ctx, drive)
	if err != nil {
		klog.ErrorS(err, "unable to create drive", "Status.Path", drive.Status.Path)
	}
	return err
}

func (d *driveEventHandler) findMatchingDrive(drives []directcsi.DirectCSIDrive, device *sys.Device) (*directcsi.DirectCSIDrive, error) {
	//  todo: run matching algorithm to find matching drive
	//  note: return `errNoMatchFound` if no match is found
	return nil, errNoMatchFound
}

func (d *driveEventHandler) Handle(ctx context.Context, device *sys.Device) error {

	if sys.LoopRegexp.MatchString(path.Base(event["DEVPATH"])) {
		klog.V(5).InfoS(
			"loopback device is ignored",
			"ACTION", event["ACTION"],
			"DEVPATH", event["DEVPATH"])
		return nil
	}

	device, err := d.createDevice(event)
	if err != nil {
		klog.ErrorS(err, "ACTION", event["ACTION"], "DEVPATH", event["DEVPATH"])
		return nil
	}

	drives, err := client.GetDriveList(
		ctx,
		[]utils.LabelValue{utils.NewLabelValue(d.nodeID)},
		[]utils.LabelValue{utils.NewLabelValue(device.Name)},
		nil,
	)
	if err != nil {
		klog.ErrorS(err, "error while fetching drive list")
		return err
	}

	drive, err := d.findMatchingDrive(drives, device)
	switch {
	case errors.Is(err, errNoMatchFound):
		if event["ACTION"] == uevent.Remove {
			klog.V(3).InfoS(
				"matching drive not found",
				"ACTION", uevent.Remove,
				"DEVPATH", event["DEVPATH"])
			return nil
		}
		return d.add(ctx, device)
	case err == nil:
		switch event["ACTION"] {
		case uevent.Remove:
			return d.remove(ctx, device, drive)
		default:
			return d.update(ctx, device, drive)
		}
	default:
		return err
	}
}
