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

	directv1beta1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

func TestV1alpha1ToV1beta1Upgrade(t *testing.T) {
	sampleObj := `kind: ConversionReview
apiVersion: apiextensions.k8s.io/v1
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: direct.csi.min.io/v1beta1
  objects:
  - apiVersion: direct.csi.min.io/v1alpha1
    kind: DirectCSIDrive
    metadata:
      creationTimestamp: "2021-02-25T09:06:13Z"
      generation: 1
    name: febe8228562efe81f487d7c83df22c990acbc790024dd1d1d4512f326dc46b12
    resourceVersion: "4642669"
    uid: e56f5721-cee4-46fc-85c7-652e29a7b087
    spec:
      directCSIOwned: false
    status:
      conditions:
      - lastTransitionTime: "2021-02-25T09:06:13Z"
        message: ""
        reason: NotAdded
        status: "False"
        type: Owned
      - lastTransitionTime: "2021-02-25T09:06:13Z"
        message: /var/lib/direct-csi/mnt/26b5e22d368f01dfbfa161c35b70d758f960a820d1b79f54950f4309f73be064
        reason: NotAdded
        status: "True"
        type: Mounted
      - lastTransitionTime: "2021-02-25T09:06:13Z"
        message: xfs
        reason: NotAdded
        status: "True"
        type: Formatted
      - lastTransitionTime: "2021-02-25T09:06:13Z"
        message: ""
        reason: Initialized
        status: "True"
        type: Initialized
      driveStatus: Available
      filesystem: xfs
      freeCapacity: 992712667136
      logicalBlockSize: 512
      mountOptions:
      - rw
      - relatime
      mountpoint: /var/lib/direct-csi/mnt/26b5e22d368f01dfbfa161c35b70d758f960a820d1b79f54950f4309f73be064
      nodeName: minio-k8s6
      path: /var/lib/direct-csi/devices/nvme1n-part-1
      physicalBlockSize: 512
      rootPartition: nvme1n1
      topology:
        direct.csi.min.io/identity: direct-csi-min-io
        direct.csi.min.io/node: minio-k8s6
        direct.csi.min.io/rack: default
        direct.csi.min.io/region: default
        direct.csi.min.io/zone: default
      totalCapacity: 1000204886017
`
	response := httptest.NewRecorder()
	request, err := http.NewRequest("POST", DriveHandlerPath, strings.NewReader(sampleObj))
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
	if e, a := versionV1Beta1, convertedObj.GetAPIVersion(); e != a {
		t.Errorf("expected= %v, actual= %v", e, a)
	}

	var directCSIDrive directv1beta1.DirectCSIDrive
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(convertedObj.Object, &directCSIDrive); err != nil {
		t.Errorf("Error while convertinng %v", err)
	}

	if directCSIDrive.Status.AccessTier != directv1beta1.AccessTierUnknown {
		t.Errorf("expected status.accessTier = %v, actual status.accessTier = %v", directv1beta1.AccessTierUnknown, directCSIDrive.Status.AccessTier)
	}

}
