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
	"fmt"
	"sync/atomic"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	// support gcp, azure, and oidc client auth
	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

// Init initializes various client interfaces.
func Init() error {
	if atomic.AddInt32(&initialized, 1) != 1 {
		return nil
	}

	var err error

	if kubeConfig, err = GetKubeConfig(); err != nil {
		return fmt.Errorf("unable to get kubernetes configuration; %v", err)
	}

	kubeConfig.WarningHandler = rest.NoWarnings{}
	if kubeClient, err = kubernetes.NewForConfig(kubeConfig); err != nil {
		return fmt.Errorf("unable to create new kubernetes client interface; %v", err)
	}

	if apiextensionsClient, err = apiextensions.NewForConfig(kubeConfig); err != nil {
		return fmt.Errorf("unable to create new API extensions client interface; %v", err)
	}
	crdClient = apiextensionsClient.CustomResourceDefinitions()

	if discoveryClient, err = discovery.NewDiscoveryClientForConfig(kubeConfig); err != nil {
		return fmt.Errorf("unable to create new discovery client interface; %v", err)
	}
	return nil
}
