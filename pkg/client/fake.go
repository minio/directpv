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

package client

import (
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	directcsiclientset "github.com/minio/directpv/pkg/clientset/typed/direct.csi.min.io/v1beta4"

	apiextensionsv1fake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	discoveryfake "k8s.io/client-go/discovery/fake"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
	metadatafake "k8s.io/client-go/metadata/fake"
)

type fakeServerGroupsAndResourcesMethod func() ([]*metav1.APIGroup, []*metav1.APIResourceList, error)

type FakeDiscovery struct {
	discoveryfake.FakeDiscovery
	fakeServerGroupsAndResourcesMethod
	versionInfo *version.Info
}

func (fd *FakeDiscovery) ServerVersion() (*version.Info, error) {
	return fd.versionInfo, nil
}

func (fd *FakeDiscovery) ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	return fd.fakeServerGroupsAndResourcesMethod()
}

// FakeInit initializes fake clients.
func FakeInit() {
	kubeClient = kubernetesfake.NewSimpleClientset()
	directClientset = clientsetfake.NewSimpleClientset()
	directCSIClient = directClientset.DirectV1beta4()
	crdClient = &apiextensionsv1fake.FakeCustomResourceDefinitions{
		Fake: &apiextensionsv1fake.FakeApiextensionsV1{
			Fake: &kubeClient.(*kubernetesfake.Clientset).Fake,
		},
	}
	discoveryClient = &discoveryfake.FakeDiscovery{}
	scheme := runtime.NewScheme()
	_ = metav1.AddMetaToScheme(scheme)
	metadataClient = metadatafake.NewSimpleMetadataClient(scheme)

	latestDirectCSIDriveInterface = directClientset.DirectV1beta4().DirectCSIDrives()
	latestDirectCSIVolumeInterface = directClientset.DirectV1beta4().DirectCSIVolumes()

	initEvent(kubeClient)
}

func SetLatestDirectCSIDriveInterface(driveInterface directcsiclientset.DirectCSIDriveInterface) {
	latestDirectCSIDriveInterface = driveInterface
}

func SetLatestDirectCSIVolumeInterface(volumeInterface directcsiclientset.DirectCSIVolumeInterface) {
	latestDirectCSIVolumeInterface = volumeInterface
}

func SetFakeDiscoveryClient(groupsAndMethodsFn fakeServerGroupsAndResourcesMethod, serverVersionInfo *version.Info) {
	discoveryClient = &FakeDiscovery{
		FakeDiscovery:                      discoveryfake.FakeDiscovery{Fake: &kubeClient.(*kubernetesfake.Clientset).Fake},
		fakeServerGroupsAndResourcesMethod: groupsAndMethodsFn,
		versionInfo:                        serverVersionInfo,
	}
}
