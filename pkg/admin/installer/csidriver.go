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

package installer

import (
	"context"
	"errors"
	"fmt"

	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	legacyclient "github.com/minio/directpv/pkg/legacy/client"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type csiDriverTask struct {
	client *client.Client
}

func (csiDriverTask) Name() string {
	return "CSIDriver"
}

func (csiDriverTask) Start(ctx context.Context, args *Args) error {
	steps := 1
	if args.Legacy {
		steps++
	}
	if !sendStartMessage(ctx, args.ProgressCh, steps) {
		return errSendProgress
	}
	return nil
}

func (csiDriverTask) End(ctx context.Context, args *Args, err error) error {
	if !sendEndMessage(ctx, args.ProgressCh, err) {
		return errSendProgress
	}
	return nil
}

func (t csiDriverTask) Execute(ctx context.Context, args *Args) error {
	return t.createCSIDriver(ctx, args)
}

func (t csiDriverTask) Delete(ctx context.Context, _ *Args) error {
	return t.deleteCSIDriver(ctx)
}

var errCSIDriverVersionUnsupported = errors.New("unsupported CSIDriver version found")

func (t csiDriverTask) doCreateCSIDriver(ctx context.Context, args *Args, version string, legacy bool, step int) (err error) {
	name := consts.Identity
	if legacy {
		name = legacyclient.Identity
	}
	if !sendProgressMessage(ctx, args.ProgressCh, fmt.Sprintf("Creating %s CSI Driver", name), step, nil) {
		return errSendProgress
	}
	defer func() {
		if err == nil {
			if !sendProgressMessage(ctx, args.ProgressCh, fmt.Sprintf("Created %s CSI Driver", name), step, csiDriverComponent(name)) {
				err = errSendProgress
			}
		}
	}()

	podInfoOnMount := true
	attachRequired := false
	switch version {
	case "v1":
		csiDriver := &storagev1.CSIDriver{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "storage.k8s.io/v1",
				Kind:       "CSIDriver",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: metav1.NamespaceNone,
				Labels:    defaultLabels,
			},
			Spec: storagev1.CSIDriverSpec{
				PodInfoOnMount: &podInfoOnMount,
				AttachRequired: &attachRequired,
				VolumeLifecycleModes: []storagev1.VolumeLifecycleMode{
					storagev1.VolumeLifecyclePersistent,
					storagev1.VolumeLifecycleEphemeral,
				},
			},
		}

		if !args.DryRun && !args.Declarative {
			_, err := t.client.Kube().StorageV1().CSIDrivers().Create(ctx, csiDriver, metav1.CreateOptions{})
			if err != nil && !apierrors.IsAlreadyExists(err) {
				return err
			}
		}

		return args.writeObject(csiDriver)

	case "v1beta1":
		csiDriver := &storagev1beta1.CSIDriver{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "storage.k8s.io/v1beta1",
				Kind:       "CSIDriver",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: metav1.NamespaceNone,
				Labels:    defaultLabels,
			},
			Spec: storagev1beta1.CSIDriverSpec{
				PodInfoOnMount: &podInfoOnMount,
				AttachRequired: &attachRequired,
				VolumeLifecycleModes: []storagev1beta1.VolumeLifecycleMode{
					storagev1beta1.VolumeLifecyclePersistent,
					storagev1beta1.VolumeLifecycleEphemeral,
				},
			},
		}

		if !args.DryRun && !args.Declarative {
			_, err := t.client.Kube().StorageV1beta1().CSIDrivers().Create(ctx, csiDriver, metav1.CreateOptions{})
			if err != nil && !apierrors.IsAlreadyExists(err) {
				return err
			}
		}

		return args.writeObject(csiDriver)

	default:
		return errCSIDriverVersionUnsupported
	}
}

func (t csiDriverTask) createCSIDriver(ctx context.Context, args *Args) (err error) {
	version := "v1"
	if args.DryRun {
		if args.KubeVersion.Major() >= 1 && args.KubeVersion.Minor() < 19 {
			version = "v1beta1"
		}
	} else {
		gvk, err := t.client.K8s().GetGroupVersionKind("storage.k8s.io", "CSIDriver", "v1", "v1beta1")
		if err != nil {
			return err
		}
		version = gvk.Version
	}

	if err := t.doCreateCSIDriver(ctx, args, version, false, 1); err != nil {
		return err
	}

	if args.Legacy {
		if err := t.doCreateCSIDriver(ctx, args, version, true, 2); err != nil {
			return err
		}
	}

	return nil
}

func (t csiDriverTask) doDeleteCSIDriver(ctx context.Context, version, name string) (err error) {
	switch version {
	case "v1":
		err = t.client.Kube().StorageV1().CSIDrivers().Delete(
			ctx, name, metav1.DeleteOptions{},
		)
	case "v1beta1":
		err = t.client.Kube().StorageV1beta1().CSIDrivers().Delete(
			ctx, name, metav1.DeleteOptions{},
		)
	default:
		return errCSIDriverVersionUnsupported
	}

	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

func (t csiDriverTask) deleteCSIDriver(ctx context.Context) error {
	gvk, err := t.client.K8s().GetGroupVersionKind("storage.k8s.io", "CSIDriver", "v1", "v1beta1")
	if err != nil {
		return err
	}

	if err = t.doDeleteCSIDriver(ctx, gvk.Version, consts.Identity); err != nil {
		return err
	}

	return t.doDeleteCSIDriver(ctx, gvk.Version, legacyclient.Identity)
}
