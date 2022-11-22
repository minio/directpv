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

package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	directpv "github.com/minio/directpv/pkg/apis/directpv.min.io/v1beta1"
	"github.com/minio/directpv/pkg/k8s"
	directv1alpha1 "github.com/minio/directpv/pkg/legacy/apis/direct.csi.min.io/v1alpha1"
	directv1beta1 "github.com/minio/directpv/pkg/legacy/apis/direct.csi.min.io/v1beta1"
	directv1beta2 "github.com/minio/directpv/pkg/legacy/apis/direct.csi.min.io/v1beta2"
	directv1beta3 "github.com/minio/directpv/pkg/legacy/apis/direct.csi.min.io/v1beta3"
	directv1beta4 "github.com/minio/directpv/pkg/legacy/apis/direct.csi.min.io/v1beta4"
	directcsi "github.com/minio/directpv/pkg/legacy/apis/direct.csi.min.io/v1beta5"
	directcsiclient "github.com/minio/directpv/pkg/legacy/client"
	typeddirectcsi "github.com/minio/directpv/pkg/legacy/clientset/typed/direct.csi.min.io/v1beta5"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/klog/v2"
)

const (
	// DriveTypeMetaKind is for the kind
	DriveTypeMetaKind = "DirectPVDrive"
	// VolumeTypeMetaKind is for the volume kind
	VolumeTypeMetaKind = "DirectPVVolume"
	// APIVersion is for the version
	APIVersion = "directpv.min.io/v1beta1"
	// DirectCSIGroup is for the group
	DirectCSIGroup = "direct.csi.min.io"
)

var (
	directCSIDriveClient  typeddirectcsi.DirectCSIDriveInterface
	directCSIVolumeClient typeddirectcsi.DirectCSIVolumeInterface
	uuidRegex             = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
)

func init() {
	k8s.Init()

	var err error
	if directCSIDriveClient, err = directcsiclient.DirectCSIDriveInterfaceForConfig(k8s.KubeConfig()); err != nil {
		klog.Fatalf("unable to create new direct-csi drive interface; %v", err)
	}

	if directCSIVolumeClient, err = directcsiclient.DirectCSIVolumeInterfaceForConfig(k8s.KubeConfig()); err != nil {
		klog.Fatalf("unable to create new direct-csi volume interface; %v", err)
	}

	gvk, err := directcsiclient.GetGroupKindVersions(
		DirectCSIGroup,
		"DirectCSIDrive",
		"v1beta5",
		directv1beta4.Version,
		directv1beta3.Version,
		directv1beta2.Version,
		directv1beta1.Version,
		directv1alpha1.Version,
	)
	if err != nil {
		klog.Fatalf("unable to get GroupKindVersions of direct-csi; %v", err)
	}

	if gvk.Group != DirectCSIGroup {
		klog.Fatalf("migration does not support direct-csi group %v", gvk.Group)
	}

	switch gvk.Version {
	case "v1beta3", "v1beta4", "v1beta5":
	default:
		klog.Fatalf("migration does not support direct-csi version %v", gvk.Version)
	}
}

// Preparing a map[driveName]fsuuid
var driveFsuuid = make(map[string]string)

func migrateVolumeCRD(oldVolumeList []directcsi.DirectCSIVolume) (newVolumeList []directpv.DirectPVVolume, theerror error) {
	newVolumes := []directpv.DirectPVVolume{}
	for _, OldVolume := range oldVolumeList {
		// FIXME: use types.NewDrive() instead of creating structure.
		NewVolume := directpv.DirectPVVolume{}

		// TypeMeta fields
		NewVolume.TypeMeta.Kind = VolumeTypeMetaKind
		NewVolume.TypeMeta.APIVersion = APIVersion

		// ObjectMeta fields
		NewVolume.ObjectMeta = OldVolume.ObjectMeta

		// Status fields
		NewVolume.Status.DataPath = OldVolume.Status.HostPath
		NewVolume.Status.StagingTargetPath = OldVolume.Status.StagingPath
		NewVolume.Status.TargetPath = OldVolume.Status.ContainerPath
		NewVolume.Status.FSUUID = driveFsuuid[OldVolume.Status.Drive]
		NewVolume.Status.TotalCapacity = OldVolume.Status.TotalCapacity
		NewVolume.Status.AvailableCapacity = OldVolume.Status.AvailableCapacity
		NewVolume.Status.UsedCapacity = OldVolume.Status.UsedCapacity

		// Append the new drive to the list
		newVolumes = append(newVolumes, NewVolume)
	}
	return newVolumes, nil
}

func migrateDriveCRD(oldDriveList []directcsi.DirectCSIDrive) (newDriveList []directpv.DirectPVDrive, therror error) {
	// Set of unique FSUUIDs:
	setOfUniqueFSUUIDs := make(map[string]struct{})

	newDrives := []directpv.DirectPVDrive{}
	for _, OldDrive := range oldDriveList {

		// we can filter out the drives other than Ready and InUse
		if OldDrive.Status.DriveStatus != directcsi.DriveStatusReady && OldDrive.Status.DriveStatus != directcsi.DriveStatusInUse {
			continue // skip this iteration to filter out this state.
		}

		// FIXME: use types.NewVolume() instead of creating structure.
		NewDrive := directpv.DirectPVDrive{}

		// TypeMeta fields
		NewDrive.TypeMeta.Kind = DriveTypeMetaKind
		NewDrive.TypeMeta.APIVersion = APIVersion

		// ObjectMeta fields
		NewDrive.ObjectMeta = OldDrive.ObjectMeta

		// Status fields
		// Cast from old type to new type
		NewDrive.Status.Status = types.DriveStatusReady
		NewDrive.Status.TotalCapacity = OldDrive.Status.TotalCapacity
		NewDrive.Status.AllocatedCapacity = OldDrive.Status.AllocatedCapacity
		NewDrive.Status.FreeCapacity = OldDrive.Status.FreeCapacity
		NewDrive.Status.FSUUID = OldDrive.Status.FilesystemUUID
		NewDrive.Status.Topology = OldDrive.Status.Topology

		// The name of the new drive should be its FSUUID and it should not be empty
		if NewDrive.ObjectMeta.Name == NewDrive.Status.FSUUID && len(NewDrive.Status.FSUUID) > 0 {
			fmt.Println("Validate that the name of the new drive is its FSUUID")
			// We need to also add validation to check if this is unique among the entire drive list set for conversion.
			// https://stackoverflow.com/questions/9251234/how-to-check-the-uniqueness-inside-a-for-loop
			if _, ok := setOfUniqueFSUUIDs[NewDrive.Status.FSUUID]; ok {
				fmt.Println("FSUUID found already, we can't convert this drive unless is unique: ", NewDrive.Status.FSUUID)
			} else {
				fmt.Println("element not found, FSUUID is unique, proceed to be added/appended for conversion")
				setOfUniqueFSUUIDs[NewDrive.Status.FSUUID] = struct{}{}
				// Append the new drive to the list
				newDrives = append(newDrives, NewDrive)
				driveFsuuid[NewDrive.ObjectMeta.Name] = NewDrive.Status.FSUUID
			}
		}
	}

	// Return the new list of drives
	return newDrives, nil
}

func main() {
	// Then we list the drives
	// This is getting all the info from all the drives
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	filteredDrives := []directcsi.DirectCSIDrive{}
	for result := range ListDrives(ctx) {
		if result.Err != nil {
			klog.Fatalf("unable to get direct-csi drives; %v", result.Err)
		}

		// FIXME: ignore result.Drive if its source version is < v1beta3

		if !uuidRegex.MatchString(result.Drive.Status.FilesystemUUID) {
			klog.Fatalf("invalid FilesystemUUID %v in DirectCSIDrive %v", result.Drive.Status.FilesystemUUID, result.Drive.Name)
		}

		for _, finalizer := range result.Drive.Finalizers {
			if !strings.Contains(finalizer, DirectCSIGroup) {
				klog.Fatalf("invalid finalizer value %v in DirectCSIDrive %v", finalizer, result.Drive.Name)
			}
		}

		filteredDrives = append(filteredDrives, result.Drive)
	}

	// To convert from old CRDs to new CRDs
	newDriveCRD, _ := migrateDriveCRD(filteredDrives)
	// Output Drives as YAML
	for index, drive := range newDriveCRD {
		// Output as YAML
		yamlName := "drive-" + strconv.Itoa(index+1) + ".yaml"
		newFile, _ := os.Create(yamlName)
		y := printers.YAMLPrinter{}
		defer newFile.Close()
		createDrive := func() *directpv.DirectPVDrive {
			return &directpv.DirectPVDrive{
				TypeMeta:   drive.TypeMeta,
				ObjectMeta: drive.ObjectMeta,
				Spec:       drive.Spec,
				Status:     drive.Status,
			}
		}
		runTimeObject := []runtime.Object{
			createDrive(),
		}
		y.PrintObj(runTimeObject[0], newFile)
	}

	filteredVolumes := []directcsi.DirectCSIVolume{}
	for result := range ListVolumes(ctx) {
		if result.Err != nil {
			klog.Fatalf("unable to get direct-csi volumes; %v", result.Err)
		}
		filteredVolumes = append(filteredVolumes, result.Volume)
	}

	newVolumeCRD, _ := migrateVolumeCRD(filteredVolumes)

	// Output Volumes as YAML
	for index, volume := range newVolumeCRD {
		// Output as YAML
		yamlName := "volume-" + strconv.Itoa(index+1) + ".yaml"
		newFile, _ := os.Create(yamlName)
		y := printers.YAMLPrinter{}
		defer newFile.Close()
		createVolume := func() *directpv.DirectPVVolume {
			return &directpv.DirectPVVolume{
				TypeMeta:   volume.TypeMeta,
				ObjectMeta: volume.ObjectMeta,
				Status:     volume.Status,
			}
		}
		runTimeObject := []runtime.Object{
			createVolume(),
		}
		y.PrintObj(runTimeObject[0], newFile)
	}
}
