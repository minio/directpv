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
	direct "github.com/minio/direct-csi/pkg/clientset"
	fakedirect "github.com/minio/direct-csi/pkg/clientset/fake"
	directcsi "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1beta2"
	fakedirectcsi "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1beta2/fake"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"

	fakeapiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1/fake"
	fakediscovery "k8s.io/client-go/discovery/fake"
	fakekube "k8s.io/client-go/kubernetes/fake"
	fakemetadata "k8s.io/client-go/metadata/fake"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
)

var fakeDirectClientset direct.Interface
var fakeKubeClient kubernetes.Interface
var fakeCRDClient apiextensions.CustomResourceDefinitionInterface
var fakeAPIExtenstionsClient apiextensions.ApiextensionsV1Interface
var fakeDiscoveryClient discovery.DiscoveryInterface
var fakeMetadataClient metadata.Interface
var fakeDirectCSIClient directcsi.DirectV1beta2Interface

var fakeInitialized bool

func InitFake() {
	if fakeInitialized {
		return
	}
	fakeKubeClient = fakekube.NewSimpleClientset()
	fakeDirectClientset = fakedirect.NewSimpleClientset()
	fakeDirectCSIClient = &fakedirectcsi.FakeDirectV1beta2{}
	fakeCRDClientSet := &fakeapiextensions.FakeApiextensionsV1{}
	fakeAPIExtenstionsClient = fakeCRDClientSet
	fakeCRDClient = fakeCRDClientSet.CustomResourceDefinitions()
	fakeDiscoveryClient = &fakediscovery.FakeDiscovery{}
	scheme := runtime.NewScheme()
	metav1.AddMetaToScheme(scheme)
	fakeMetadataClient = fakemetadata.NewSimpleMetadataClient(scheme)

	fakeInitialized = true
}

func SetFakeDirectCSIClient(fakeClient directcsi.DirectV1beta2Interface) {
	fakeDirectCSIClient = fakeClient
}
