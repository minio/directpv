// This file is part of MinIO DirectPV
// Copyright (c) 2024 MinIO, Inc.
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

package admin

import (
	"context"
	"reflect"
	"testing"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRepairJobResources(t *testing.T) {
	k8s.FakeInit()
	client.FakeInit()

	ctx := context.Background()

	// Seed node-server daemonset from which the repair job inherits
	// container image and other parameters.
	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.NodeServerName,
			Namespace: consts.AppName,
			UID:       "fake-daemonset-uid",
		},
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  consts.NodeServerName,
							Image: "directpv:latest",
						},
					},
				},
			},
		},
	}
	if _, err := k8s.KubeClient().AppsV1().DaemonSets(consts.AppName).Create(ctx, daemonSet, metav1.CreateOptions{}); err != nil {
		t.Fatalf("unable to seed daemonset: %v", err)
	}

	// Seed a drive to be repaired.
	drive := types.NewDrive(
		directpvtypes.DriveID("drive-1"),
		types.DriveStatus{Status: directpvtypes.DriveStatusReady},
		directpvtypes.NodeID("node-1"),
		directpvtypes.DriveName("sda"),
		directpvtypes.AccessTierDefault,
	)
	if _, err := client.DriveClient().Create(ctx, drive, metav1.CreateOptions{}); err != nil {
		t.Fatalf("unable to seed drive: %v", err)
	}

	c := &Client{Client: client.GetClient()}
	results, err := c.Repair(
		ctx,
		RepairArgs{DriveIDs: []directpvtypes.DriveID{"drive-1"}},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected: 1 repair result, got: %v", len(results))
	}

	job, err := k8s.KubeClient().BatchV1().Jobs(consts.AppName).Get(ctx, "repair-drive-1", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("unable to get repair job: %v", err)
	}
	if len(job.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("expected: 1 container in repair job, got: %v", len(job.Spec.Template.Spec.Containers))
	}

	// The repair job container must declare resource requests and limits,
	// otherwise the job pod is rejected by ResourceQuota admission in the
	// "directpv" namespace.
	container := job.Spec.Template.Spec.Containers[0]
	if !reflect.DeepEqual(container.Resources, repairJobResourceRequirements) {
		t.Fatalf("expected resources: %+v, got: %+v", repairJobResourceRequirements, container.Resources)
	}
	if cpu := container.Resources.Requests.Cpu(); cpu == nil || cpu.Cmp(resource.MustParse("100m")) != 0 {
		t.Fatalf("expected cpu request: 100m, got: %v", cpu)
	}
	if memory := container.Resources.Requests.Memory(); memory == nil || memory.Cmp(resource.MustParse("128Mi")) != 0 {
		t.Fatalf("expected memory request: 128Mi, got: %v", memory)
	}
	if cpu := container.Resources.Limits.Cpu(); cpu == nil || cpu.Cmp(resource.MustParse("1")) != 0 {
		t.Fatalf("expected cpu limit: 1, got: %v", cpu)
	}
	if memory := container.Resources.Limits.Memory(); memory == nil || memory.Cmp(resource.MustParse("1Gi")) != 0 {
		t.Fatalf("expected memory limit: 1Gi, got: %v", memory)
	}
}
