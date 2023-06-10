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
	"sort"
	"strings"
)

func readdirnames(dirname string, errorIfNotExist bool) ([]string, error) {
	dir, err := os.Open(dirname)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !errorIfNotExist {
			err = nil
		}
		return nil, err
	}
	defer dir.Close()
	return dir.Readdirnames(-1)
}

func readFirstLine(filename string) (string, error) {
	getError := func(err error) error {
		switch {
		case errors.Is(err, os.ErrNotExist), errors.Is(err, os.ErrInvalid):
			return nil
		case strings.Contains(strings.ToLower(err.Error()), "no such device"):
			return nil
		case strings.Contains(strings.ToLower(err.Error()), "invalid argument"):
			return nil
		}
		return err
	}

	file, err := os.Open(filename)
	if err != nil {
		return "", getError(err)
	}
	defer file.Close()
	s, err := bufio.NewReader(file).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", getError(err)
	}
	return strings.TrimSpace(s), nil
}

func toSlice(m map[string]string, separator string) []string {
	keys := []string{}
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := []string{}
	for _, key := range keys {
		result = append(result, key+separator+m[key])
	}
	return result
}
