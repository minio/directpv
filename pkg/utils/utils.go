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
	if _, err = writer.Write([]byte("\n---\n")); err != nil {
		return err
	}
	return nil
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

func TrimDevPrefix(name string) string {
	return strings.TrimPrefix(name, "/dev/")
}

func AddDevPrefix(name string) string {
	if strings.HasPrefix(name, "/dev/") {
		return name
	}

	return "/dev/" + name
}
