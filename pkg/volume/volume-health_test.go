// This file is part of MinIO DirectPV
// Copyright (c) 2021, 2022, 2023 MinIO, Inc.
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

package volume

import (
	"context"
	"testing"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func newTestVolume(name string, stagingPath, containerPath string, errorCondition *metav1.Condition) *types.Volume {
	volume := &types.Volume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				string(directpvtypes.NodeLabelKey):      "test-node",
				string(directpvtypes.CreatedByLabelKey): consts.ControllerName,
			},
		},
		Status: types.VolumeStatus{
			StagingTargetPath: stagingPath,
			TargetPath:        containerPath,
			Conditions:        []metav1.Condition{},
		},
	}
	if errorCondition != nil {
		volume.Status.Conditions = append(volume.Status.Conditions, *errorCondition)
	}
	return volume
}

func newErrorCondition(hasError bool, message string) *metav1.Condition {
	condition := metav1.Condition{
		Type:    string(directpvtypes.VolumeConditionTypeError),
		Status:  metav1.ConditionFalse,
		Reason:  string(directpvtypes.VolumeConditionReasonNoError),
		Message: message,
	}
	if hasError {
		condition.Status = metav1.ConditionTrue
		condition.Reason = string(directpvtypes.VolumeConditionReasonNotMounted)
	}
	return &condition
}

func TestCheckVolumesHealth(t *testing.T) {
	objects := []runtime.Object{
		newTestVolume("volume-1", "/stagingpath/volume-1", "/containerpath/volume-1", nil),
		newTestVolume("volume-2", "/stagingpath/volume-2", "/containerpath/volume-2", newErrorCondition(false, "")),
		newTestVolume("volume-3", "/stagingpath/volume-3", "/containerpath/volume-3", newErrorCondition(false, "")),
		newTestVolume("volume-4", "/stagingpath/volume-4", "/containerpath/volume-4", newErrorCondition(false, "")),
		newTestVolume("volume-5", "/stagingpath/volume-5", "/containerpath/volume-5", newErrorCondition(false, "")),
	}

	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(objects...))
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	getMountsFn := func(volumeName string) (m utils.StringSet) {
		m = make(utils.StringSet)
		switch volumeName {
		case "volume-1":
			m["/stagingpath/volume-1"] = struct{}{}
			m["/containerpath/volume-1"] = struct{}{}
			return
		case "volume-2":
			m["/stagingpath/volume-2"] = struct{}{}
			m["/containerpath/volume-2"] = struct{}{}
			return
		case "volume-3":
			m["/containerpath/volume-3"] = struct{}{}
			return
		case "volume-4":
			m["/stagingpath/volume-4"] = struct{}{}
			return
		case "volume-5":
			m["/stagingpath/volume-x"] = struct{}{}
			m["/containerpath/volume-x"] = struct{}{}
			return
		default:
			return
		}
	}

	expectedErrorConditions := map[string]*metav1.Condition{
		"volume-1": newErrorCondition(false, ""),
		"volume-2": newErrorCondition(false, ""),
		"volume-3": newErrorCondition(true, string(directpvtypes.VolumeConditionMessageStagingPathNotMounted)),
		"volume-4": newErrorCondition(true, string(directpvtypes.VolumeConditionMessageTargetPathNotMounted)),
		"volume-5": newErrorCondition(true, string(directpvtypes.VolumeConditionMessageStagingPathNotMounted)),
	}

	if err := checkVolumesHealth(context.TODO(), directpvtypes.NodeID("test-node"), getMountsFn); err != nil {
		t.Fatalf("unable to check volumes health: %v", err)
	}

	for volumeName, condition := range expectedErrorConditions {
		volume, err := client.VolumeClient().Get(context.TODO(), volumeName, metav1.GetOptions{TypeMeta: types.NewVolumeTypeMeta()})
		if err != nil {
			t.Fatalf("Error while getting the volume %v: %+v", volume.Name, err)
		}
		errorCondition := k8s.GetConditionByType(volume.Status.Conditions, string(directpvtypes.VolumeConditionTypeError))
		if errorCondition == nil {
			t.Fatalf("[volume: %s] Expected error condition but got nil", volumeName)
		}
		if errorCondition.Status != condition.Status {
			t.Fatalf("[volume: %s] Expected condition status %v but got %v", volumeName, condition.Status, errorCondition.Status)
		}
		if errorCondition.Reason != condition.Reason {
			t.Fatalf("[volume: %s] Expected condition reason %v but got %v", volumeName, condition.Reason, errorCondition.Reason)
		}
		if errorCondition.Message != condition.Message {
			t.Fatalf("[volume: %s] Expected condition message %v but got %v", volumeName, condition.Message, errorCondition.Message)
		}
	}
}
