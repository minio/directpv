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

package ellipsis

import (
	"reflect"
	"testing"
)

func TestExpand(t *testing.T) {
	testCases := []struct {
		input       string
		output      []string
		errReturned bool
	}{
		// Valid case - Start with ellipsis
		{"{a...c}", []string{"a", "b", "c"}, false},
		// Valid case - Start with ellipsis
		{"{f...c}", []string{"c", "d", "e", "f"}, false},
		// Valid case - Start with ellipsis
		{"{az...bc}", []string{"az", "ba", "bb", "bc"}, false},
		// Valid case- Start with ellipsis
		{"{a...c}a", []string{"aa", "ba", "ca"}, false},
		// Valid case- Start with ellipsis
		{"{a...c}a1", []string{"aa1", "ba1", "ca1"}, false},
		// Valid case- Start with ellipsis
		{"{a...c}{0...2}", []string{"a0", "a1", "a2", "b0", "b1", "b2", "c0", "c1", "c2"}, false},
		// Valid case- Start with ellipsis
		{"{a...c}p{0...2}", []string{"ap0", "ap1", "ap2", "bp0", "bp1", "bp2", "cp0", "cp1", "cp2"}, false},
		// Valid case- Start with ellipsis
		{"{a...c}p{0...2}9", []string{"ap09", "ap19", "ap29", "bp09", "bp19", "bp29", "cp09", "cp19", "cp29"}, false},
		// Valid case- Start with ellipsis
		{"{a...c}p{0...2}9{d...a}", []string{"ap09a", "ap09b", "ap09c", "ap09d", "ap19a", "ap19b", "ap19c", "ap19d", "ap29a",
			"ap29b", "ap29c", "ap29d", "bp09a", "bp09b", "bp09c", "bp09d", "bp19a", "bp19b", "bp19c", "bp19d", "bp29a",
			"bp29b", "bp29c", "bp29d", "cp09a", "cp09b", "cp09c", "cp09d", "cp19a", "cp19b", "cp19c", "cp19d", "cp29a", "cp29b", "cp29c", "cp29d"}, false},
		// Valid case- Start with non-ellipsis
		{"abc", []string{"abc"}, false},
		// Valid case- Start with non-ellipsis
		{"ab{p...r}", []string{"abp", "abq", "abr"}, false},
		// Valid case- Start with non-ellipsis
		{"ab{p...r}1", []string{"abp1", "abq1", "abr1"}, false},
		// Valid case- Start with non-ellipsis
		{"ab{p...r}0{1...2}", []string{"abp01", "abp02", "abq01", "abq02", "abr01", "abr02"}, false},
		// Valid case- ellipsis start with two digit
		{"a{12...20}x", []string{"a12x", "a13x", "a14x", "a15x", "a16x", "a17x", "a18x", "a19x", "a20x"}, false},
		// Valid case - ellipsis start with two digit end with two digits
		{"ax{ab...dx}y", []string{"axaby", "axacy", "axady", "axaey", "axafy", "axagy", "axahy", "axaiy", "axajy", "axaky",
			"axaly", "axamy", "axany", "axaoy", "axapy", "axaqy", "axary", "axasy", "axaty", "axauy", "axavy", "axawy", "axaxy", "axayy", "axazy",
			"axbay", "axbby", "axbcy", "axbdy", "axbey", "axbfy", "axbgy", "axbhy", "axbiy", "axbjy", "axbky", "axbly", "axbmy", "axbny", "axboy",
			"axbpy", "axbqy", "axbry", "axbsy", "axbty", "axbuy", "axbvy", "axbwy", "axbxy", "axbyy", "axbzy", "axcay", "axcby", "axccy", "axcdy",
			"axcey", "axcfy", "axcgy", "axchy", "axciy", "axcjy", "axcky", "axcly", "axcmy", "axcny", "axcoy", "axcpy", "axcqy", "axcry", "axcsy",
			"axcty", "axcuy", "axcvy", "axcwy", "axcxy", "axcyy", "axczy", "axday", "axdby", "axdcy", "axddy", "axdey", "axdfy", "axdgy", "axdhy",
			"axdiy", "axdjy", "axdky", "axdly", "axdmy", "axdny", "axdoy", "axdpy", "axdqy", "axdry", "axdsy", "axdty", "axduy", "axdvy", "axdwy", "axdxy"}, false},
		// Invalid case with one dot
		{"a{a.c}p", nil, true},
		// Invalid case - two dots
		{"a{a..c}p", nil, true},
		// Invalid case - four dots
		{"a{a....c}p", nil, true},
	}
	for i, test := range testCases {
		expansion, err := Expand(test.input)
		errReturned := err != nil
		if errReturned != test.errReturned {
			t.Fatalf("Test %d: expected %t got %t", i+1, test.errReturned, errReturned)
		}
		if !reflect.DeepEqual(expansion, test.output) {
			t.Fatalf("Test %d: expected %s got %s", i+1, test.output, expansion)
		}
	}
}

func TestGetEllipsis(t *testing.T) {
	testCases := []struct {
		arg         string
		ellipses    []*ellipsis
		errReturned bool
	}{
		// Valid case
		{"{a...z}", []*ellipsis{{start: 1, end: 26, isAlpha: true, startIndex: 0, endIndex: 7}}, false},
		// Valid case
		{"{aa...az}", []*ellipsis{{start: 27, end: 52, isAlpha: true, startIndex: 0, endIndex: 9}}, false},
		// Valid case
		{"{0...11}", []*ellipsis{{start: 0, end: 11, isAlpha: false, startIndex: 0, endIndex: 8}}, false},
		// Alpha numeric combination
		{"{a0...z}", []*ellipsis{}, true},
		// One dot in expansion
		{"{a.z}", []*ellipsis{}, true},
		// Two dot in expansion
		{"{a..z}", []*ellipsis{}, true},
		// Four or more dots in expansion
		{"{a....z}", []*ellipsis{}, true},
		// No dot in expansion
		{"{123}", []*ellipsis{}, true},
		// Multiple opening braces in ellipsis
		{"{a...{a...z}}", []*ellipsis{}, true},
		// No RHS
		{"{a...}z", []*ellipsis{}, true},
		// No LHS
		{"{...b}z", []*ellipsis{}, true},
		// Multiple openin braces
		{"{1.{...{zz}", []*ellipsis{}, true},
		// Invalid numer of braces
		{"1}ccc{sss}", []*ellipsis{}, true},
		// Alphabet in LHS number in RHS
		{"{11...az}", []*ellipsis{}, true},
		// Alphabet in LHS number in RHS
		{"{a...0}", []*ellipsis{}, true},
		// Number in LHS alphabet in RHS
		{"{0...a}", []*ellipsis{}, true},
		// alphabet in LHS and Number in RHS
		{"{a...0}", []*ellipsis{}, true},
	}

	for i, test := range testCases {
		ellipses, err := getEllipses(test.arg)
		errReturned := err != nil
		if errReturned != test.errReturned {
			t.Fatalf("Test %d: expected %t got %t", i+1, test.errReturned, errReturned)
		}

		for index, p := range ellipses {
			ts := test.ellipses[index]
			if p.start != ts.start {
				t.Fatalf("Test %d: expected %d got %d", i+1, ts.start, p.start)
			}
			if p.end != ts.end {
				t.Fatalf("Test %d: expected %d got %d", i+1, ts.end, p.end)
			}
			if p.isAlpha != ts.isAlpha {
				t.Fatalf("Test %d: expected %t got %t", i+1, ts.isAlpha, p.isAlpha)
			}
			if p.startIndex != ts.startIndex {
				t.Fatalf("Test %d: expected %d got %d", i+1, ts.startIndex, p.startIndex)
			}
			if p.endIndex != ts.endIndex {
				t.Fatalf("Test %d: expected %d got %d", i+1, ts.endIndex, p.endIndex)
			}
		}
	}
}
