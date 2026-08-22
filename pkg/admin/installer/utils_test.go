// This file is part of MinIO DirectPV
// Copyright (c) 2026 MinIO, Inc.
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
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestApplyResources(t *testing.T) {
	podSpec := &corev1.PodSpec{
		Containers: []corev1.Container{
			{Name: "container-1"},
			{Name: "container-2"},
		},
	}

	// Empty resources must leave the pod spec untouched.
	applyResources(podSpec, corev1.ResourceRequirements{})
	for i := range podSpec.Containers {
		if len(podSpec.Containers[i].Resources.Requests) != 0 || len(podSpec.Containers[i].Resources.Limits) != 0 {
			t.Fatalf("expected resources unchanged for container %v, got: %+v", podSpec.Containers[i].Name, podSpec.Containers[i].Resources)
		}
	}

	// Non-empty resources must be applied to all containers.
	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
	}
	applyResources(podSpec, resources)
	for i := range podSpec.Containers {
		containerResources := podSpec.Containers[i].Resources
		if cpu := containerResources.Requests[corev1.ResourceCPU]; cpu.Cmp(resource.MustParse("100m")) != 0 {
			t.Fatalf("expected cpu request 100m for container %v, got: %v", podSpec.Containers[i].Name, cpu)
		}
		if memory := containerResources.Limits[corev1.ResourceMemory]; memory.Cmp(resource.MustParse("1Gi")) != 0 {
			t.Fatalf("expected memory limit 1Gi for container %v, got: %v", podSpec.Containers[i].Name, memory)
		}
	}

	// Each container must own its resource lists; mutating one container's
	// resources must not affect the others.
	podSpec.Containers[0].Resources.Requests[corev1.ResourceCPU] = resource.MustParse("1")
	if cpu := podSpec.Containers[1].Resources.Requests[corev1.ResourceCPU]; cpu.Cmp(resource.MustParse("100m")) != 0 {
		t.Fatalf("expected container resources to be independent, got: %v", cpu)
	}
}
