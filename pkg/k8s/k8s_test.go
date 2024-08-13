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

package k8s

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
)

func init() {
	FakeInit()
}

var (
	apiGroups = []*metav1.APIGroup{
		{
			Name: "policy",
			Versions: []metav1.GroupVersionForDiscovery{
				{
					GroupVersion: "policy/v1beta1",
					Version:      "v1beta1",
				},
			},
		},
		{
			Name: "storage.k8s.io",
			Versions: []metav1.GroupVersionForDiscovery{
				{
					GroupVersion: "storage.k8s.io/v1",
					Version:      "v1",
				},
			},
		},
	}

	apiResourceList = []*metav1.APIResourceList{
		{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "policy/v1beta1",
				Kind:       "PodSecurityPolicy",
			},
			GroupVersion: "policy/v1beta1",
			APIResources: []metav1.APIResource{
				{
					Name:       "policy",
					Group:      "policy",
					Version:    "v1beta1",
					Namespaced: false,
					Kind:       "PodSecurityPolicy",
				},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "storage.k8s.io/v1",
				Kind:       "CSIDriver",
			},
			GroupVersion: "storage.k8s.io/v1",
			APIResources: []metav1.APIResource{
				{
					Name:       "CSIDriver",
					Group:      "storage.k8s.io",
					Version:    "v1",
					Namespaced: false,
					Kind:       "CSIDriver",
				},
			},
		},
	}
)

func getDiscoveryGroupsAndMethods() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	return apiGroups, apiResourceList, nil
}

func TestVolumeStatusTransitions(t *testing.T) {
	statusList := []metav1.Condition{
		{
			Type:               "staged",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Type:               "published",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
		},
	}

	testCases := []struct {
		name       string
		condType   string
		condStatus metav1.ConditionStatus
	}{
		{
			name:       "NodeStageVolumeTransition",
			condType:   "staged",
			condStatus: metav1.ConditionTrue,
		},
		{
			name:       "NodePublishVolumeTransition",
			condType:   "published",
			condStatus: metav1.ConditionTrue,
		},
		{
			name:       "NodeUnpublishVolumeTransition",
			condType:   "published",
			condStatus: metav1.ConditionFalse,
		},
		{
			name:       "NodeUnstageVolumeTransition",
			condType:   "staged",
			condStatus: metav1.ConditionFalse,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			UpdateCondition(statusList, testCase.condType, testCase.condStatus, "", "")
			if !IsCondition(statusList, testCase.condType, testCase.condStatus, "", "") {
				t.Fatalf("case %v: Status transition failed (Type, Status) = (%s, %v) condition list: %v", testCase.name, testCase.condType, testCase.condStatus, statusList)
			}
		})
	}
}

func TestGetKubeVersion(t *testing.T) {
	testCases := []struct {
		info      version.Info
		major     uint
		minor     uint
		expectErr bool
	}{
		{version.Info{Major: "a", Minor: "0"}, 0, 0, true},                                   // invalid major
		{version.Info{Major: "-1", Minor: "0"}, 0, 0, true},                                  // invalid major
		{version.Info{Major: "0", Minor: "a"}, 0, 0, true},                                   // invalid minor
		{version.Info{Major: "0", Minor: "-1"}, 0, 0, true},                                  // invalid minor
		{version.Info{Major: "0", Minor: "-1", GitVersion: "commit-eks-id"}, 0, 0, true},     // invalid minor for eks
		{version.Info{Major: "0", Minor: "incompat", GitVersion: "commit-eks-"}, 0, 0, true}, // invalid minor for eks
		{version.Info{Major: "0", Minor: "0"}, 0, 0, false},
		{version.Info{Major: "1", Minor: "0"}, 1, 0, false},
		{version.Info{Major: "0", Minor: "1"}, 0, 1, false},
		{version.Info{Major: "1", Minor: "18"}, 1, 18, false},
		{version.Info{Major: "1", Minor: "18+", GitVersion: "commit-eks-id"}, 1, 18, false},
		{version.Info{Major: "1", Minor: "18-", GitVersion: "commit-eks-id"}, 1, 18, false},
		{version.Info{Major: "1", Minor: "18incompat", GitVersion: "commit-eks-id"}, 1, 18, false},
		{version.Info{Major: "1", Minor: "18-incompat", GitVersion: "commit-eks-id"}, 1, 18, false},
	}

	for i, testCase := range testCases {
		client.DiscoveryClient = NewFakeDiscovery(getDiscoveryGroupsAndMethods, &testCase.info)
		major, minor, err := client.GetKubeVersion()
		if testCase.expectErr {
			if err == nil {
				t.Fatalf("case %v: expected error, but succeeded", i+1)
			}
			continue
		}

		if err != nil {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}

		if major != testCase.major {
			t.Fatalf("case %v: major: expected: %v, got: %v", i+1, testCase.major, major)
		}

		if minor != testCase.minor {
			t.Fatalf("case %v: minor: expected: %v, got: %v", i+1, testCase.minor, minor)
		}
	}
}
