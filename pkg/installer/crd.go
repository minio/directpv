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
	_ "embed"
	"fmt"
	"io"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/drive"
	"github.com/minio/directpv/pkg/initrequest"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/node"
	"github.com/minio/directpv/pkg/volume"
	"k8s.io/apiextensions-apiserver/pkg/apihelpers"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

//go:embed directpv.min.io_directpvdrives.yaml
var drivesYAML []byte

//go:embed directpv.min.io_directpvvolumes.yaml
var volumesYAML []byte

//go:embed directpv.min.io_directpvnodes.yaml
var nodesYAML []byte

//go:embed directpv.min.io_directpvinitrequests.yaml
var initrequestsYAML []byte

func setNoneConversionStrategy(crd *apiextensions.CustomResourceDefinition) {
	crd.Spec.Conversion = &apiextensions.CustomResourceConversion{
		Strategy: apiextensions.NoneConverter,
	}
}

func updateLabels(
	object metav1.Object,
	labels map[directpvtypes.LabelKey]directpvtypes.LabelValue,
) {
	values := object.GetLabels()
	if values == nil {
		values = make(map[string]string)
	}

	for key, value := range labels {
		values[string(key)] = string(value)
	}

	object.SetLabels(values)
}

func getLatestCRDVersionObject(
	newCRD *apiextensions.CustomResourceDefinition,
) (crdVersion apiextensions.CustomResourceDefinitionVersion, err error) {
	for i := range newCRD.Spec.Versions {
		if newCRD.Spec.Versions[i].Name == consts.LatestAPIVersion {
			return newCRD.Spec.Versions[i], nil
		}
	}

	return crdVersion, fmt.Errorf("no version %v found crd %v", consts.LatestAPIVersion, newCRD.Name)
}

func updateCRD(
	ctx context.Context,
	existingCRD, newCRD *apiextensions.CustomResourceDefinition,
) (*apiextensions.CustomResourceDefinition, error) {
	existingCRDStorageVersion, err := apihelpers.GetCRDStorageVersion(existingCRD)
	if err != nil {
		return nil, err
	}

	setNoneConversionStrategy(existingCRD)

	// CRD is already in the latest version
	if existingCRDStorageVersion == consts.LatestAPIVersion {
		return existingCRD, nil
	}

	var versionEntryFound bool
	// Set all the existing versions to false
	for i := range existingCRD.Spec.Versions {
		if existingCRD.Spec.Versions[i].Name == consts.LatestAPIVersion {
			existingCRD.Spec.Versions[i].Storage = true
			versionEntryFound = true
		} else {
			existingCRD.Spec.Versions[i].Storage = false
		}
	}

	if !versionEntryFound {
		latestVersionObject, err := getLatestCRDVersionObject(newCRD)
		if err != nil {
			return nil, err
		}
		existingCRD.Spec.Versions = append(existingCRD.Spec.Versions, latestVersionObject)
	}

	return k8s.CRDClient().Update(ctx, existingCRD, metav1.UpdateOptions{})
}

func createCRDs(ctx context.Context, args *Args) error {
	register := func(data []byte) error {
		object := map[string]interface{}{}
		if err := yaml.Unmarshal(data, &object); err != nil {
			return err
		}

		var crd apiextensions.CustomResourceDefinition
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object, &crd); err != nil {
			return err
		}

		if args.DryRun {
			updateLabels(
				&crd,
				map[directpvtypes.LabelKey]directpvtypes.LabelValue{
					directpvtypes.VersionLabelKey: consts.LatestAPIVersion,
				},
			)

			fmt.Print(mustGetYAML(crd))
			return nil
		}

		existingCRD, err := k8s.CRDClient().Get(ctx, crd.Name, metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}

			setNoneConversionStrategy(&crd)

			_, err := k8s.CRDClient().Create(ctx, &crd, metav1.CreateOptions{})
			if err != nil {
				return err
			}

			_, err = io.WriteString(args.auditWriter, mustGetYAML(crd))
			return err
		}

		updatedCRD, err := updateCRD(ctx, existingCRD, &crd)
		if err != nil {
			return err
		}

		_, err = io.WriteString(args.auditWriter, mustGetYAML(updatedCRD))
		return err
	}

	if err := register(drivesYAML); err != nil {
		return err
	}

	if err := register(volumesYAML); err != nil {
		return err
	}

	if err := register(nodesYAML); err != nil {
		return err
	}

	return register(initrequestsYAML)
}

func removeVolumes(ctx context.Context) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	for result := range volume.NewLister().List(ctx) {
		if result.Err != nil {
			if apierrors.IsNotFound(result.Err) {
				break
			}
			return result.Err
		}

		result.Volume.RemovePVProtection()
		result.Volume.RemovePurgeProtection()

		_, err := client.VolumeClient().Update(ctx, &result.Volume, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

		err = client.VolumeClient().Delete(ctx, result.Volume.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func removeDrives(ctx context.Context) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	for result := range drive.NewLister().List(ctx) {
		if result.Err != nil {
			if apierrors.IsNotFound(result.Err) {
				break
			}
			return result.Err
		}
		result.Drive.Finalizers = []string{}
		_, err := client.DriveClient().Update(ctx, &result.Drive, metav1.UpdateOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}

		err = client.DriveClient().Delete(ctx, result.Drive.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func removeNodes(ctx context.Context) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	for result := range node.NewLister().List(ctx) {
		if result.Err != nil {
			if apierrors.IsNotFound(result.Err) {
				break
			}
			return result.Err
		}
		result.Node.Finalizers = []string{}
		_, err := client.NodeClient().Update(ctx, &result.Node, metav1.UpdateOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		err = client.NodeClient().Delete(ctx, result.Node.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func removeInitRequests(ctx context.Context) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	for result := range initrequest.NewLister().List(ctx) {
		if result.Err != nil {
			if apierrors.IsNotFound(result.Err) {
				break
			}
			return result.Err
		}
		result.InitRequest.Finalizers = []string{}
		_, err := client.InitRequestClient().Update(ctx, &result.InitRequest, metav1.UpdateOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		err = client.InitRequestClient().Delete(ctx, result.InitRequest.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func deleteCRDs(ctx context.Context, force bool) error {
	if !force {
		return nil
	}

	if err := removeVolumes(ctx); err != nil {
		return err
	}

	if err := removeDrives(ctx); err != nil {
		return err
	}

	if err := removeNodes(ctx); err != nil {
		return err
	}

	if err := removeInitRequests(ctx); err != nil {
		return err
	}

	driveCRDName := consts.DriveResource + "." + consts.GroupName
	err := k8s.CRDClient().Delete(ctx, driveCRDName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	volumeCRDName := consts.VolumeResource + "." + consts.GroupName
	err = k8s.CRDClient().Delete(ctx, volumeCRDName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	nodeCRDName := consts.NodeResource + "." + consts.GroupName
	err = k8s.CRDClient().Delete(ctx, nodeCRDName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	initRequestCRDName := consts.InitRequestResource + "." + consts.GroupName
	err = k8s.CRDClient().Delete(ctx, initRequestCRDName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}
