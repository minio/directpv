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

package converter

import (
	"fmt"

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	versionV1Alpha1 = "direct.csi.min.io/v1alpha1"
	versionV1Beta1  = "direct.csi.min.io/v1beta1"
	versionV1Beta2  = "direct.csi.min.io/v1beta2"
	versionV1Beta3  = "direct.csi.min.io/v1beta3"
	versionV1Beta4  = "direct.csi.min.io/v1beta4"
)

var supportedVersions = []string{
	versionV1Alpha1,
	versionV1Beta1,
	versionV1Beta2,
	versionV1Beta3,
	versionV1Beta4,
} // ordered

type crdKind string

const (
	driveCRDKind  crdKind = "DirectCSIDrive"
	volumeCRDKind crdKind = "DirectCSIVolume"
)

// convertFunc is the user defined function for any conversion. The code in this file is a
// template that can be use for any CR conversion given this function.
type convertFunc func(Object *unstructured.Unstructured, version string) (*unstructured.Unstructured, metav1.Status)

// conversionResponseFailureWithMessagef is a helper function to create an AdmissionResponse
// with a formatted embedded error message.
func conversionResponseFailureWithMessagef(msg string, params ...interface{}) *v1.ConversionResponse {
	return &v1.ConversionResponse{
		Result: metav1.Status{
			Message: fmt.Sprintf(msg, params...),
			Status:  metav1.StatusFailure,
		},
	}
}

func statusErrorWithMessage(msg string, params ...interface{}) metav1.Status {
	return metav1.Status{
		Message: fmt.Sprintf(msg, params...),
		Status:  metav1.StatusFailure,
	}
}

func statusSucceed() metav1.Status {
	return metav1.Status{
		Status: metav1.StatusSuccess,
	}
}
