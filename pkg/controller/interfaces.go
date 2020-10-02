// This file is part of MinIO Kubernetes Cloud
// Copyright (c) 2020 MinIO, Inc.
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

package controller

import (
	"context"

	// storage
	direct "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	directclientset "github.com/minio/direct-csi/pkg/clientset"

	// k8s client
	kubeclientset "k8s.io/client-go/kubernetes"
)

// Set the clients for each of the listeners
type GenericListener interface {
	InitializeKubeClient(kubeclientset.Interface)
	InitializeDirectCSIClient(directclientset.Interface)
}

type StorageTopologyListener interface {
	GenericListener

	Add(ctx context.Context, b *direct.StorageTopology) error
	Update(ctx context.Context, old *direct.StorageTopology, new *direct.StorageTopology) error
	Delete(ctx context.Context, b *direct.StorageTopology) error
}

func (c *DirectCSIController) AddStorageTopologyListener(st StorageTopologyListener) {
	c.initialized = true
	c.StorageTopologyListener = st
}
