// This file is part of MinIO DirectPV
// Copyright (c) 2023 MinIO, Inc.
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
package main

import (
	"context"
	"testing"
	"time"

	"github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/drive"
	"github.com/minio/directpv/pkg/initrequest"
	pkgtypes "github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/volume"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func init() {
	client.FakeInit()
}

func newVolume(name, nodeName string) *pkgtypes.Volume {
	volume := pkgtypes.NewVolume(
		name,
		"",
		types.NodeID(nodeName),
		"sda",
		"sda",
		int64(0),
	)
	return volume
}

func newDrive(name, nodeName string) *pkgtypes.Drive {
	drive := pkgtypes.NewDrive(
		types.DriveID(name),
		pkgtypes.DriveStatus{},
		types.NodeID(nodeName),
		types.DriveName("sda"),
		types.AccessTierDefault,
	)
	return drive
}

func newInitRequest(name, nodeName string) *pkgtypes.InitRequest {
	initReq := pkgtypes.NewInitRequest(
		name,
		types.NodeID(nodeName),
		[]pkgtypes.InitDevice{},
	)
	return initReq
}

func newNode(name string) *pkgtypes.Node {
	node := pkgtypes.NewNode(
		types.NodeID(name),
		[]pkgtypes.Device{},
	)
	return node
}

func TestDrain(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	objects := []runtime.Object{
		newNode("test-node-1"),
		newNode("test-node-2"),
		newDrive("test-drive-1", "test-node-1"),
		newDrive("test-drive-2", "test-node-1"),
		newDrive("test-drive-3", "test-node-2"),
		newDrive("test-drive-4", "test-node-2"),
		newVolume("test-volume-1", "test-node-1"),
		newVolume("test-volume-2", "test-node-1"),
		newVolume("test-volume-3", "test-node-2"),
		newVolume("test-volume-4", "test-node-2"),
		newInitRequest("test-initreq-1", "test-node-1"),
		newInitRequest("test-initreq-2", "test-node-1"),
		newInitRequest("test-initreq-3", "test-node-2"),
		newInitRequest("test-initreq-4", "test-node-2"),
	}

	clientset := pkgtypes.NewExtFakeClientset(clientsetfake.NewSimpleClientset(objects...))
	client.SetNodeInterface(clientset.DirectpvLatest().DirectPVNodes())
	client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())
	client.SetInitRequestInterface(clientset.DirectpvLatest().DirectPVInitRequests())

	quietFlag = true
	dangerousFlag = true
	nodesArgs = []string{"test-node-1", "test-node-2"}
	drainMain(ctx)

	drives, err := drive.NewLister().IgnoreNotFound(true).Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(drives) != 0 {
		t.Fatal("drives are not cleared upon draining")
	}

	volumes, err := volume.NewLister().IgnoreNotFound(true).Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(volumes) != 0 {
		t.Fatal("volumes are not cleared upon draining")
	}

	initReqs, err := initrequest.NewLister().IgnoreNotFound(true).Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(initReqs) != 0 {
		t.Fatal("initrequests are not cleared upon draining")
	}

	for _, node := range nodesArgs {
		_, err := client.NodeClient().Get(ctx, node, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
		if err == nil {
			t.Fatalf("node %s not deleted upon draining", node)
		}
	}
}
