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
	"reflect"
	"strings"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
