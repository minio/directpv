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

	"github.com/minio/direct-csi/pkg/utils"
)

func TestV1alpha1ToV1beta1DriveUpgrade(t *testing.T) {
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
	ServeDriveConversion(response, request)
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
		t.Errorf("Error while converting %v", err)
	}

	if directCSIDrive.Status.AccessTier != directv1beta1.AccessTierUnknown {
		t.Errorf("expected status.accessTier = %v, actual status.accessTier = %v", directv1beta1.AccessTierUnknown, directCSIDrive.Status.AccessTier)
	}

}

func TestV1alpha1ToV1beta1VolumeUpgrade(t *testing.T) {
	sampleObj := `kind: ConversionReview
apiVersion: apiextensions.k8s.io/v1
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: direct.csi.min.io/v1beta1
  objects:
  - apiVersion: direct.csi.min.io/v1alpha1
    kind: DirectCSIVolume
    metadata:
      creationTimestamp: "2021-03-15T09:01:01Z"
      finalizers:
      - direct.csi.min.io/pv-protection
      - direct.csi.min.io/purge-protection
      generation: 3
      labels:
        direct.csi.min.io/app: minio-example
        direct.csi.min.io/organization: minio
      managedFields:
      - apiVersion: direct.csi.min.io/v1alpha1
        fieldsType: FieldsV1
        fieldsV1:
          f:metadata:
            f:finalizers:
              .: {}
              v:"direct.csi.min.io/purge-protection": {}
              v:"direct.csi.min.io/pv-protection": {}
            f:labels:
              .: {}
              f:direct.csi.min.io/app: {}
              f:direct.csi.min.io/organization: {}
          f:status:
            .: {}
            f:availableCapacity: {}
            f:conditions:
              .: {}
              k:{"type":"Published"}:
                .: {}
                f:lastTransitionTime: {}
                f:message: {}
                f:reason: {}
                f:status: {}
                f:type: {}
              k:{"type":"Staged"}:
                .: {}
                f:lastTransitionTime: {}
                f:message: {}
                f:reason: {}
                f:status: {}
                f:type: {}
            f:containerPath: {}
            f:drive: {}
            f:hostPath: {}
            f:nodeName: {}
            f:stagingPath: {}
            f:totalCapacity: {}
            f:usedCapacity: {}
        manager: direct-csi
        operation: Update
        time: "2021-03-15T09:01:07Z"
      name: pvc-ddedfae0-a545-4801-9d17-f10547531bd9
      resourceVersion: "10418778"
      uid: 41abe032-f9f4-41fe-b691-c7cabae66e22
    status:
      availableCapacity: 2147483648
      conditions:
      - lastTransitionTime: "2021-03-15T09:01:00Z"
        message: ""
        reason: NotInUse
        status: "False"
        type: Staged
      - lastTransitionTime: "2021-03-15T09:01:00Z"
        message: ""
        reason: NotInUse
        status: "True"
        type: Published
      containerPath: /var/lib/kubelet/pods/630551fa-ff43-423d-b752-42d7f000f94e/volumes/kubernetes.io~csi/pvc-ddedfae0-a545-4801-9d17-f10547531bd9/mount
      drive: 27bc586d9cece384bce426b410d05fd498951f8f0b3dfec9b848a67fd3ad6444
      hostPath: /var/lib/direct-csi/mnt/27bc586d9cece384bce426b410d05fd498951f8f0b3dfec9b848a67fd3ad6444/pvc-ddedfae0-a545-4801-9d17-f10547531bd9
      nodeName: minio-k8s8
      stagingPath: /var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-ddedfae0-a545-4801-9d17-f10547531bd9/globalmount
      totalCapacity: 2147483648
      usedCapacity: 0
`
	response := httptest.NewRecorder()
	request, err := http.NewRequest("POST", DriveHandlerPath, strings.NewReader(sampleObj))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Add("Content-Type", "application/yaml")
	ServeVolumeConversion(response, request)
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

	var directCSIVolume directv1beta1.DirectCSIVolume
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(convertedObj.Object, &directCSIVolume); err != nil {
		t.Errorf("Error while converting %v", err)
	}

	if !utils.IsCondition(directCSIVolume.Status.Conditions, string(directv1beta1.DirectCSIVolumeConditionReady), metav1.ConditionTrue, string(directv1beta1.DirectCSIVolumeReasonReady), "") {
		t.Errorf("unexpected status.conditions = %v", directCSIVolume.Status.Conditions)
	}

}
