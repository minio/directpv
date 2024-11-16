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

package client

import (
	"context"
	"fmt"
	"os"

	"github.com/minio/directpv/pkg/k8s"
	directcsi "github.com/minio/directpv/pkg/legacy/apis/direct.csi.min.io/v1beta5"
	typeddirectcsi "github.com/minio/directpv/pkg/legacy/clientset/typed/direct.csi.min.io/v1beta5"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
)

var (
	initialized int32
	client      *Client
)

// Client represents the legacy client
type Client struct {
	DriveClient  typeddirectcsi.DirectCSIDriveInterface
	VolumeClient typeddirectcsi.DirectCSIVolumeInterface
	K8sClient    *k8s.Client
}

// Discovery returns the discovery client
func (client Client) Discovery() discovery.DiscoveryInterface {
	return client.K8sClient.DiscoveryClient
}

// Drive returns the legacy drive client
func (client Client) Drive() typeddirectcsi.DirectCSIDriveInterface {
	return client.DriveClient
}

// Volume returns the volume client
func (client Client) Volume() typeddirectcsi.DirectCSIVolumeInterface {
	return client.VolumeClient
}

// DirectCSI group and identity names.
const (
	GroupName = "direct.csi.min.io"
	Identity  = "direct-csi-min-io"
)

// DirectCSIVersionLabelKey is the version with group and version ...
const DirectCSIVersionLabelKey = directcsi.Group + "/" + directcsi.Version

// DirectCSIDriveTypeMeta gets new direct-csi drive meta.
func DirectCSIDriveTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: DirectCSIVersionLabelKey,
		Kind:       "DirectCSIDrive",
	}
}

// DirectCSIVolumeTypeMeta gets new direct-csi volume meta.
func DirectCSIVolumeTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: DirectCSIVersionLabelKey,
		Kind:       "DirectCSIVolume",
	}
}

// DriveClient gets latest versioned drive interface.
func DriveClient() typeddirectcsi.DirectCSIDriveInterface {
	return client.DriveClient
}

// VolumeClient gets latest versioned volume interface.
func VolumeClient() typeddirectcsi.DirectCSIVolumeInterface {
	return client.VolumeClient
}

// GetClient returns the client
func GetClient() *Client {
	return client
}

// RemoveAllDrives removes legacy drive CRDs.
func (client Client) RemoveAllDrives(ctx context.Context, backupFile string) (backupCreated bool, err error) {
	var drives []directcsi.DirectCSIDrive
	for result := range client.ListDrives(ctx) {
		if result.Err != nil {
			return false, fmt.Errorf("unable to get legacy drives; %w", result.Err)
		}
		drives = append(drives, result.Drive)
	}
	if len(drives) == 0 {
		return false, nil
	}

	data, err := utils.ToYAML(directcsi.DirectCSIDriveList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "v1",
		},
		Items: drives,
	})
	if err != nil {
		return false, fmt.Errorf("unable to generate legacy drives YAML; %w", err)
	}

	if err = os.WriteFile(backupFile, data, os.ModePerm); err != nil {
		return false, fmt.Errorf("unable to write legacy drives YAML; %w", err)
	}

	for _, drive := range drives {
		drive.Finalizers = []string{}
		if _, err := client.Drive().Update(ctx, &drive, metav1.UpdateOptions{}); err != nil {
			return false, fmt.Errorf("unable to update legacy drive %v; %w", drive.Name, err)
		}
		if err := client.Drive().Delete(ctx, drive.Name, metav1.DeleteOptions{}); err != nil {
			return false, fmt.Errorf("unable to remove legacy drive %v; %w", drive.Name, err)
		}
	}

	return true, nil
}

// RemoveAllVolumes removes legacy volume CRDs.
func (client Client) RemoveAllVolumes(ctx context.Context, backupFile string) (backupCreated bool, err error) {
	var volumes []directcsi.DirectCSIVolume
	for result := range client.ListVolumes(ctx) {
		if result.Err != nil {
			return false, fmt.Errorf("unable to get legacy volumes; %w", result.Err)
		}
		volumes = append(volumes, result.Volume)
	}
	if len(volumes) == 0 {
		return false, nil
	}

	data, err := utils.ToYAML(directcsi.DirectCSIVolumeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "v1",
		},
		Items: volumes,
	})
	if err != nil {
		return false, fmt.Errorf("unable to generate legacy volumes YAML; %w", err)
	}

	if err = os.WriteFile(backupFile, data, os.ModePerm); err != nil {
		return false, fmt.Errorf("unable to write legacy volumes YAML; %w", err)
	}

	for _, volume := range volumes {
		volume.Finalizers = nil
		if _, err := client.Volume().Update(ctx, &volume, metav1.UpdateOptions{}); err != nil {
			return false, fmt.Errorf("unable to update legacy volume %v; %w", volume.Name, err)
		}
		if err := client.Volume().Delete(ctx, volume.Name, metav1.DeleteOptions{}); err != nil {
			return false, fmt.Errorf("unable to remove legacy volume %v; %w", volume.Name, err)
		}
	}

	return true, nil
}
