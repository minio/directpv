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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	homedir "github.com/mitchellh/go-homedir"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/sys"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var ( // Default direct csi directory where direct csi audit logs are stored.
	defaultDirectCsiDir = ".direct-csi"

	// Directory contains below files for audit logs
	auditDir = "audit"
)

func BoolToCondition(val bool) metav1.ConditionStatus {
	if val {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}

func ValidateAccessTier(at string) (directcsi.AccessTier, error) {
	switch directcsi.AccessTier(strings.Title(at)) {
	case directcsi.AccessTierWarm:
		return directcsi.AccessTierWarm, nil
	case directcsi.AccessTierHot:
		return directcsi.AccessTierHot, nil
	case directcsi.AccessTierCold:
		return directcsi.AccessTierCold, nil
	case directcsi.AccessTierUnknown:
		return directcsi.AccessTierUnknown, fmt.Errorf("Please set any one among ['hot','warm', 'cold']")
	default:
		return directcsi.AccessTierUnknown, fmt.Errorf("Invalid 'access-tier' value, Please set any one among ['hot','warm','cold']")
	}
}

func defaultIfZero(left, right interface{}) interface{} {
	lval := reflect.ValueOf(left)
	if lval.IsZero() {
		return right
	}
	return left
}

func DefaultIfZero(left, right interface{}) interface{} {
	return defaultIfZero(left, right)
}

func DefaultIfZeroString(left, right string) string {
	return defaultIfZero(left, right).(string)
}

func DefaultIfZeroInt(left, right int) int {
	return defaultIfZero(left, right).(int)
}

func DefaultIfZeroInt64(left, right int64) int64 {
	return defaultIfZero(left, right).(int64)
}

func DefaultIfZeroFloat(left, right float32) float32 {
	return defaultIfZero(left, right).(float32)
}

func DefaultIfZeroFloat64(left, right float64) float64 {
	return defaultIfZero(left, right).(float64)
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

func GetDrivePath(drive *directcsi.DirectCSIDrive) string {
	return getRootBlockFile(drive.Status.Path)
}

func getDefaultDirectCsiDir() (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, defaultDirectCsiDir), nil
}

func GetDefaultAuditDir() (string, error) {
	defaultDir, err := getDefaultDirectCsiDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(defaultDir, auditDir), nil
}

// Attempts to create all directories, ignores any permission denied errors.
func MkdirAllIgnorePerm(path string) error {
	err := os.MkdirAll(path, 0700)
	if err != nil && errors.Is(err, os.ErrPermission) {
		// It is possible in kubernetes like deployments this directory
		// is already mounted and is not writable, ignore any write errors.
		err = nil
	}
	return err
}

type SafeFile struct {
	Filename     string
	TempFilename string
	TempFile     *os.File
}

func (safeFile *SafeFile) Write(obj interface{}) error {
	y, err := ToYAML(obj)
	if err != nil {
		return err
	}
	y = y + "\n --- \n "
	if _, err := safeFile.TempFile.Write([]byte(y)); err != nil {
		return err
	}
	return nil
}

func (safeFile *SafeFile) Close() error {
	if err := safeFile.TempFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(safeFile.TempFilename, fmt.Sprintf("%v-%v", safeFile.Filename, time.Now().UnixNano())); err != nil {
		return err
	}
	return nil
}
