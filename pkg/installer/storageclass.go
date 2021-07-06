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

package installer

import (
	"context"
	"errors"

	"github.com/minio/direct-csi/pkg/utils"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/klog/v2"
)

var _ Installer = &SCInstaller{}

type SCInstaller struct {
	name string

	*InstallConfig
}

func NewSCInstaller(name string) *SCInstaller {
	return &SCInstaller{
		name: name,
	}
}

func (sc *SCInstaller) Init(i *InstallConfig) error {
	sc.InstallConfig = i

	identity := sc.GetIdentity()
	if identity == "" {
		err := errors.New("Identity cannot be empty")
		klog.ErrorS(err, "Invalid configuration", "Installer", "StorageClassInstaller")
		return err
	}
	sc.name = utils.SanitizeKubeResourceName(utils.DefaultIfZeroString(sc.name, identity))

	return nil
}

func (sc *SCInstaller) Install(ctx context.Context) error {
	scName := utils.SanitizeKubeResourceName(sc.name)
	allowExpansionFalse := false
	allowTopologiesWithName := utils.NewIdentityTopologySelector(scName)
	reclaimPolicyDelete := corev1.PersistentVolumeReclaimDelete
	bindingModeWaitForFirstConsumer := storagev1.VolumeBindingWaitForFirstConsumer

	// Create StorageClass for the new driver
	storageClass := &storagev1.StorageClass{
		TypeMeta: utils.NewTypeMeta("storage.k8s.io/v1", "StorageClass"),
		ObjectMeta: utils.NewObjectMeta(
			scName,
			metav1.NamespaceNone,
			defaultLabels,
			defaultAnnotations,
			[]string{
				metav1.FinalizerDeleteDependents, // foregroundDeletion finalizer
			},
			nil,
		),
		Provisioner:          scName,
		AllowVolumeExpansion: &allowExpansionFalse,
		VolumeBindingMode:    &bindingModeWaitForFirstConsumer,
		AllowedTopologies: []corev1.TopologySelectorTerm{
			allowTopologiesWithName,
		},
		ReclaimPolicy: &reclaimPolicyDelete,
	}

	createdSC, err := utils.GetKubeClient().StorageV1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{
		DryRun: sc.getDryRunDirectives(),
	})
	if err != nil {
		return err
	}
	return sc.PostProc(createdSC)
}

func (sc *SCInstaller) Uninstall(ctx context.Context) error {
	scName := sc.name
	foregroundDeletePropagation := metav1.DeletePropagationForeground
	// Delete Namespace Obj
	return utils.GetKubeClient().StorageV1().StorageClasses().Delete(ctx, scName, metav1.DeleteOptions{
		DryRun:            sc.getDryRunDirectives(),
		PropagationPolicy: &foregroundDeletePropagation,
	})
}
