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

package matcher

import (
	"reflect"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFmap(t *testing.T) {
	expectedResult := []string{"a", "b", "c"}
	result := fmap([]string{"A", "b", "C"}, strings.ToLower)
	if !reflect.DeepEqual(result, expectedResult) {
		t.Fatalf("expected: %v, got: %v", expectedResult, result)
	}
}

func TestGlobMatchNodesDrives(t *testing.T) {
	testCases := []struct {
		nodes          []string
		drives         []string
		node           string
		drive          string
		expectedResult bool
	}{
		{[]string{"worker1*"}, nil, "worker15", "", true},
		{[]string{"master*", "worker1*"}, nil, "master", "", true},
		{[]string{"worker1*"}, nil, "master", "", false},

		{nil, []string{"sd*"}, "", "sdaz", true},
		{nil, []string{"*1"}, "", "nvme0n1", true},
		{nil, []string{"*1"}, "", "nvme0n1p3", false},

		{[]string{"master*", "worker1*"}, []string{"sd*"}, "master", "sdb", true},
		{[]string{"master*", "worker1*"}, []string{"sd*"}, "master", "hdb", false},
	}

	for i, testCase := range testCases {
		result := GlobMatchNodesDrives(testCase.nodes, testCase.drives, testCase.node, testCase.drive)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v, got: %v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestStringIn(t *testing.T) {
	testCases := []struct {
		slice          []string
		value          string
		expectedResult bool
	}{
		{[]string{"A", "b", "C"}, "b", true},
		{[]string{}, "b", false},
		{[]string{"A", "b", "C"}, "B", false},
	}

	for i, testCase := range testCases {
		result := StringIn(testCase.slice, testCase.value)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v, got: %v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestGlobMatch(t *testing.T) {
	testCases := []struct {
		name           string
		patterns       []string
		expectedResult bool
	}{
		{"sda", nil, true},
		{"sda", []string{"sd*"}, true},
		{"sda", []string{"sd*", "hd*"}, true},
		{"sda", []string{"hd*"}, false},
	}

	for i, testCase := range testCases {
		result := GlobMatch(testCase.name, testCase.patterns)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v, got: %v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestMatchTrueConditions(t *testing.T) {
	conditions1 := []metav1.Condition{
		{Type: "One", Status: metav1.ConditionTrue},
		{Type: "Two", Status: metav1.ConditionFalse},
	}

	conditions2 := []metav1.Condition{
		{Type: "Published", Status: metav1.ConditionTrue},
		{Type: "Staged", Status: metav1.ConditionTrue},
	}

	testCases := []struct {
		conditions     []metav1.Condition
		types          []string
		statusList     []string
		expectedResult bool
	}{
		{conditions1, []string{"one"}, []string{"one"}, true},
		{conditions2, []string{"Published"}, []string{"published"}, true},
		{conditions2, []string{"Published", "staged"}, []string{"published", "staged"}, true},
		{conditions1, []string{"one"}, []string{"two"}, false},
		{conditions2, []string{"Published"}, []string{"staged"}, false},
	}

	for i, testCase := range testCases {
		result := MatchTrueConditions(testCase.conditions, testCase.types, testCase.statusList)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v, got: %v", i+1, testCase.expectedResult, result)
		}
	}
}
