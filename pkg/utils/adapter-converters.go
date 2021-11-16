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

	runtime "k8s.io/apimachinery/pkg/runtime"
)

func ErrUnsupportedConversion(from, to string) error {
	return fmt.Errorf("Unsupported conversion %s -> %s", from, to)
}

type conversionFunc func(runtime.Object) (runtime.Object, error)

func noop(from runtime.Object) (runtime.Object, error) {
	return nil, nil
}

func runConversionChain(obj runtime.Object, converters []conversionFunc) (runtime.Object, error) {
	result := obj.DeepCopyObject()
	for _, converter := range converters {
		intermediateResult, err := converter(result)
		if err != nil {
			return nil, err
		}
		result = intermediateResult.DeepCopyObject()
	}
	return result, nil
}
