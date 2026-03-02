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

package k8s

import (
	apiextensionsv1fake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	discoveryfake "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
)

type fakeServerGroupsAndResourcesMethod func() ([]*metav1.APIGroup, []*metav1.APIResourceList, error)

// FakeDiscovery creates fake discovery.
type FakeDiscovery struct {
	discoveryfake.FakeDiscovery
	fakeServerGroupsAndResourcesMethod
	versionInfo *version.Info
}

// ServerVersion returns version info
func (fd *FakeDiscovery) ServerVersion() (*version.Info, error) {
	return fd.versionInfo, nil
}

// ServerGroupsAndResources returns APIGroups and APIResourceLists
func (fd *FakeDiscovery) ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	return fd.fakeServerGroupsAndResourcesMethod()
}

// FakeInit initializes fake clients.
func FakeInit() {
	var kubeClient kubernetes.Interface = kubernetesfake.NewClientset()
	fakeApiextensionsV1 := apiextensionsv1fake.FakeApiextensionsV1{
		Fake: &kubeClient.(*kubernetesfake.Clientset).Fake,
	}
	crdClient := fakeApiextensionsV1.CustomResourceDefinitions()
	discoveryClient := &discoveryfake.FakeDiscovery{}
	scheme := runtime.NewScheme()
	_ = metav1.AddMetaToScheme(scheme)
	defaultClient = &Client{
		KubeClient:      kubeClient,
		CRDClient:       crdClient,
		DiscoveryClient: discoveryClient,
	}
}

// SetKubeInterface sets the given kube interface
// Note: To be used for writing test cases only
func SetKubeInterface(i kubernetes.Interface) {
	defaultClient.KubeClient = i
}

// NewFakeDiscovery creates a fake discovery interface
// Note: To be used for writing test cases only
func NewFakeDiscovery(groupsAndMethodsFn fakeServerGroupsAndResourcesMethod, serverVersionInfo *version.Info) *FakeDiscovery {
	return &FakeDiscovery{
		FakeDiscovery:                      discoveryfake.FakeDiscovery{Fake: &defaultClient.KubeClient.(*kubernetesfake.Clientset).Fake},
		fakeServerGroupsAndResourcesMethod: groupsAndMethodsFn,
		versionInfo:                        serverVersionInfo,
	}
}
