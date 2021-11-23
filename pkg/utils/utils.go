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

package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/sys"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var ( // Default direct csi directory where direct csi audit logs are stored.
	defaultDirectCSIDir = ".direct-csi"

	// Directory contains below files for audit logs
	auditDir = "audit"
)

// BoolToCondition converts boolean value to condition status.
func BoolToCondition(val bool) metav1.ConditionStatus {
	if val {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}

// DefaultIfZeroString returns string which is non empty of left or right.
func DefaultIfZeroString(left, right string) string {
	if left != "" {
		return left
	}
	return right
}

func getRootBlockFile(devName string) string {
	switch {
	case strings.HasPrefix(devName, sys.HostDevRoot):
		return devName
	case strings.Contains(devName, sys.DirectCSIDevRoot):
		return getRootBlockFile(filepath.Base(devName))
	default:
		name := strings.ReplaceAll(
			strings.Replace(devName, sys.DirectCSIPartitionInfix, "", 1),
			sys.DirectCSIPartitionInfix,
			sys.HostPartitionInfix,
		)
		return filepath.Join(sys.HostDevRoot, name)
	}
}

// GetDrivePath gets sanitized drive path.
func GetDrivePath(drive *directcsi.DirectCSIDrive) string {
	return getRootBlockFile(drive.Status.Path)
}

type SafeFile struct {
	filename string
	tempFile *os.File
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

func (safeFile *SafeFile) Write(p []byte) (int, error) {
	return safeFile.tempFile.Write(p)
}

func (safeFile *SafeFile) Close() error {
	if err := safeFile.tempFile.Close(); err != nil {
		return err
	}
	return os.Rename(safeFile.tempFile.Name(), safeFile.filename)
}

func NewSafeFile(fileName string) (*SafeFile, error) {
	tempFile, err := os.CreateTemp("", "safefile.")
	if err != nil {
		return nil, err
	}
	return &SafeFile{
		tempFile: tempFile,
		filename: fileName,
	}, nil
}

func getDirectCSIHomeDir() (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, defaultDirectCSIDir), nil
}

func GetDefaultAuditDir() (string, error) {
	defaultDir, err := getDirectCSIHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(defaultDir, auditDir), nil
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
