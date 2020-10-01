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

package centralcontroller

import (
	"context"

	"github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	"github.com/minio/direct-csi/pkg/util"
)

func (c *Controller) OnAdd(ctx context.Context, st *v1alpha1.StorageTopology) error {
	kClient := util.GetKubeClientOrDie()

	if err := createDirectCSINamespace(ctx, kClient, st.Name); err != nil {
		return err
	}
	if err := createRBACRoles(ctx, kClient, st.Name); err != nil {
		return err
	}
	if err := createCSIDriver(ctx, kClient, st.Name); err != nil {
		return err
	}
	if err := createStorageClass(ctx, kClient, st.Name); err != nil {
		return err
	}
	if err := createDaemonSet(ctx, kClient, st.Name, c.Identity, st.Spec); err != nil {
		return err
	}
	if err := createDeployment(ctx, kClient, st.Name, c.Identity); err != nil {
		return err
	}

	return nil
}

func (c *Controller) OnUpdate(ctx context.Context, old, new *v1alpha1.StorageTopology) error {
	return nil
}

func (c *Controller) OnDelete(ctx context.Context, obj *v1alpha1.StorageTopology) error {
	return nil
}
