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
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
	"sigs.k8s.io/yaml"

	directcsiv1beta1 "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta1"
	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
)

const (
	defaultDirectCSIDir = ".direct-csi" // Default direct csi directory where direct csi audit logs are stored.
	auditDir            = "audit"       // Directory contains below files for audit logs
)

// Color print functions.
var (
	Bold       = color.New(color.Bold).SprintFunc()
	Red        = color.New(color.FgRed).SprintFunc()
	Yellow     = color.New(color.FgYellow).SprintFunc()
	BinaryName = func() string {
		base := filepath.Base(os.Args[0])
		return strings.ReplaceAll(strings.ReplaceAll(base, "kubectl-", ""), "_", "-")
	}
	BinaryNameTransform = func(text string) string {
		transformed := &strings.Builder{}
		if err := template.Must(template.
			New("").Parse(text)).Execute(transformed, BinaryName()); err != nil {
			panic(err)
		}
		return transformed.String()
	}
)

// ToYAML converts value to YAML string.
func ToYAML(obj interface{}) (string, error) {
	data, err := yaml.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("unable to marshal object to YAML; %w", err)
	}
	return string(data), nil
}

//WriteObject writes the writer content
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

// GetDefaultAuditDir returns the default audit directory
func GetDefaultAuditDir() (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, defaultDirectCSIDir, auditDir), nil
}

// OpenAuditFile opens the file for writing
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

// GetMajorMinorFromStr returns the manjor minor number
func GetMajorMinorFromStr(majMin string) (major, minor uint32, err error) {
	tokens := strings.SplitN(majMin, ":", 2)
	if len(tokens) != 2 {
		err = fmt.Errorf("unknown format of %v", majMin)
		return
	}

	var major64, minor64 uint64
	major64, err = strconv.ParseUint(tokens[0], 10, 32)
	if err != nil {
		return
	}
	major = uint32(major64)

	minor64, err = strconv.ParseUint(tokens[1], 10, 32)
	minor = uint32(minor64)
	return
}

// IsV1Beta1Drive checks if the drive are of beta1 version
func IsV1Beta1Drive(drive *directcsi.DirectCSIDrive) bool {
	if labels := drive.GetLabels(); labels != nil {
		return labels[string(VersionLabelKey)] == directcsiv1beta1.Version
	}
	return false
}
