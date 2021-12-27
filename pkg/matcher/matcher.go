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
	"path/filepath"
	"strings"

	"github.com/mb0/glob"
	"github.com/minio/directpv/pkg/sys"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func fmap(slice []string, applyFunc func(string) string) (result []string) {
	for _, value := range slice {
		result = append(result, applyFunc(value))
	}
	return
}

// GlobMatch matches given name in list of glob patterns.
func GlobMatch(name string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}

	for _, pattern := range patterns {
		if matched, _ := glob.Match(pattern, name); matched {
			return true
		}
	}
	return false
}

// GlobMatchNodesDrivesStatuses matches node/drive/status in list of nodes/drives/statuses glob patterns.
func GlobMatchNodesDrivesStatuses(nodes, drives, statuses []string, node, drive, status string) bool {
	matchGlob := func(patterns []string, value string, applyFunc func(src string) string) bool {
		if applyFunc != nil {
			patterns = fmap(patterns, applyFunc)
			value = applyFunc(value)
		}

		return GlobMatch(value, patterns)
	}

	getDriveName := func(d string) string {
		d = strings.ReplaceAll(d, sys.DirectCSIPartitionInfix, "")
		d = strings.ReplaceAll(d, sys.DirectCSIDevRoot+"/", "")
		d = strings.ReplaceAll(d, sys.HostDevRoot+"/", "")
		return filepath.Base(d)
	}

	switch {
	case !matchGlob(nodes, node, nil):
		return false
	case !matchGlob(drives, drive, getDriveName):
		return false
	case !matchGlob(statuses, status, strings.ToLower):
		return false
	default:
		return true
	}
}

// StringIn checks whether value in the slice.
func StringIn(slice []string, value string) bool {
	for _, s := range slice {
		if value == s {
			return true
		}
	}

	return false
}

// MatchTrueConditions matches whether types and status list are in a true conditions or not.
func MatchTrueConditions(conditions []metav1.Condition, types, statusList []string) bool {
	statusList = fmap(statusList, strings.ToLower)
	types = fmap(types, strings.ToLower)
	statusMatches := 0
	for _, condition := range conditions {
		ctype := strings.ToLower(condition.Type)
		if condition.Status == metav1.ConditionTrue && StringIn(types, ctype) && StringIn(statusList, ctype) {
			statusMatches++
		}
	}
	return statusMatches == len(statusList)
}
