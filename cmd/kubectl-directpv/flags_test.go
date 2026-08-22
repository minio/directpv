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

package main

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestParseResourceRequirements(t *testing.T) {
	resetFlags := func() {
		cpuRequest = ""
		cpuLimit = ""
		memoryRequest = ""
		memoryLimit = ""
	}
	t.Cleanup(resetFlags)

	for _, tt := range []struct {
		name        string
		cpuRequest  string
		cpuLimit    string
		memoryReq   string
		memoryLimit string
		wantErr     bool
		errContains string
		check       func(t *testing.T, resources corev1.ResourceRequirements)
	}{
		{
			name: "no flags",
			check: func(t *testing.T, resources corev1.ResourceRequirements) {
				if len(resources.Requests) != 0 || len(resources.Limits) != 0 {
					t.Fatalf("expected empty resources, got: %+v", resources)
				}
			},
		},
		{
			name:        "valid requests and limits",
			cpuRequest:  "100m",
			cpuLimit:    "2",
			memoryReq:   "128Mi",
			memoryLimit: "4Gi",
			check: func(t *testing.T, resources corev1.ResourceRequirements) {
				if cpu := resources.Requests[corev1.ResourceCPU]; cpu.Cmp(resource.MustParse("100m")) != 0 {
					t.Fatalf("expected cpu request 100m, got: %v", cpu)
				}
				if cpu := resources.Limits[corev1.ResourceCPU]; cpu.Cmp(resource.MustParse("2")) != 0 {
					t.Fatalf("expected cpu limit 2, got: %v", cpu)
				}
				if memory := resources.Requests[corev1.ResourceMemory]; memory.Cmp(resource.MustParse("128Mi")) != 0 {
					t.Fatalf("expected memory request 128Mi, got: %v", memory)
				}
				if memory := resources.Limits[corev1.ResourceMemory]; memory.Cmp(resource.MustParse("4Gi")) != 0 {
					t.Fatalf("expected memory limit 4Gi, got: %v", memory)
				}
			},
		},
		{
			name:       "only request set",
			cpuRequest: "100m",
			check: func(t *testing.T, resources corev1.ResourceRequirements) {
				if len(resources.Requests) != 1 || len(resources.Limits) != 0 {
					t.Fatalf("expected only cpu request, got: %+v", resources)
				}
			},
		},
		{
			name:       "invalid quantity",
			cpuRequest: "abc",
			wantErr:    true,
		},
		{
			name:        "negative request",
			cpuRequest:  "-100m",
			wantErr:     true,
			errContains: "must not be negative",
		},
		{
			name:        "zero limit",
			cpuLimit:    "0",
			wantErr:     true,
			errContains: "must be greater than zero",
		},
		{
			name:        "zero memory limit",
			memoryLimit: "0",
			wantErr:     true,
			errContains: "must be greater than zero",
		},
		{
			name:        "request greater than limit",
			cpuRequest:  "2",
			cpuLimit:    "1",
			wantErr:     true,
			errContains: "must be less than or equal to",
		},
		{
			name:        "memory request greater than limit",
			memoryReq:   "4Gi",
			memoryLimit: "1Gi",
			wantErr:     true,
			errContains: "must be less than or equal to",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags()
			cpuRequest = tt.cpuRequest
			cpuLimit = tt.cpuLimit
			memoryRequest = tt.memoryReq
			memoryLimit = tt.memoryLimit

			resources, err := parseResourceRequirements()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got: %+v", resources)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("expected error containing %q, got: %v", tt.errContains, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.check(t, resources)
		})
	}
}
