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

package main

import (
	"testing"
)

func TestValidOrg(t1 *testing.T) {
	testCases := []struct {
		name       string
		orgList    []string
		shouldFail bool
	}{
		{
			name:       "TestValidOrgs",
			orgList:    []string{"orgname", "org-name", "org-name-123", "orgName", "Org123-Name", "org/name", "org/sub/name", "test__org"},
			shouldFail: false,
		},
		{
			name:       "TestInvalidOrgs",
			orgList:    []string{"123", "123oef$%@", "_org", "123_we", "org//", "org/a/", "test___org", "test..org"},
			shouldFail: true,
		},
	}

	for _, tt := range testCases {
		t1.Run(tt.name, func(t1 *testing.T) {
			for _, org := range tt.orgList {
				err := validOrg(org)
				if tt.shouldFail && err == nil {
					t1.Fatalf("validOrg(%s) expected to fail but succeeded", org)
				}
				if !tt.shouldFail && err != nil {
					t1.Fatalf("validOrg(%s) expected to succeed but failed with: %v", org, err)
				}
			}
		})
	}
}

func TestValidRegistry(t1 *testing.T) {
	testCases := []struct {
		name       string
		regList    []string
		shouldFail bool
	}{
		{
			name:       "TestValidRegistries",
			regList:    []string{"reg:8000", "registry.in", "registry-private.domain", "registry-private.domain:8080"},
			shouldFail: false,
		},
		{
			name:       "TestInvalidRegistries",
			regList:    []string{"test_", "_reg.domain", "registry:123456"},
			shouldFail: true,
		},
	}

	for _, tt := range testCases {
		t1.Run(tt.name, func(t1 *testing.T) {
			for _, reg := range tt.regList {
				err := validRegistry(reg)
				if tt.shouldFail && err == nil {
					t1.Fatalf("validRegistry(%s) expected to fail but succeeded", reg)
				}
				if !tt.shouldFail && err != nil {
					t1.Fatalf("validRegistry(%s) expected to succeed but failed with: %v", reg, err)
				}
			}
		})
	}
}
