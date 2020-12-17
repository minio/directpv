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
	"path/filepath"

	directv1alpha1 "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1alpha1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/golang/glog"
	"github.com/spf13/viper"
	"k8s.io/utils/mount"
)

var directCSIClient directv1alpha1.DirectV1alpha1Interface
var kubeClient kubernetes.Interface
var crdClient apiextensions.CustomResourceDefinitionInterface
var discoveryClient discovery.DiscoveryInterface
var metadataClient metadata.Interface

func Init() {
	kubeConfig := viper.GetString("kubeconfig")
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			glog.Fatalf("could not find client configuration: %v", err)
		}
		glog.Infof("obtained client config successfully")
	}

	kubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("could not initialize kubeclient: %v", err)
	}

	directCSIClient, err = directv1alpha1.NewForConfig(config)
	if err != nil {
		glog.Fatalf("could not initialize direct-csi client: %v", err)
	}

	crdClientset, err := apiextensions.NewForConfig(config)
	if err != nil {
		glog.Fatalf("could not initialize crd client: %v", err)
	}
	crdClient = crdClientset.CustomResourceDefinitions()

	discoveryClient, err = discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		glog.Fatalf("could not initialize discovery client: %v", err)
	}

	metadataClient, err = metadata.NewForConfig(config)
	if err != nil {
		glog.Fatalf("could not initialize metadata client: %v", err)
	}
}

func GetKubeClient() kubernetes.Interface {
	return kubeClient
}

func GetDirectCSIClient() directv1alpha1.DirectV1alpha1Interface {
	return directCSIClient
}

func GetCRDClient() apiextensions.CustomResourceDefinitionInterface {
	return crdClient
}

func GetDiscoveryClient() discovery.DiscoveryInterface {
	return discoveryClient
}

func GetMetadataClient() metadata.Interface {
	return metadataClient
}

func AddFinalizer(objectMeta *metav1.ObjectMeta, finalizer string) []string {
	finalizers := objectMeta.GetFinalizers()
	for _, f := range finalizers {
		if f == finalizer {
			return finalizers
		}
	}
	finalizers = append(finalizers, finalizer)
	return finalizers
}

func RemoveFinalizer(objectMeta *metav1.ObjectMeta, finalizer string) []string {
	removeByIndex := func(s []string, index int) []string {
		return append(s[:index], s[index+1:]...)
	}
	finalizers := objectMeta.GetFinalizers()
	for index, f := range finalizers {
		if f == finalizer {
			finalizers = removeByIndex(finalizers, index)
			break
		}
	}
	return finalizers
}

func UpdateVolumeStatusCondition(statusConditions []metav1.Condition, condType string, condStatus metav1.ConditionStatus) {
	for i := range statusConditions {
		if statusConditions[i].Type == condType {
			statusConditions[i].Status = condStatus
			statusConditions[i].LastTransitionTime = metav1.Now()
			break
		}
	}
	return
}

func CheckVolumeStatusCondition(statusConditions []metav1.Condition, condType string, condStatus metav1.ConditionStatus) bool {
	for i := range statusConditions {
		if statusConditions[i].Type == condType && statusConditions[i].Status == condStatus {
			return true
		}
	}
	return false

}

// UnmountIfMounted - Idempotent function to unmount a target
func UnmountIfMounted(mountPoint string) error {
	shouldUmount := false
	mountPoints, mntErr := mount.New("").List()
	if mntErr != nil {
		return mntErr
	}
	for _, mp := range mountPoints {
		abPath, _ := filepath.Abs(mp.Path)
		if mountPoint == abPath {
			shouldUmount = true
			break
		}
	}
	if shouldUmount {
		if mErr := mount.New("").Unmount(mountPoint); mErr != nil {
			return mErr
		}
	}
	return nil
}
