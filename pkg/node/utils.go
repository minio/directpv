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
	"fmt"
	"path/filepath"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/matcher"
	"github.com/minio/directpv/pkg/mount"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"
	"k8s.io/klog/v2"
)

func (n *NodeServer) checkDrive(ctx context.Context, drive *directcsi.DirectCSIDrive, volumeID string) error {

	probeXFSUUID := func(majMin string) (string, error) {
		major, minor, err := utils.GetMajorMinorFromStr(majMin)
		if err != nil {
			klog.V(5).Infof("invalid maj:minor (%s) detected while probing FSUUID: %s", majMin, err.Error())
			return "", err
		}
		dev, err := n.getDevice(major, minor)
		if err != nil {
			klog.V(5).Infof("could not retrieve the device name from maj:min = %s, error: %s", majMin, err.Error())
			return "", err
		}
		fsInfo, err := n.fsProbe(ctx, dev)
		if err != nil {
			klog.V(5).Infof("could not probe fs for device = %s, error: %s", dev, err.Error())
			return "", err
		}
		if fsInfo.Type() != "xfs" {
			klog.V(5).Infof("unexpected fs found for device = %s, fs: %s", dev, fsInfo.Type())
			return "", fmt.Errorf("unexpected fs found in drive %s fs: %s", dev, fsInfo.Type())
		}
		return fsInfo.ID(), nil
	}

	if drive.Status.DriveStatus != directcsi.DriveStatusInUse {
		return fmt.Errorf("drive %v is not in InUse state", drive.Name)
	}

	finalizer := directcsi.DirectCSIDriveFinalizerPrefix + volumeID
	if !matcher.StringIn(drive.Finalizers, finalizer) {
		return fmt.Errorf("drive %v does not have volume finalizer %v", drive.Name, finalizer)
	}

	mounts, err := n.probeMounts()
	if err != nil {
		return err
	}

	majorMinor := fmt.Sprintf("%v:%v", drive.Status.MajorNumber, drive.Status.MinorNumber)
	mountInfos, found := mounts[majorMinor]
	if !found {
		return fmt.Errorf("mount information not found for major/minor %v of drive %v", majorMinor, drive.Name)
	}

	mountPoint := filepath.Join(sys.MountRoot, drive.Status.FilesystemUUID)
	for _, mountInfo := range mountInfos {
		if mountInfo.MountPoint == mountPoint {
			probedFSUUID, err := probeXFSUUID(mountInfo.MajorMinor)
			if err != nil {
				return err
			}
			if probedFSUUID != drive.Status.FilesystemUUID {
				return fmt.Errorf("fssuid check failed for drive %s. probedfsuuid: %s, fsuuid: %s", drive.Name, probedFSUUID, drive.Status.FilesystemUUID)
			}
			return nil
		}
	}

	return fmt.Errorf("drive %v is not mounted at mount point %v", drive.Name, mountPoint)
}

func checkStagingTargetPath(stagingPath string, probeMounts func() (map[string][]mount.MountInfo, error)) error {
	mounts, err := probeMounts()
	if err != nil {
		return err
	}

	for _, mountInfos := range mounts {
		for _, mountInfo := range mountInfos {
			if mountInfo.MountPoint == stagingPath {
				return nil
			}
		}
	}

	return fmt.Errorf("stagingPath %v is not mounted", stagingPath)
}
