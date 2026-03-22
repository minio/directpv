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

package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"sigs.k8s.io/yaml"
)

var (
	uuidRegex = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
	byteUnits = []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"}
)

// IsUUID checks whether value is UUID string.
func IsUUID(value string) bool {
	return uuidRegex.MatchString(value)
}

// Contains checks whether value in the slice.
func Contains[ctype comparable](slice []ctype, value ctype) bool {
	for _, s := range slice {
		if value == s {
			return true
		}
	}

	return false
}

// ToYAML converts any type to YAML
func ToYAML(i interface{}) ([]byte, error) {
	data, err := yaml.Marshal(i)
	if err != nil {
		return nil, err
	}

	return append(data, []byte("\n---\n")...), nil
}

// ToJSON converts any type to JSON
func ToJSON(obj interface{}) ([]byte, error) {
	return json.MarshalIndent(obj, "", "  ")
}

// SafeFile is used to write the yaml
type SafeFile struct {
	filename string
	tempFile *os.File
}

// NewSafeFile returns new SafeFile
func NewSafeFile(filename string) (*SafeFile, error) {
	tempFile, err := os.CreateTemp(filepath.Dir(filename), "safefile.")
	if err != nil {
		return nil, err
	}
	return &SafeFile{
		tempFile: tempFile,
		filename: filename,
	}, nil
}

// Write writes to the file
func (safeFile *SafeFile) Write(p []byte) (int, error) {
	return safeFile.tempFile.Write(p)
}

// Close after writing to file
func (safeFile *SafeFile) Close() error {
	if err := safeFile.tempFile.Close(); err != nil {
		return err
	}
	return os.Rename(safeFile.tempFile.Name(), safeFile.filename)
}

// TrimDevPrefix trims dev directory prefix.
func TrimDevPrefix(name string) string {
	return strings.TrimPrefix(name, "/dev/")
}

// AddDevPrefix adds dev directory prefix.
func AddDevPrefix(name string) string {
	if strings.HasPrefix(name, "/dev/") {
		return name
	}

	return "/dev/" + name
}

// StringSet is set of strings.
type StringSet map[string]struct{}

// Set sets a string value.
func (set StringSet) Set(value string) {
	set[value] = struct{}{}
}

// Exist checks whether given value is in the set or not.
func (set StringSet) Exist(value string) (found bool) {
	_, found = set[value]
	return
}

// ToSlice converts set to slice of strings.
func (set StringSet) ToSlice() (values []string) {
	for value := range set {
		values = append(values, value)
	}
	return
}

// Equal checks whether given StringSet is same or not.
func (set StringSet) Equal(set2 StringSet) (found bool) {
	if len(set) != len(set2) {
		return false
	}

	for value := range set {
		if _, found := set2[value]; !found {
			return false
		}
	}

	return true
}

// IsEmpty returns true if the set is empty
func (set StringSet) IsEmpty() bool {
	return len(set) == 0
}

// Eprintf prints the message to the stdout and stderr based on inputs
func Eprintf(quiet, asErr bool, format string, a ...any) {
	if quiet {
		return
	}
	if asErr {
		fmt.Fprintf(os.Stderr, "%v ", color.RedString("ERROR"))
	}
	fmt.Fprintf(os.Stderr, format, a...)
}

// IBytes produces a human readable representation of an IEC size rounding to two decimal places.
func IBytes(ui64 uint64) string {
	value := ui64
	base := uint64(1)
	var unit string
	for _, unit = range byteUnits {
		if value < 1024 {
			break
		}
		value /= 1024
		base *= 1024
	}
	reminder := float64(ui64-(value*base)) / float64(base)

	rounded := uint64(100 * reminder)
	if rounded != 0 {
		return fmt.Sprintf("%v.%v %v", value, rounded, unit)
	}
	return fmt.Sprintf("%v %v", value, unit)
}
