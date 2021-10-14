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

	yamlFormatter "sigs.k8s.io/yaml"
)

// MustYAML converts value to YAML string.
func MustYAML(obj interface{}) string {
	y, err := ToYAML(obj)
	if err != nil {
		panic(err)
	}
	return y
}

// ToYAML converts value to YAML string.
func ToYAML(obj interface{}) (string, error) {
	formattedObj, err := yamlFormatter.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("error marshaling to YAML: %v", err)
	}
	return string(formattedObj), nil
}

// LogYAML prints value as YAML string to standard output.
func LogYAML(obj interface{}) error {
	y, err := ToYAML(obj)
	if err != nil {
		return err
	}

	fmt.Print(string(y))
	fmt.Println()
	fmt.Println("---")
	fmt.Println()
	return nil
}
