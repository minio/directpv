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

package listener

import (
	"context"

	// storage
	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta1"
	"github.com/minio/direct-csi/pkg/clientset"

	// k8s client
	kubeclientset "k8s.io/client-go/kubernetes"
)

// Set the clients for each of the listeners
type GenericListener interface {
	InitializeKubeClient(kubeclientset.Interface)
	InitializeDirectCSIClient(clientset.Interface)
}

type DirectCSIVolumeListener interface {
	GenericListener

	Add(ctx context.Context, b *directcsi.DirectCSIVolume) error
	Update(ctx context.Context, old *directcsi.DirectCSIVolume, new *directcsi.DirectCSIVolume) error
	Delete(ctx context.Context, b *directcsi.DirectCSIVolume) error
}

func (c *DirectCSIController) AddDirectCSIVolumeListener(b DirectCSIVolumeListener) {
	c.initialized = true
	c.DirectCSIVolumeListener = b
}

type DirectCSIDriveListener interface {
	GenericListener

	Add(ctx context.Context, b *directcsi.DirectCSIDrive) error
	Update(ctx context.Context, old *directcsi.DirectCSIDrive, new *directcsi.DirectCSIDrive) error
	Delete(ctx context.Context, b *directcsi.DirectCSIDrive) error
}

func (c *DirectCSIController) AddDirectCSIDriveListener(b DirectCSIDriveListener) {
	c.initialized = true
	c.DirectCSIDriveListener = b
}
