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
	"fmt"
	"strings"
)

// SemVer denotes semantic version.
//
// Format RegEx:
// ^(?P<major>0|[1-9]\d*)
// \.(?P<minor>0|[1-9]\d*)
// \.(?P<patch>0|[1-9]\d*)
// (?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?
// (?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$
type SemVer struct {
	value string
}

func (v *SemVer) String() string {
	return v.value
}

// Compare compares two version.
func (v *SemVer) Compare(other *SemVer) int {
	return strings.Compare(v.value, other.value)
}

// NewVersion creates new semver.
func NewVersion(version string) (*SemVer, error) {
	versionNumParser := func(end1, end2 rune,
		followFn1, followFn2 func(rune) (bool, bool, interface{}),
		shouldContinue bool,
	) func(r rune) (bool, bool, interface{}) {
		// This is the definition of parseFunc
		// input is the current rune being parsed
		// return values are
		//  - validSoFar - indicates that the parsing has been successful so far
		//  - mustHaveMoreInput - indicates that the parsing cannot end at the current rune

		// var parseFn func(in rune) (validSoFar bool, mustHaveMoreInput bool)

		var parse func(r rune) (bool, bool, interface{})
		var parseEnd func(r rune) (bool, bool, interface{})
		var parseStart func(r rune) (bool, bool, interface{})

		parse = func(r rune) (bool, bool, interface{}) {
			if r >= '0' && r <= '9' {
				return true, shouldContinue, parse
			}
			if r == end1 {
				return true, shouldContinue, followFn1
			}
			if r == end2 {
				return true, shouldContinue, followFn2
			}
			return false, false, parse
		}
		parseEnd = func(r rune) (bool, bool, interface{}) {
			if r == end1 {
				return true, shouldContinue, followFn1
			}
			if r == end2 {
				return true, shouldContinue, followFn2
			}
			return false, false, parseEnd
		}

		parseStart = func(r rune) (bool, bool, interface{}) {
			if r == '0' {
				return true, shouldContinue, parseEnd
			}
			if r >= '1' && r <= '9' {
				return true, shouldContinue, parse
			}
			return false, false, parseStart
		}
		return parseStart
	}

	prereleaseParser := func(parseBuildMeta func(r rune) (bool, bool, interface{})) func(r rune) (bool, bool, interface{}) {
		// This is the definition of parseFunc
		// input is the current rune being parsed
		// return values are
		//  - validSoFar - indicates that the parsing has been successful so far
		//  - mustHaveMoreInput - indicates that the parsing cannot end at the current rune
		var parsePrereleaseStart func(in rune) (validSoFar bool, mustHaveMoreInput bool, resp interface{})
		var parsePrereleaseZero func(in rune) (validSoFar bool, mustHaveMoreInput bool, resp interface{})
		var parsePrerelease func(in rune) (validSoFar bool, mustHaveMoreInput bool, resp interface{})
		var parsePrereleaseMustAlphaNumeric func(in rune) (validSoFar bool, mustHaveMoreInput bool, resp interface{})

		parsePrerelease = func(r rune) (bool, bool, interface{}) {
			if r == '-' {
				return true, false, parsePrerelease
			}
			if r >= '0' && r <= '9' {
				return true, false, parsePrerelease
			}
			if r >= 'a' && r <= 'z' {
				return true, false, parsePrerelease
			}
			if r >= 'A' && r <= 'Z' {
				return true, false, parsePrerelease
			}
			if r == '.' {
				return true, true, parsePrereleaseStart
			}
			if r == '+' {
				return true, true, parseBuildMeta
			}
			return false, false, parsePrerelease
		}

		parsePrereleaseMustAlphaNumeric = func(r rune) (bool, bool, interface{}) {
			if r >= '0' && r <= '9' {
				return true, true, parsePrereleaseMustAlphaNumeric
			}
			if r == '-' {
				return true, false, parsePrerelease
			}
			if r >= 'a' && r <= 'z' {
				return true, false, parsePrerelease
			}
			if r >= 'A' && r <= 'Z' {
				return true, false, parsePrerelease
			}
			return false, false, parsePrereleaseMustAlphaNumeric
		}

		parsePrereleaseZero = func(r rune) (bool, bool, interface{}) {
			if r == '.' {
				return true, true, parsePrereleaseStart
			}
			if r == '+' {
				return true, true, parseBuildMeta
			}
			if r == '-' {
				return true, false, parsePrerelease
			}
			if r >= '1' && r <= '9' {
				return true, true, parsePrereleaseMustAlphaNumeric
			}
			if r >= 'a' && r <= 'z' {
				return true, false, parsePrerelease
			}
			if r >= 'A' && r <= 'Z' {
				return true, false, parsePrerelease
			}
			return false, false, parsePrereleaseZero
		}

		parsePrereleaseStart = func(r rune) (bool, bool, interface{}) {
			if r == '+' {
				return true, true, parseBuildMeta
			}
			if r == '-' {
				return true, false, parsePrerelease
			}
			if r == '0' {
				return true, false, parsePrereleaseZero
			}
			if r >= 'A' && r <= 'Z' {
				return true, false, parsePrerelease
			}
			if r >= 'a' && r <= 'z' {
				return true, false, parsePrerelease
			}
			if r >= '1' && r <= '9' {
				return true, false, parsePrerelease
			}
			return false, false, parsePrereleaseStart
		}
		return parsePrereleaseStart
	}

	var parseBuildMeta func(rune) (bool, bool, interface{})
	var parseBuildMetaBody func(rune) (bool, bool, interface{})
	var parseBuildMetaDot func(rune) (bool, bool, interface{})

	parseBuildMetaDot = func(r rune) (bool, bool, interface{}) {
		if r == '-' {
			return true, false, parseBuildMetaBody
		}
		if r >= 'a' && r <= 'z' {
			return true, false, parseBuildMetaBody
		}
		if r >= 'A' && r <= 'Z' {
			return true, false, parseBuildMetaBody
		}
		if r >= '0' && r <= '9' {
			return true, false, parseBuildMetaBody
		}
		return false, false, parseBuildMetaDot
	}

	parseBuildMetaBody = func(r rune) (bool, bool, interface{}) {
		if r == '.' {
			return true, true, parseBuildMetaDot
		}
		if r == '-' {
			return true, false, parseBuildMetaBody
		}
		if r >= 'a' && r <= 'z' {
			return true, false, parseBuildMetaBody
		}
		if r >= 'A' && r <= 'Z' {
			return true, false, parseBuildMetaBody
		}
		if r >= '0' && r <= '9' {
			return true, false, parseBuildMetaBody
		}
		return false, false, parseBuildMetaBody
	}

	parseBuildMeta = func(r rune) (bool, bool, interface{}) {
		if r == '-' {
			return true, false, parseBuildMetaBody
		}
		if r >= 'a' && r <= 'z' {
			return true, false, parseBuildMetaBody
		}
		if r >= 'A' && r <= 'Z' {
			return true, false, parseBuildMetaBody
		}
		if r >= '0' && r <= '9' {
			return true, false, parseBuildMetaBody
		}
		return false, false, parseBuildMeta
	}

	parsePrerelease := prereleaseParser(parseBuildMeta)
	parsePatch := versionNumParser('-', '+', parsePrerelease, parseBuildMeta, false)
	parseMinor := versionNumParser('.', '.', parsePatch, parsePatch, true)
	parseMajor := versionNumParser('.', '.', parseMinor, parseMinor, true)

	var parseV func(rune) (bool, bool, interface{})
	parseV = func(r rune) (bool, bool, interface{}) {
		if r == 'v' {
			return true, true, parseMajor
		}
		return false, false, parseV
	}

	var parseFn func(rune) (bool, bool, interface{})
	parseFn = parseV

	var valid, shouldContinue bool
	var parseFnInt interface{}

	for _, r := range version {
		valid, shouldContinue, parseFnInt = parseFn(r)
		if !valid {
			return nil, fmt.Errorf("invalid semver value %v", version)
		}

		parseFn = parseFnInt.(func(rune) (bool, bool, interface{}))
	}
	if shouldContinue {
		return nil, fmt.Errorf("invalid semver value %v", version)
	}
	return &SemVer{
		value: version,
	}, nil
}
