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

package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
	"sigs.k8s.io/yaml"
)

const (
	defaultDirectCSIDir = ".direct-csi" // Default direct csi directory where direct csi audit logs are stored.
	auditDir            = "audit"       // Directory contains below files for audit logs
)

// Color print functions.
var (
	Bold   = color.New(color.Bold).SprintFunc()
	Red    = color.New(color.FgRed).SprintFunc()
	Yellow = color.New(color.FgYellow).SprintFunc()
)

// ToYAML converts value to YAML string.
func ToYAML(obj interface{}) (string, error) {
	data, err := yaml.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("unable to marshal object to YAML; %w", err)
	}
	return string(data), nil
}

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

type SafeFile struct {
	filename string
	tempFile *os.File
}

func (safeFile *SafeFile) Write(p []byte) (int, error) {
	return safeFile.tempFile.Write(p)
}

func (safeFile *SafeFile) Close() error {
	if err := safeFile.tempFile.Close(); err != nil {
		return err
	}
	return os.Rename(safeFile.tempFile.Name(), safeFile.filename)
}

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

func GetDefaultAuditDir() (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, defaultDirectCSIDir, auditDir), nil
}

func OpenAuditFile(auditFile string) (*SafeFile, error) {
	defaultAuditDir, err := GetDefaultAuditDir()
	if err != nil {
		return nil, fmt.Errorf("unable to get default audit directory ; %w", err)
	}
	if err := os.MkdirAll(defaultAuditDir, 0700); err != nil {
		return nil, fmt.Errorf("unable to create default audit directory : %w", err)
	}
	return NewSafeFile(filepath.Join(defaultAuditDir, fmt.Sprintf("%v-%v", auditFile, time.Now().UnixNano())))
}
