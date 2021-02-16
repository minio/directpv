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

package converter

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	directv1alpha1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

func TestConverter(t *testing.T) {
	sampleObj := `kind: ConversionReview
apiVersion: apiextensions.k8s.io/v1
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: direct.csi.min.io/v1test
  objects:
  - apiVersion: direct.csi.min.io/v1alpha1
    kind: DirectCSIDrive
    metadata:
      name: name-name
    spec:
      directCSIOwned: true
    status:
      path: /dev/xvdb
      nodeName: directcsinode-1
      driveStatus: Available
`
	response := httptest.NewRecorder()
	request, err := http.NewRequest("POST", driveHandlerPath, strings.NewReader(sampleObj))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Add("Content-Type", "application/yaml")
	ServeConversion(response, request)
	convertReview := v1.ConversionReview{}
	scheme := runtime.NewScheme()
	yamlSerializer := json.NewSerializerWithOptions(json.DefaultMetaFactory, scheme, scheme, json.SerializerOptions{Yaml: true})
	if _, _, err := yamlSerializer.Decode(response.Body.Bytes(), nil, &convertReview); err != nil {
		t.Fatalf("cannot decode data: \n %v\n Error: %v", response.Body, err)
	}
	if convertReview.Response.Result.Status != metav1.StatusSuccess {
		t.Fatalf("cr conversion failed: %v", convertReview.Response)
	}
	convertedObj := unstructured.Unstructured{}
	if _, _, err := yamlSerializer.Decode(convertReview.Response.ConvertedObjects[0].Raw, nil, &convertedObj); err != nil {
		t.Fatal(err)
	}
	if e, a := versionV1Test, convertedObj.GetAPIVersion(); e != a {
		t.Errorf("expected= %v, actual= %v", e, a)
	}

	var directCSIDrive directv1alpha1.DirectCSIDrive
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(convertedObj.Object, &directCSIDrive); err != nil {
		t.Errorf("Error while convertinng %v", err)
	}

	if directCSIDrive.Status.DriveStatus != directv1alpha1.DriveStatusUnavailable {
		t.Errorf("expected status.driveStatus = %v, actual status.driveStatus = %v", directv1alpha1.DriveStatusUnavailable, directCSIDrive.Status.DriveStatus)
	}

}
