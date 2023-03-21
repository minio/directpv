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

package node

import (
	"context"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/minio/directpv/pkg/client"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/xfs"
)

func TestNodeExpandVolume(t *testing.T) {
	volumeID := "volume-id-1"
	volume := types.NewVolume(volumeID, "fsuuid1", "node-1", "drive-1", "sda", 100*MiB)
	volume.Status.DataPath = "volume/id/1/data/path"
	volume.Status.StagingTargetPath = "volume/id/1/staging/target/path"

	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(volume))
	client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	nodeServer := createFakeServer()
	nodeServer.getDeviceByFSUUID = func(fsuuid string) (string, error) {
		return "sda", nil
	}
	nodeServer.setQuota = func(ctx context.Context, device, dataPath, name string, quota xfs.Quota, update bool) error {
		return nil
	}

	if _, err := nodeServer.NodeExpandVolume(context.TODO(), &csi.NodeExpandVolumeRequest{
		VolumeId:      volumeID,
		VolumePath:    "volume-id-1-volume-path",
		CapacityRange: &csi.CapacityRange{RequiredBytes: 100 * MiB},
	}); err != nil {
		t.Fatal(err)
	}
}
