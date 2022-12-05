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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"sigs.k8s.io/yaml"
)

// Contains checks whether value in the slice.
func Contains[ctype comparable](slice []ctype, value ctype) bool {
	for _, s := range slice {
		if value == s {
			return true
		}
	}

	return false
}

// ToYAML converts value to YAML string.
func ToYAML(obj interface{}) (string, error) {
	data, err := yaml.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("unable to marshal object to YAML; %w", err)
	}
	return string(data), nil
}

// WriteObject writes the writer content
func WriteObject(writer io.Writer, obj interface{}) error {
	y, err := ToYAML(obj)
	if err != nil {
		return err
	}
	if _, err = writer.Write([]byte(y)); err != nil {
		return err
	}
	_, err = writer.Write([]byte("---\n"))
	return err
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

// Writes writes to the file
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
