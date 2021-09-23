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

package rangeexpander

import (
	"container/list"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var alphaRegexp = regexp.MustCompile("^[a-z]+$")

func (e *ellipsis) expand() (values []string) {
	start, end := e.start, e.end
	if start > end {
		start, end = end, start
	}

	for i := start; i <= end; i++ {
		if e.isAlpha {
			values = append(values, int2alpha(i))
		} else {
			values = append(values, fmt.Sprintf("%v", i))
		}
	}

	return values
}

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
		}
		value = string(byte(r+96)) + value

		if ui64 <= 26 {
			break
		}
		ui64 /= 26
	}

	return value
}

type ellipsis struct {
	start      uint64
	end        uint64
	isAlpha    bool
	startIndex int
	endIndex   int
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

	return &ellipsis{
		start:      startValue,
		end:        endValue,
		isAlpha:    isAlphaStart,
		startIndex: start,
		endIndex:   end,
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

// ExpandPatterns - expands the input to slice of string
func ExpandPatterns(input string) ([]string, error) {
	ellipses, err := getEllipses(input)
	if err != nil {
		return nil, err
	}
	// If Input : a{0..3}k{p...q}n then it is broken dow into slices as below
	// elipses : [[0,1,2,3],[p,q]] &&  nonEllipsis : [a,k,n]

	// create a slice of non ellipsis
	var nonEllipsis []string
	if len(ellipses) != 0 {
		prefix := input[0:ellipses[0].startIndex]
		if len(prefix) != 0 {
			nonEllipsis = append(nonEllipsis, prefix)
		}
		for i := range ellipses {
			if i < len(ellipses)-1 {
				mid := input[(*ellipses[i]).endIndex:(*ellipses[i+1]).startIndex]
				if len(mid) != 0 {
					nonEllipsis = append(nonEllipsis, mid)
				}
			}
		}
		suffix := input[ellipses[len(ellipses)-1].endIndex:]
		if len(suffix) != 0 {
			nonEllipsis = append(nonEllipsis, suffix)
		}
	} else if len(input) != 0 {
		nonEllipsis = append(nonEllipsis, input)
	}

	// Create a queue to expand pattern
	queue := list.New()

	// Input start with ellipsis
	if len(ellipses) != 0 && ellipses[0].startIndex == 0 {
		for _, val := range ellipses[0].expand() {
			queue.PushBack(val)
		}
		for i := range ellipses {
			size := queue.Len()
			for k := 0; k < size; k++ {
				if i < len(nonEllipsis) {
					populateQueue(queue, i, nil, nonEllipsis, true)
				}
			}
			size = queue.Len()
			for k := 0; k < size; k++ {
				if i < len(ellipses)-1 {
					populateQueue(queue, i, ellipses, nil, true)
				}
			}
		}
	} else if len(nonEllipsis) != 0 {
		queue.PushBack(nonEllipsis[0])
		for i := range nonEllipsis {
			size := queue.Len()
			for k := 0; k < size; k++ {
				if i < len(ellipses) {
					populateQueue(queue, i, ellipses, nil, false)
				}
			}
			size = queue.Len()
			for k := 0; k < size; k++ {
				if i < len(nonEllipsis)-1 {
					populateQueue(queue, i, nil, nonEllipsis, false)
				}
			}
		}
	}
	return queueToSlice(queue), nil
}

func populateQueue(queue *list.List, i int, e []*ellipsis, ne []string, startWithEllipsis bool) {
	if startWithEllipsis {
		if e != nil {
			front := queue.Front()
			for _, val := range e[i+1].expand() {
				queue.PushBack(front.Value.(string) + val)
			}
			queue.Remove(front)
		}
		if ne != nil {
			front := queue.Front()
			queue.PushBack(front.Value.(string) + ne[i])
			queue.Remove(front)
		}
	} else {
		if e != nil {
			front := queue.Front()
			for _, val := range e[i].expand() {
				queue.PushBack(front.Value.(string) + val)
			}
			queue.Remove(front)
		}
		if ne != nil {
			front := queue.Front()
			queue.PushBack(front.Value.(string) + ne[i+1])
			queue.Remove(front)
		}
	}
}

func queueToSlice(queue *list.List) []string {
	var result []string
	// Convert the queue to slice of string
	size := queue.Len()
	for k := 0; k < size; k++ {
		front := queue.Front()
		result = append(result, front.Value.(string))
		queue.Remove(front)
	}
	return result
}

// checks for equality of two slices of string
func testEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
