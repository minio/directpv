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
	"bytes"
	"testing"

	"github.com/minio/directpv/pkg/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWriteObject(t *testing.T) {
	var byteBuffer bytes.Buffer
	meta := metav1.ObjectMeta{
		Name:      consts.GroupName,
		Namespace: consts.GroupName,
		Annotations: map[string]string{
			consts.GroupName + "/created-by": "kubectl/directpv",
		},
		Labels: map[string]string{
			"app":  consts.GroupName,
			"type": "CSIDriver",
		},
	}
	testCases := []struct {
		input       interface{}
		errReturned bool
	}{
		{
			input: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: meta,
				Spec: corev1.NamespaceSpec{
					Finalizers: []corev1.FinalizerName{},
				},
				Status: corev1.NamespaceStatus{},
			},
			errReturned: false,
		},
		{[]string{"1"}, false},
	}
	for i, test := range testCases {
		err := WriteObject(&byteBuffer, testCases[i].input)
		errReturned := err != nil
		if errReturned != test.errReturned {
			t.Fatalf("Test %d: expected %t got %t", i+1, test.errReturned, errReturned)
		}
	}
}
