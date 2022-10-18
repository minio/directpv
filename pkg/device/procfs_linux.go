//go:build linux

// This file is part of MinIO DirectPV
// Copyright (c) 2022 MinIO, Inc.
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

package device

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"
)

func parseCDROMs(r io.Reader) (map[string]struct{}, error) {
	reader := bufio.NewReader(r)
	names := map[string]struct{}{}
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		if tokens := strings.SplitAfterN(s, "drive name:", 2); len(tokens) == 2 {
			for _, token := range strings.Fields(tokens[1]) {
				if token != "" {
					names[token] = struct{}{}
				}
			}
			break
		}
	}
	return names, nil
}

func getCDROMs() (map[string]struct{}, error) {
	file, err := os.Open("/proc/sys/dev/cdrom/info")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]struct{}{}, nil
		}
		return nil, err
	}

	defer file.Close()
	return parseCDROMs(file)
}

func parseSwaps(r io.Reader) (map[string]struct{}, error) {
	reader := bufio.NewReader(r)

	filenames := map[string]struct{}{}
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		filenames[strings.Fields(s)[0]] = struct{}{}
	}

	return filenames, nil
}

func getSwaps() (map[string]struct{}, error) {
	file, err := os.Open("/proc/swaps")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]struct{}{}, nil
		}
		return nil, err
	}

	defer file.Close()
	return parseSwaps(file)
}
