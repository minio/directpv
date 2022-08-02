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

package semver

import (
	"testing"
)

func TestSemVer(t *testing.T) {
	testSet := map[string]bool{
		// valid semvers
		"v0.0.4":                 true,
		"v1.2.3-1.1+build":       true,
		"v10.20.30-01xtrtrtr":    true,
		"v1.1.2-prerelease+meta": true,
		"v1.1.2+meta-":           true,
		"v1.1.2+meta-valid":      true,
		"v1.0.0-alpha":           true,
		"v1.0.0-beta":            true,
		"v1.0.0-alpha.beta":      true,
		"v1.0.0-alpha.beta.1":    true,
		"v1.0.0-alpha.1":         true,
		"v1.0.0-alpha0.valid":    true,
		"v1.0.0-alpha.0valid":    true,
		"v1.0.0-alpha-a.b-c-somethinglong+build.1-aef.1-its-okay": true,
		"v1.0.0-rc.1+build.1":                   true,
		"v2.0.0-rc.1+build.123":                 true,
		"v1.2.3-beta":                           true,
		"v10.2.3-DEV-SNAPSHOT":                  true,
		"v1.2.3-SNAPSHOT-123":                   true,
		"v1.0.0":                                true,
		"v2.0.0":                                true,
		"v1.1.7":                                true,
		"v2.0.0+build.1848":                     true,
		"v2.0.1-alpha.1227":                     true,
		"v1.0.0-alpha+beta":                     true,
		"v1.2.3----RC-SNAPSHOT.12.9.1--.12+788": true,
		"v1.2.3----R-S.12.9.1--.12+meta":        true,
		"v1.2.3----RC-SNAPSHOT.12.9.1--.12":     true,
		"v1.0.0+0.build.1-rc.10000aaa-kk-0.1":   true,
		"v99999999999999999999999.999999999999999999.99999999999999999": true,
		"v1.0.0-0A.is.legal": true,
		// invalid semvers
		"v1":                   false,
		"v1.2":                 false,
		"v1.2.3-0123":          false,
		"v1.2.3-0123.0123":     false,
		"v1.1.2+.123":          false,
		"v+invalid":            false,
		"v-invalid":            false,
		"v-invalid+invalid":    false,
		"v-invalid.01":         false,
		"valpha":               false,
		"valpha.beta":          false,
		"valpha.beta.1":        false,
		"valpha.1":             false,
		"valpha+beta":          false,
		"valpha_beta":          false,
		"valpha.":              false,
		"valpha..":             false,
		"vbeta":                false,
		"v1.0.0-alpha_beta":    false,
		"v-alpha.":             false,
		"v1.0.0-alpha..":       false,
		"v1.0.0-alpha..1":      false,
		"v1.0.0-alpha...1":     false,
		"v1.0.0-alpha....1":    false,
		"v1.0.0-alpha.....1":   false,
		"v1.0.0-alpha......1":  false,
		"v1.0.0-alpha.......1": false,
		"v01.1.1":              false,
		"v1.01.1":              false,
		"v1.1.01":              false,
		"v1.2.3.DEV":           false,
		"v1.2-SNAPSHOT":        false,
		"v1.2.31.2.3----RC-SNAPSHOT.12.09.1--..12+788": false,
		"v1.2-RC-SNAPSHOT":          false,
		"v-1.0.3-gamma+b7718":       false,
		"v+justmeta":                false,
		"v9.8.7+meta+meta":          false,
		"v9.8.7-whatever+meta+meta": false,
		"v99999999999999999999999.999999999999999999.99999999999999999----RC-SNAPSHOT.12.09.1--------------------------------..12": false,
	}

	for item, valid := range testSet {
		_, err := NewVersion(item)
		if err != nil && valid {
			t.Errorf("Parser failure: %s is a valid semver, but got result as invalid", item)
		}
		if err == nil && !valid {
			t.Errorf("Parser failure: %s is an invalid semver, but got result as valid", item)
		}
	}
}
