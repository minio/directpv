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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	directv1beta1 "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta1"
	directv1beta2 "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta2"
	directv1beta3 "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	directv1beta4 "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	json1 "k8s.io/apimachinery/pkg/util/json"

	"github.com/minio/directpv/pkg/utils"
)

func getLabelValue(obj metav1.Object, key string) string {
	if labels := obj.GetLabels(); labels != nil {
		return labels[key]
	}
	return ""
}

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
	serveDriveConversion(response, request)
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

func TestV1alpha1ToV1beta2VolumeUpgrade(t *testing.T) {
	sampleObj := `kind: ConversionReview
apiVersion: apiextensions.k8s.io/v1
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: direct.csi.min.io/v1beta2
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
	request, err := http.NewRequest("POST", VolumeHandlerPath, strings.NewReader(sampleObj))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Add("Content-Type", "application/yaml")
	serveVolumeConversion(response, request)
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
	if e, a := versionV1Beta2, convertedObj.GetAPIVersion(); e != a {
		t.Errorf("expected= %v, actual= %v", e, a)
	}

	var directCSIVolume directv1beta2.DirectCSIVolume
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(convertedObj.Object, &directCSIVolume); err != nil {
		t.Errorf("Error while converting %v", err)
	}

	if nodeLabelV := getLabelValue(&directCSIVolume, string(utils.NodeLabelKey)); nodeLabelV != directCSIVolume.Status.NodeName {
		t.Errorf("expected node label value = %s, got = %s", directCSIVolume.Status.NodeName, nodeLabelV)
	}

	if createdByLabelV := getLabelValue(&directCSIVolume, string(utils.CreatedByLabelKey)); createdByLabelV != "directcsi-controller" {
		t.Errorf("expected created-by label value = directcsi-controller, got = %s", createdByLabelV)
	}

	if !utils.IsCondition(directCSIVolume.Status.Conditions, string(directv1beta2.DirectCSIVolumeConditionReady), metav1.ConditionTrue, string(directv1beta2.DirectCSIVolumeReasonReady), "") {
		t.Errorf("unexpected status.conditions = %v", directCSIVolume.Status.Conditions)
	}

}

func TestV1alpha1ToV1beta3VolumeUpgrade(t *testing.T) {
	sampleObj := `kind: ConversionReview
apiVersion: apiextensions.k8s.io/v1
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: direct.csi.min.io/v1beta3
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
	request, err := http.NewRequest("POST", VolumeHandlerPath, strings.NewReader(sampleObj))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Add("Content-Type", "application/yaml")
	serveVolumeConversion(response, request)
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
	if e, a := versionV1Beta3, convertedObj.GetAPIVersion(); e != a {
		t.Errorf("expected= %v, actual= %v", e, a)
	}

	var directCSIVolume directv1beta3.DirectCSIVolume
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(convertedObj.Object, &directCSIVolume); err != nil {
		t.Errorf("Error while converting %v", err)
	}

	if nodeLabelV := getLabelValue(&directCSIVolume, string(utils.NodeLabelKey)); nodeLabelV != directCSIVolume.Status.NodeName {
		t.Errorf("expected node label value = %s, got = %s", directCSIVolume.Status.NodeName, nodeLabelV)
	}

	if createdByLabelV := getLabelValue(&directCSIVolume, string(utils.CreatedByLabelKey)); createdByLabelV != "directcsi-controller" {
		t.Errorf("expected created-by label value = directcsi-controller, got = %s", createdByLabelV)
	}

	if !utils.IsCondition(directCSIVolume.Status.Conditions, string(directv1beta3.DirectCSIVolumeConditionReady), metav1.ConditionTrue, string(directv1beta3.DirectCSIVolumeReasonReady), "") {
		t.Errorf("unexpected status.conditions = %v", directCSIVolume.Status.Conditions)
	}

}

func TestV1beta1ToV1beta2DriveUpgrade(t *testing.T) {
	sampleObj := `kind: ConversionReview
apiVersion: apiextensions.k8s.io/v1
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: direct.csi.min.io/v1beta2
  objects:
  - apiVersion: direct.csi.min.io/v1beta1
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
      accessTier: Unknown
      conditions:
      - lastTransitionTime: "2021-05-18T06:45:57Z"
        message: ""
        reason: NotAdded
        status: "False"
        type: Owned
      - lastTransitionTime: "2021-05-18T06:45:57Z"
        message: /var/lib/direct-csi/mnt/26b5e22d368f01dfbfa161c35b70d758f960a820d1b79f54950f4309f73be064
        reason: NotAdded
        status: "True"
        type: Mounted
      - lastTransitionTime: "2021-05-18T06:45:57Z"
        message: xfs
        reason: NotAdded
        status: "True"
        type: Formatted
      - lastTransitionTime: "2021-05-18T06:45:57Z"
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
	serveDriveConversion(response, request)
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
	if e, a := versionV1Beta2, convertedObj.GetAPIVersion(); e != a {
		t.Errorf("expected= %v, actual= %v", e, a)
	}

	var directCSIDrive directv1beta2.DirectCSIDrive
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(convertedObj.Object, &directCSIDrive); err != nil {
		t.Errorf("Error while converting %v", err)
	}

	if directCSIDrive.Status.MajorNumber != uint32(0) {
		t.Errorf("expected status.majorNumber = %v, actual status.majorNumber = %v", uint32(0), directCSIDrive.Status.MajorNumber)
	}

	if directCSIDrive.Status.MinorNumber != uint32(0) {
		t.Errorf("expected status.minorNumber = %v, actual status.minorNumber = %v", uint32(0), directCSIDrive.Status.MinorNumber)
	}

	if directCSIDrive.Status.FilesystemUUID != "" {
		t.Errorf("expected status.filesystemUUID = \"\", actual status.filesystemUUID = %v", directCSIDrive.Status.FilesystemUUID)
	}

	if directCSIDrive.Status.PartitionUUID != "" {
		t.Errorf("expected status.partitionUUID = \"\", actual status.partitionUUID = %v", directCSIDrive.Status.PartitionUUID)
	}

	if versionLabelV := getLabelValue(&directCSIDrive, string(utils.VersionLabelKey)); versionLabelV != "v1beta1" {
		t.Errorf("expected version label value = v1beta1, got = %s", versionLabelV)
	}

	if nodeLabelV := getLabelValue(&directCSIDrive, string(utils.NodeLabelKey)); nodeLabelV != directCSIDrive.Status.NodeName {
		t.Errorf("expected node label value = %s, got = %s", directCSIDrive.Status.NodeName, nodeLabelV)
	}

	if createdByLabelV := getLabelValue(&directCSIDrive, string(utils.CreatedByLabelKey)); createdByLabelV != "directcsi-driver" {
		t.Errorf("expected created-by label value = directcsi-driver, got = %s", createdByLabelV)
	}

	if accessTierLabelV := getLabelValue(&directCSIDrive, string(utils.AccessTierLabelKey)); accessTierLabelV != string(directCSIDrive.Status.AccessTier) {
		t.Errorf("expected access-tier label value = %s, got = %s", string(directCSIDrive.Status.AccessTier), accessTierLabelV)
	}
}

func TestV1beta2ToV1beta3DriveUpgrade(t *testing.T) {
	sampleObj := `kind: ConversionReview
apiVersion: apiextensions.k8s.io/v1
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: direct.csi.min.io/v1beta3
  objects:
  - apiVersion: direct.csi.min.io/v1beta2
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
      accessTier: Unknown
      conditions:
      - lastTransitionTime: "2021-05-18T06:45:57Z"
        message: ""
        reason: NotAdded
        status: "False"
        type: Owned
      - lastTransitionTime: "2021-05-18T06:45:57Z"
        message: /var/lib/direct-csi/mnt/26b5e22d368f01dfbfa161c35b70d758f960a820d1b79f54950f4309f73be064
        reason: NotAdded
        status: "True"
        type: Mounted
      - lastTransitionTime: "2021-05-18T06:45:57Z"
        message: xfs
        reason: NotAdded
        status: "True"
        type: Formatted
      - lastTransitionTime: "2021-05-18T06:45:57Z"
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
	serveDriveConversion(response, request)
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
	if e, a := versionV1Beta3, convertedObj.GetAPIVersion(); e != a {
		t.Errorf("expected= %v, actual= %v", e, a)
	}

	var directCSIDrive directv1beta3.DirectCSIDrive
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(convertedObj.Object, &directCSIDrive); err != nil {
		t.Errorf("Error while converting %v", err)
	}

	if directCSIDrive.Status.MajorNumber != uint32(0) {
		t.Errorf("expected status.majorNumber = %v, actual status.majorNumber = %v", uint32(0), directCSIDrive.Status.MajorNumber)
	}

	if directCSIDrive.Status.MinorNumber != uint32(0) {
		t.Errorf("expected status.minorNumber = %v, actual status.minorNumber = %v", uint32(0), directCSIDrive.Status.MinorNumber)
	}

	if directCSIDrive.Status.FilesystemUUID != "" {
		t.Errorf("expected status.filesystemUUID = \"\", actual status.filesystemUUID = %v", directCSIDrive.Status.FilesystemUUID)
	}

	if directCSIDrive.Status.PartitionUUID != "" {
		t.Errorf("expected status.partitionUUID = \"\", actual status.partitionUUID = %v", directCSIDrive.Status.PartitionUUID)
	}

	if versionLabelV := getLabelValue(&directCSIDrive, string(utils.VersionLabelKey)); versionLabelV != "v1beta2" {
		t.Errorf("expected version label value = v1beta2, got = %s", versionLabelV)
	}

	if nodeLabelV := getLabelValue(&directCSIDrive, string(utils.NodeLabelKey)); nodeLabelV != directCSIDrive.Status.NodeName {
		t.Errorf("expected node label value = %s, got = %s", directCSIDrive.Status.NodeName, nodeLabelV)
	}

	if createdByLabelV := getLabelValue(&directCSIDrive, string(utils.CreatedByLabelKey)); createdByLabelV != "directcsi-driver" {
		t.Errorf("expected created-by label value = directcsi-driver, got = %s", createdByLabelV)
	}

	if accessTierLabelV := getLabelValue(&directCSIDrive, string(utils.AccessTierLabelKey)); accessTierLabelV != string(directCSIDrive.Status.AccessTier) {
		t.Errorf("expected access-tier label value = %s, got = %s", string(directCSIDrive.Status.AccessTier), accessTierLabelV)
	}
}

func TestMigrate(t *testing.T) {
	testCases := []struct {
		srcObject    runtime.Object
		destObject   runtime.Object
		groupVersion schema.GroupVersion
	}{
		// upgrade drive v1beta1 => v1beta4
		{
			srcObject: &directv1beta1.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta1", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta1.DirectCSIDriveFinalizerDataProtection),
						directv1beta1.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta1.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta1.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta1.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			destObject: &directv1beta4.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta4", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta4.DirectCSIDriveFinalizerDataProtection),
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta4.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta4.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta4"},
		},
		// upgrade drive v1beta1 => v1beta3
		{
			srcObject: &directv1beta1.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta1", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta1.DirectCSIDriveFinalizerDataProtection),
						directv1beta1.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta1.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta1.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta1.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			destObject: &directv1beta3.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta3.DirectCSIDriveFinalizerDataProtection),
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta3.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta3.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta3"},
		},
		// upgrade drive v1beta2 => v1beta4
		{
			srcObject: &directv1beta2.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta2", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta2.DirectCSIDriveFinalizerDataProtection),
						directv1beta2.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta2.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta2.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta2.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			destObject: &directv1beta4.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta4", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta3.DirectCSIDriveFinalizerDataProtection),
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta4.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta4.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta4"},
		},
		// upgrade drive v1beta2 => v1beta3
		{
			srcObject: &directv1beta2.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta2", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta2.DirectCSIDriveFinalizerDataProtection),
						directv1beta2.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta2.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta2.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta2.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			destObject: &directv1beta3.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta3.DirectCSIDriveFinalizerDataProtection),
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta3.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta3.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta3"},
		},

		// downgrade drive v1beta4 => v1beta1
		{
			srcObject: &directv1beta4.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta4", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta4.DirectCSIDriveFinalizerDataProtection),
						directv1beta4.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta4.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta4.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta4.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			destObject: &directv1beta1.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta1", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta1.DirectCSIDriveFinalizerDataProtection),
						directv1beta1.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta1.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta1.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta1.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta1"},
		},
		// downgrade drive v1beta4 => v1beta2
		{
			srcObject: &directv1beta4.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta4", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta4.DirectCSIDriveFinalizerDataProtection),
						directv1beta4.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta4.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta4.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta4.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			destObject: &directv1beta2.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta2", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta2.DirectCSIDriveFinalizerDataProtection),
						directv1beta2.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta2.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta2.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta2.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta2"},
		},
		// downgrade drive v1beta4 => v1beta3
		{
			srcObject: &directv1beta4.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta4", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta4.DirectCSIDriveFinalizerDataProtection),
						directv1beta4.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta4.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta4.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta4.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			destObject: &directv1beta3.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta3.DirectCSIDriveFinalizerDataProtection),
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta3.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta3.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta3"},
		},

		// downgrade drive v1beta3 => v1beta1
		{
			srcObject: &directv1beta3.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta3.DirectCSIDriveFinalizerDataProtection),
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta3.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta3.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			destObject: &directv1beta1.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta1", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta1.DirectCSIDriveFinalizerDataProtection),
						directv1beta1.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta1.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta1.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta1.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta1"},
		},
		// downgrade drive v1beta3 => v1beta2
		{
			srcObject: &directv1beta3.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta3.DirectCSIDriveFinalizerDataProtection),
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta3.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta3.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta3.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			destObject: &directv1beta2.DirectCSIDrive{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta2", Kind: "DirectCSIDrive"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Finalizers: []string{
						string(directv1beta2.DirectCSIDriveFinalizerDataProtection),
						directv1beta2.DirectCSIDriveFinalizerPrefix + "volume-1",
						directv1beta2.DirectCSIDriveFinalizerPrefix + "volume-2",
					},
				},
				Status: directv1beta2.DirectCSIDriveStatus{
					NodeName:          "node-name",
					DriveStatus:       directv1beta2.DriveStatusInUse,
					FreeCapacity:      2048,
					AllocatedCapacity: 1024,
					TotalCapacity:     3072,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta2"},
		},

		// upgrage volume v1beta1 => v1beta4
		{
			srcObject: &directv1beta1.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta1", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta1.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta1.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta4.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta4", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta4.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta4.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta4"},
		},
		// upgrage volume v1beta2 => v1beta4
		{
			srcObject: &directv1beta2.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta2", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta2.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta2.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta4.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta4", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta4.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta4.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta4"},
		},
		// upgrage volume v1beta1 => v1beta3
		{
			srcObject: &directv1beta1.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta1", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta1.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta1.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta3.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta3.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta3.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta3"},
		},
		// upgrage volume v1beta2 => v1beta3
		{
			srcObject: &directv1beta2.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta2", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta2.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta2.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta3.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta3.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta3.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta3"},
		},
		// downgrade volume v1beta4 => v1beta1
		{
			srcObject: &directv1beta4.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta4", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta4.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta4.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta1.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta1", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta1.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta1.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta1"},
		},
		// downgrage volume v1beta4 => v1beta2
		{
			srcObject: &directv1beta4.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta4", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta4.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta4.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta2.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta2", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta2.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta2.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta2"},
		},
		// downgrage volume v1beta4 => v1beta3
		{
			srcObject: &directv1beta4.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta4", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta4.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta4.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta3.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta3.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta3.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta3"},
		},

		// downgrade volume v1beta3 => v1beta1
		{
			srcObject: &directv1beta3.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta3.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta3.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta1.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta1", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta1.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta1.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta1"},
		},
		// downgrage volume v1beta3 => v1beta2
		{
			srcObject: &directv1beta3.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta3", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta3.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta3.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			destObject: &directv1beta2.DirectCSIVolume{
				TypeMeta: metav1.TypeMeta{APIVersion: "direct.csi.min.io/v1beta2", Kind: "DirectCSIVolume"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-volume-1",
					Finalizers: []string{
						string(directv1beta2.DirectCSIVolumeFinalizerPurgeProtection),
					},
				},
				Status: directv1beta2.DirectCSIVolumeStatus{
					NodeName:      "test-node",
					HostPath:      "hostpath",
					Drive:         "test-drive",
					TotalCapacity: 2048,
				},
			},
			groupVersion: schema.GroupVersion{
				Group:   "direct.csi.min.io",
				Version: "v1beta2"},
		},
	}

	for i, test := range testCases {
		objBytes, err := json1.Marshal(test.srcObject)
		if err != nil {
			t.Fatalf("failed to marshaling source object: %v", test.srcObject)
		}
		cr := unstructured.Unstructured{}
		if err := cr.UnmarshalJSON(objBytes); err != nil {
			t.Fatalf("failed to umarshaling source object: %v", test.srcObject)
		}
		result := &unstructured.Unstructured{}

		err = Migrate(&cr, result, test.groupVersion)
		if err != nil {
			t.Fatalf("failed to convert runtime object: %v", err)
		}
		gv := result.GetObjectKind().GroupVersionKind().GroupVersion().String()
		if gv != test.destObject.GetObjectKind().GroupVersionKind().GroupVersion().String() {
			t.Fatalf("Test  %d failed wrong group version: %s, expected: %s", i+1, gv, test.groupVersion.Version)
		}
	}
}
