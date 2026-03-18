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
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var alphaRegexp = regexp.MustCompile("^[a-z]+$")

func alpha2int(value string) (ui64 uint64) {
	p := uint64(1)
	for i := len(value) - 1; i >= 0; i-- {
		ui64 += uint64(value[i]-96) * p
		p *= 26
	}
	return ui64
}

func int2alpha(ui64 uint64) (value string) {
	for {
		r := ui64 % 26
		if r == 0 {
			r = 26
			ui64 -= 26
		}
		value = string(byte(r+96)) + value

		if ui64 < 26 {
			break
		}
		ui64 /= 26
	}

	return value
}

type ellipsis struct {
	start   uint64
	end     uint64
	isAlpha bool

	startIndex int
	endIndex   int

	current uint64
	prefix  string
	suffix  string
	next    *ellipsis
}

func (e *ellipsis) reset() {
	e.current = e.start
}

func (e *ellipsis) get(prefix string) string {
	if e.current > e.end {
		return ""
	}

	value := strconv.FormatUint(e.current, 10)
	if e.isAlpha {
		value = int2alpha(e.current)
	}
	value = prefix + e.prefix + value + e.suffix

	if e.next != nil {
		if newValue := e.next.get(value); newValue != "" {
			return newValue
		}

		e.next.reset()
		e.current++
		return e.get(prefix)
	}

	e.current++
	return value
}

func (e *ellipsis) expand() (result []string) {
	var value string
	for {
		if value = e.get(""); value == "" {
			break
		}
		result = append(result, value)
	}
	return result
}

func parseEllipsis(arg string, start, end int) (*ellipsis, error) {
	pattern := arg[start:end]
	parseValue := func(value string) (ui64 uint64, isAlpha bool, err error) {
		if ui64, err = strconv.ParseUint(value, 10, 64); err == nil {
			return ui64, false, nil
		}

		if alphaRegexp.MatchString(value) {
			return alpha2int(value), true, nil
		}
		return 0, false, err
	}

	tokens := strings.Split(arg[start+1:end-1], "...")
	switch len(tokens) {
	case 0, 1:
		return nil, fmt.Errorf("%v: invalid ellipsis %v at %v", arg, pattern, start)
	}

	startValue, isAlphaStart, err := parseValue(tokens[0])
	if err != nil {
		return nil, fmt.Errorf("%v: invalid start value '%v' in ellipsis %v at %v", arg, tokens[0], pattern, start)
	}

	endValue, isAlphaEnd, err := parseValue(tokens[1])
	if err != nil {
		return nil, fmt.Errorf("%v: invalid end value '%v' in ellipsis %v at %v", arg, tokens[1], pattern, start)
	}

	if isAlphaStart != isAlphaEnd {
		return nil, fmt.Errorf("%v: invalid ellipsis %v at %v; start/end must be same kind", arg, pattern, start)
	}

	if startValue > endValue {
		startValue, endValue = endValue, startValue
	}

	return &ellipsis{
		start:      startValue,
		end:        endValue,
		isAlpha:    isAlphaStart,
		startIndex: start,
		endIndex:   end,
		current:    startValue,
	}, nil
}

func getEllipses(arg string) (ellipses []*ellipsis, err error) {
	curlyOpened := false
	start := 0
	for i, c := range arg {
		switch c {
		case '{':
			if curlyOpened {
				return nil, fmt.Errorf("%v: nested ellipsis pattern at %v", arg, i+1)
			}

			curlyOpened = true
			start = i

		case '}':
			if !curlyOpened {
				return nil, fmt.Errorf("%v: invalid ellipsis pattern at %v", arg, i+1)
			}
			curlyOpened = false

			ellipsis, err := parseEllipsis(arg, start, i+1)
			if err != nil {
				return nil, err
			}

			ellipses = append(ellipses, ellipsis)
		}
	}

	return ellipses, nil
}

// Expand expends ellipses of given argument.
func Expand(arg string) ([]string, error) {
	ellipses, err := getEllipses(arg)
	if err != nil {
		return nil, err
	}

	if len(ellipses) == 0 {
		return []string{arg}, nil
	}

	startIndex := 0
	var prev *ellipsis
	for _, e := range ellipses {
		e.prefix = arg[startIndex:e.startIndex]
		if prev != nil {
			prev.next = e
		}
		startIndex = e.endIndex
		prev = e
	}
	ellipses[len(ellipses)-1].suffix = arg[startIndex:]

	return ellipses[0].expand(), nil
}
