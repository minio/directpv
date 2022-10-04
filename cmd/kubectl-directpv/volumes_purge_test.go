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
	"testing"

	"github.com/minio/directpv/pkg/client"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/volume"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
)

func TestVolumesPurge(t *testing.T) {
	createTestVolume := func(volumeName string) *types.Volume {
		return &types.Volume{
			TypeMeta: types.NewVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
				Finalizers: []string{
					string(consts.VolumeFinalizerPurgeProtection),
					string(consts.VolumeFinalizerPVProtection),
				},
			},
			Status: types.VolumeStatus{},
		}
	}

	createTestPV := func(pvName string, phase corev1.PersistentVolumePhase) *corev1.PersistentVolume {
		return &corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: pvName,
			},
			Status: corev1.PersistentVolumeStatus{
				Phase: phase,
			},
		}
	}

	testVolumeObjects := []runtime.Object{
		createTestVolume("volume-1"),
		createTestVolume("volume-2"),
		createTestVolume("volume-3"),
		createTestVolume("volume-4"),
	}

	testPVObjects := []runtime.Object{
		createTestPV("volume-1", corev1.VolumeReleased),
		createTestPV("volume-2", corev1.VolumeFailed),
		createTestPV("volume-3", corev1.VolumeBound),
	}

	if err := validateVolumeSelectors(); err != nil {
		t.Fatalf("validateVolumeSelectors failed with %v", err)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(testVolumeObjects...))
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())
	client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
	k8s.SetKubeInterface(kubernetesfake.NewSimpleClientset(testPVObjects...))

	if err := purgeVolumes(ctx, nil); err != nil {
		t.Fatal(err)
	}

	volumes, err := volume.GetVolumeList(ctx, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// all volumes except bound volume should be removed
	if len(volumes) != 1 {
		t.Fatalf("volume count: expected: 1, got: %v", len(volumes))
	}
}
