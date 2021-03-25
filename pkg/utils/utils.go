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
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/yaml"

	"github.com/golang/glog"

	"fmt"
)

func JSONifyAndLog(val interface{}) {
	jsonBytes, err := json.MarshalIndent(val, "", " ")
	if err != nil {
		return
	}
	glog.V(3).Infof(string(jsonBytes))
}

func BoolToCondition(val bool) metav1.ConditionStatus {
	if val {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}

func LogYAML(obj interface{}) error {
	y, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}
	PrintYaml(y)
	return nil
}

func PrintYaml(data []byte) {
	fmt.Print(string(data))
	fmt.Println()
	fmt.Println("---")
	fmt.Println()
}
