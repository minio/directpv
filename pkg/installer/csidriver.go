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
	"io"

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	legacyclient "github.com/minio/directpv/pkg/legacy/client"
	"github.com/minio/directpv/pkg/utils"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	errCSIDriverVersionUnsupported = errors.New("unsupported CSIDriver version found")

	existingCSIDriverV1      *storagev1.CSIDriver
	existingCSIDriverV1beta1 *storagev1beta1.CSIDriver
)

type csiDriverTask struct{}

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

func (csiDriverTask) Execute(ctx context.Context, args *Args) error {
	return createCSIDriver(ctx, args)
}

func (csiDriverTask) Delete(ctx context.Context, _ *Args) error {
	return deleteCSIDriver(ctx)
}

func doCreateCSIDriver(ctx context.Context, args *Args, version string, legacy bool, step int) (err error) {
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
		update := false
		creationTimestamp := metav1.Time{}
		resourceVersion := ""
		uid := types.UID("")
		if !args.dryRun() {
			existingCSIDriverV1, err = k8s.KubeClient().StorageV1().CSIDrivers().Get(
				ctx, name, metav1.GetOptions{},
			)
			switch {
			case err != nil:
				if !apierrors.IsNotFound(err) {
					return err
				}
			default:
				update = true
				creationTimestamp = existingCSIDriverV1.CreationTimestamp
				resourceVersion = existingCSIDriverV1.ResourceVersion
				uid = existingCSIDriverV1.UID
				if _, err = io.WriteString(args.backupWriter, utils.MustGetYAML(existingCSIDriverV1)); err != nil {
					return err
				}
			}
		}

		csiDriver := &storagev1.CSIDriver{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "storage.k8s.io/v1",
				Kind:       "CSIDriver",
			},
			ObjectMeta: metav1.ObjectMeta{
				CreationTimestamp: creationTimestamp,
				ResourceVersion:   resourceVersion,
				UID:               uid,
				Name:              name,
				Namespace:         metav1.NamespaceNone,
				Annotations:       map[string]string{},
				Labels:            defaultLabels,
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

		if args.dryRun() {
			args.DryRunPrinter(csiDriver)
			return nil
		}

		if update {
			_, err = k8s.KubeClient().StorageV1().CSIDrivers().Update(ctx, csiDriver, metav1.UpdateOptions{})
		} else {
			_, err = k8s.KubeClient().StorageV1().CSIDrivers().Create(ctx, csiDriver, metav1.CreateOptions{})
		}
		if err != nil {
			return err
		}

		_, err = io.WriteString(args.auditWriter, utils.MustGetYAML(csiDriver))
		return err

	case "v1beta1":
		update := false
		creationTimestamp := metav1.Time{}
		resourceVersion := ""
		uid := types.UID("")
		if !args.dryRun() {
			existingCSIDriverV1beta1, err := k8s.KubeClient().StorageV1beta1().CSIDrivers().Get(
				ctx, name, metav1.GetOptions{},
			)
			switch {
			case err != nil:
				if !apierrors.IsNotFound(err) {
					return err
				}
			default:
				update = true
				creationTimestamp = existingCSIDriverV1beta1.CreationTimestamp
				resourceVersion = existingCSIDriverV1beta1.ResourceVersion
				uid = existingCSIDriverV1beta1.UID
				if _, err = io.WriteString(args.backupWriter, utils.MustGetYAML(existingCSIDriverV1beta1)); err != nil {
					return err
				}
			}
		}

		csiDriver := &storagev1beta1.CSIDriver{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "storage.k8s.io/v1beta1",
				Kind:       "CSIDriver",
			},
			ObjectMeta: metav1.ObjectMeta{
				CreationTimestamp: creationTimestamp,
				ResourceVersion:   resourceVersion,
				UID:               uid,
				Name:              name,
				Namespace:         metav1.NamespaceNone,
				Annotations:       map[string]string{},
				Labels:            defaultLabels,
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

		if args.dryRun() {
			args.DryRunPrinter(csiDriver)
			return nil
		}

		if update {
			_, err = k8s.KubeClient().StorageV1beta1().CSIDrivers().Update(ctx, csiDriver, metav1.UpdateOptions{})
		} else {
			_, err = k8s.KubeClient().StorageV1beta1().CSIDrivers().Create(ctx, csiDriver, metav1.CreateOptions{})
		}
		if err != nil {
			return err
		}

		_, err = io.WriteString(args.auditWriter, utils.MustGetYAML(csiDriver))
		return err

	default:
		return errCSIDriverVersionUnsupported
	}
}

func createCSIDriver(ctx context.Context, args *Args) (err error) {
	version := "v1"
	if args.dryRun() {
		if args.KubeVersion.Major() >= 1 && args.KubeVersion.Minor() < 19 {
			version = "v1beta1"
		}
	} else {
		gvk, err := k8s.GetGroupVersionKind("storage.k8s.io", "CSIDriver", "v1", "v1beta1")
		if err != nil {
			return err
		}
		version = gvk.Version
	}

	if err := doCreateCSIDriver(ctx, args, version, false, 1); err != nil {
		return err
	}

	if args.Legacy {
		if err := doCreateCSIDriver(ctx, args, version, true, 2); err != nil {
			return err
		}
	}

	return nil
}

func doRevertCSIDriver(ctx context.Context, name string) error {
	switch {
	case existingCSIDriverV1 != nil:
		csiDriver, err := k8s.KubeClient().StorageV1().CSIDrivers().Get(
			ctx, name, metav1.GetOptions{},
		)
		if err != nil {
			return err
		}

		existingCSIDriverV1.ResourceVersion = csiDriver.ResourceVersion
		_, err = k8s.KubeClient().StorageV1().CSIDrivers().Update(ctx, existingCSIDriverV1, metav1.UpdateOptions{})
		return err
	case existingCSIDriverV1beta1 != nil:
		csiDriver, err := k8s.KubeClient().StorageV1beta1().CSIDrivers().Get(
			ctx, name, metav1.GetOptions{},
		)
		if err != nil {
			return err
		}

		existingCSIDriverV1beta1.ResourceVersion = csiDriver.ResourceVersion
		_, err = k8s.KubeClient().StorageV1beta1().CSIDrivers().Update(ctx, existingCSIDriverV1beta1, metav1.UpdateOptions{})
		return err
	}

	return nil
}

func revertCSIDriver(ctx context.Context, args *Args) error {
	var err, legacyErr error
	err = doRevertCSIDriver(ctx, consts.Identity)
	if args.Legacy {
		legacyErr = doRevertCSIDriver(ctx, legacyclient.Identity)
	}

	if err != nil && legacyErr != nil {
		return fmt.Errorf("unable to revert; CSIDriver: %v; legacy CSIDriver: %v", err, legacyErr)
	}

	if err != nil {
		return err
	}

	return legacyErr
}

func doDeleteCSIDriver(ctx context.Context, version, name string) (err error) {
	switch version {
	case "v1":
		err = k8s.KubeClient().StorageV1().CSIDrivers().Delete(
			ctx, name, metav1.DeleteOptions{},
		)
	case "v1beta1":
		err = k8s.KubeClient().StorageV1beta1().CSIDrivers().Delete(
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

func deleteCSIDriver(ctx context.Context) error {
	gvk, err := k8s.GetGroupVersionKind("storage.k8s.io", "CSIDriver", "v1", "v1beta1")
	if err != nil {
		return err
	}

	if err = doDeleteCSIDriver(ctx, gvk.Version, consts.Identity); err != nil {
		return err
	}

	return doDeleteCSIDriver(ctx, gvk.Version, legacyclient.Identity)
}
