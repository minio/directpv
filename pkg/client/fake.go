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

package client

import (
	clientsetfake "github.com/minio/direct-csi/pkg/clientset/fake"
	directcsifake "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1beta3/fake"

	apiextensionsv1fake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	discoveryfake "k8s.io/client-go/discovery/fake"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
	metadatafake "k8s.io/client-go/metadata/fake"
)

// FakeInit initializes fake clients.
func FakeInit() {
	kubeClient = kubernetesfake.NewSimpleClientset()
	directClientset = clientsetfake.NewSimpleClientset()
	directCSIClient = directClientset.DirectV1beta3()
	apiextensionsClient = &apiextensionsv1fake.FakeApiextensionsV1{}
	crdClient = apiextensionsClient.CustomResourceDefinitions()
	discoveryClient = &discoveryfake.FakeDiscovery{}

	scheme := runtime.NewScheme()
	_ = metav1.AddMetaToScheme(scheme)
	metadataClient = metadatafake.NewSimpleMetadataClient(scheme)

	initEvent(kubeClient)
}

// SetDirectCSIClient sets fake direct-csi client.
func SetDirectCSIClient(fakeClient *directcsifake.FakeDirectV1beta3) {
	directCSIClient = fakeClient
}
