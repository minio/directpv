// This file is part of MinIO Direct CSI
// Copyright (c) 2020 MinIO, Inc.
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
	directv1alpha1 "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/golang/glog"
	"github.com/spf13/viper"
)

var directCSIClient directv1alpha1.DirectV1alpha1Interface
var kubeClient kubernetes.Interface

func init() {
	kubeConfig := viper.GetString("kubeconfig")
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		glog.Fatalf("could not find client configuration: %v", err)
	}
	kubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("could not initialize kubeclient: %v", err)
	}
	directCSIClient, err = directv1alpha1.NewForConfig(config)
	if err != nil {
		glog.Fatalf("could not initialize direct-csi client: %v", err)
	}
}

func GetKubeClient() kubernetes.Interface {
	return kubeClient
}

func GetDirectCSIClient() directv1alpha1.DirectV1alpha1Interface {
	return directCSIClient
}
