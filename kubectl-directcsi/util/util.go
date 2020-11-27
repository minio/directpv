/*
 * This file is part of MinIO Direct CSI
 * Copyright (C) 2020, MinIO, Inc.
 *
 * This code is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License, version 3,
 * as published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License, version 3,
 * along with this program.  If not, see <http://www.gnu.org/licenses/>
 *
 */

package util

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"

	directv1alpha1 "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	k8sretry "k8s.io/client-go/util/retry"

	"github.com/golang/glog"
	"github.com/minio/kubectl-directcsi/util/randomstring"
	clientset "k8s.io/client-go/kubernetes"
)

func newHostPathVolume(name, path string) corev1.Volume {
	hostPathType := corev1.HostPathDirectoryOrCreate
	volumeSource := corev1.VolumeSource{
		HostPath: &corev1.HostPathVolumeSource{
			Path: path,
			Type: &hostPathType,
		},
	}

	return corev1.Volume{
		Name:         name,
		VolumeSource: volumeSource,
	}
}

func newDirectCSIPluginsSocketDir(kubeletDir, name string) string {
	return filepath.Join(kubeletDir, "plugins", sanitize(name))
}

func newVolumeMount(name, path string, bidirectional bool) corev1.VolumeMount {
	mountProp := corev1.MountPropagationNone
	if bidirectional {
		mountProp = corev1.MountPropagationBidirectional
	}
	return corev1.VolumeMount{
		Name:             name,
		MountPath:        path,
		MountPropagation: &mountProp,
	}
}

func sanitize(s string) string {
	re := regexp.MustCompile("[^a-zA-Z0-9-]")
	s = re.ReplaceAllString(s, "-")
	if s[len(s)-1] == '-' {
		s = s + "X"
	}
	return s
}

func generateSanitizedUniqueNameFrom(name string) string {
	sanitizedName := sanitize(name)
	// Max length of name is 255. If needed, cut out last 6 bytes
	// to make room for randomstring
	if len(sanitizedName) >= 255 {
		sanitizedName = sanitizedName[0:249]
	}

	// Get a 5 byte randomstring
	shortUUID := randomstring.New(5)

	// Concatenate sanitizedName (249) and shortUUID (5) with a '-' in between
	// Max length of the returned name cannot be more than 255 bytes
	return fmt.Sprintf("%s-%s", sanitizedName, shortUUID)
}

func retry(f func() error) error {
	if err := k8sretry.OnError(k8sretry.DefaultBackoff, func(err error) bool {
		if errors.IsAlreadyExists(err) {
			return false
		}
		return true
	}, f); err != nil {
		if !errors.IsAlreadyExists(err) {
			glog.Errorf("Creation failed: %v", err)
			return err
		}
	}
	return nil
}

func getConf() (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	return config, nil
}

// GetKubeClient provides k8s client for k8s resources
func GetKubeClient() kubernetes.Interface {
	conf, err := getConf()
	if err != nil {
		glog.Fatalf("could not initialize kubeclient: %v", err)
	}
	kubeClient, err := kubernetes.NewForConfig(conf)
	if err != nil {
		glog.Fatalf("could not initialize kubeclient: %v", err)
	}
	return kubeClient
}

// GetDirectCSIClient provides k8s client for DirectCSI CR
func GetDirectCSIClient() directv1alpha1.DirectV1alpha1Interface {
	conf, err := getConf()
	if err != nil {
		glog.Fatalf("could not initialize kubeclient: %v", err)
	}
	directCSIClient, err := directv1alpha1.NewForConfig(conf)
	if err != nil {
		glog.Fatalf("could not initialize kubeclient: %v", err)
	}
	return directCSIClient
}

// GetKubeExtensionClient provides k8s client for CRDs
func GetKubeExtensionClient() *apiextension.Clientset {
	conf, err := getConf()
	if err != nil {
		glog.Fatalf("could not initialize kubeclient: %v", err)
	}
	extClient, err := apiextension.NewForConfig(conf)
	if err != nil {
		glog.Fatalf("could not initialize kubeclient: %v", err)
	}
	return extClient
}

func CreateCRD(ctx context.Context, client *apiextension.Clientset, crd *apiextensionv1.CustomResourceDefinition) error {
	_, err := client.ApiextensionsV1beta1().CustomResourceDefinitions().Create(context.Background(), crd, v1.CreateOptions{})
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return fmt.Errorf("CustomResourceDefinition %s: already present, skipped", crd.ObjectMeta.Name)
		}
		return err
	}
	fmt.Printf("CustomResourceDefinition %s: created\n", crd.ObjectMeta.Name)
	return nil
}

func CreateCSIService(ctx context.Context, kClient clientset.Interface, name, identity string) error {
	csiPort := corev1.ServicePort{
		Port: 12345,
		Name: "unused",
	}
	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				createByLabel: directCSIController,
			},
			Name:      name,
			Namespace: identity,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{csiPort},
			Selector: map[string]string{
				createByLabel: directCSIController,
			},
		},
	}

	return retry(func() error {
		_, err := kClient.CoreV1().Services(identity).Create(ctx, svc, metav1.CreateOptions{})
		return err
	})
}

func CreateCSIDriver(ctx context.Context, kClient clientset.Interface, name string) error {
	podInfoOnMount := false
	attachRequired := false
	csiDriver := &storagev1beta1.CSIDriver{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CSIDriver",
			APIVersion: "storage.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				createByLabel: directCSIController,
			},
		},
		Spec: storagev1beta1.CSIDriverSpec{
			PodInfoOnMount: &podInfoOnMount,
			AttachRequired: &attachRequired,
			VolumeLifecycleModes: []storagev1beta1.VolumeLifecycleMode{
				storagev1beta1.VolumeLifecyclePersistent,
				storagev1beta1.VolumeLifecycleEphemeral,
			},
		},
	}

	// Create CSIDriver Obj
	return retry(func() error {
		_, err := kClient.StorageV1beta1().CSIDrivers().Create(ctx, csiDriver, metav1.CreateOptions{})
		return err
	})
}

func CreateDirectCSINamespace(ctx context.Context, kClient clientset.Interface, identity string) error {
	ns := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: identity,
			Labels: map[string]string{
				createByLabel: directCSIController,
			},
		},
		Spec: corev1.NamespaceSpec{
			Finalizers: []corev1.FinalizerName{},
		},
		Status: corev1.NamespaceStatus{},
	}

	// Create Namespace Obj
	return retry(func() error {
		_, err := kClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		return err
	})
}

func CreateStorageClass(ctx context.Context, kClient clientset.Interface, name string) error {
	allowExpansion := false
	bindingMode := storagev1beta1.VolumeBindingImmediate
	allowedTopologies := []corev1.TopologySelectorTerm{}
	retainPolicy := corev1.PersistentVolumeReclaimRetain

	// Create StorageClass for the new driver
	storageClass := &storagev1beta1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: "storage.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				createByLabel: directCSIController,
			},
		},
		Provisioner:          name,
		AllowVolumeExpansion: &allowExpansion,
		VolumeBindingMode:    &bindingMode,
		AllowedTopologies:    allowedTopologies,
		ReclaimPolicy:        &retainPolicy,
	}

	return retry(func() error {
		_, err := kClient.StorageV1beta1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{})
		return err
	})
}

func CreateDeployment(ctx context.Context, kClient clientset.Interface, name, identity, kubeletDirPath, csiRootPath string) error {
	generatedSelectorValue := generateSanitizedUniqueNameFrom(name)
	var replicas int32 = 3
	privileged := true
	podSpec := corev1.PodSpec{
		ServiceAccountName: name,
		Volumes: []corev1.Volume{
			newHostPathVolume(volumeNameSocketDir, newDirectCSIPluginsSocketDir(kubeletDirPath, name)),
		},
		Containers: []corev1.Container{
			{
				Name:  csiProvisionerContainerName,
				Image: csiProvisionerContainerImage,
				Args: []string{
					"--v=5",
					"--timeout=300s",
					fmt.Sprintf("--csi-address=$(%s)", endpointEnvVarCSI),
					"--enable-leader-election",
					"--leader-election-type=leases",
					"--feature-gates=Topology=true",
					"--strict-topology",
				},
				Env: []corev1.EnvVar{
					{
						Name:  endpointEnvVarCSI,
						Value: "unix:///csi/csi.sock",
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(volumeNameSocketDir, "/csi", false),
				},
				TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
				TerminationMessagePath:   "/var/log/controller-provisioner-termination-log",
				// TODO: Enable this after verification
				// LivenessProbe: &corev1.Probe{
				// 	FailureThreshold:    5,
				// 	InitialDelaySeconds: 10,
				// 	TimeoutSeconds:      3,
				// 	PeriodSeconds:       2,
				// 	Handler: corev1.Handler{
				// 		HTTPGet: &corev1.HTTPGetAction{
				// 			Path: healthZContainerPortPath,
				// 			Port: intstr.FromInt(9898),
				// 		},
				// 	},
				// },
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
			},
			{
				Name:  directCSIContainerName,
				Image: directCSIContainerImage,
				Args: []string{
					"--v=5",
					fmt.Sprintf("--identity=%s", name),
					fmt.Sprintf("--endpoint=$(%s)", endpointEnvVarCSI),
					"--controller",
				},
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Ports: []corev1.ContainerPort{
					{
						ContainerPort: 9898,
						Name:          "healthz",
						Protocol:      corev1.ProtocolTCP,
					},
				},
				Env: []corev1.EnvVar{
					{
						Name: kubeNodeNameEnvVar,
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "v1",
								FieldPath:  "spec.nodeName",
							},
						},
					},
					{
						Name:  endpointEnvVarCSI,
						Value: "unix:///csi/csi.sock",
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(volumeNameSocketDir, "/csi", false),
				},
			},
		},
	}

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: identity,
			Labels: map[string]string{
				createByLabel: directCSIController,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: metav1.AddLabelToSelector(&metav1.LabelSelector{}, directCSISelector, generatedSelectorValue),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					Labels: map[string]string{
						directCSISelector: generatedSelectorValue,
					},
				},
				Spec: podSpec,
			},
		},
		Status: appsv1.DeploymentStatus{},
	}
	return retry(func() error {
		_, err := kClient.AppsV1().Deployments(identity).Create(ctx, deployment, metav1.CreateOptions{})
		return err
	})
}

func CreateDaemonSet(ctx context.Context, kClient clientset.Interface, name, identity, kubeletDirPath, csiRootPath string) error {
	generatedSelectorValue := generateSanitizedUniqueNameFrom(name)

	privileged := true
	podSpec := corev1.PodSpec{
		ServiceAccountName: name,
		HostNetwork:        true,
		HostIPC:            true,
		HostPID:            true,
		Volumes: []corev1.Volume{
			newHostPathVolume(volumeNameSocketDir, newDirectCSIPluginsSocketDir(kubeletDirPath, name)),
			newHostPathVolume(volumeNameMountpointDir, kubeletDirPath+"/pods"),
			newHostPathVolume(volumeNameRegistrationDir, kubeletDirPath+"/plugins_registry"),
			newHostPathVolume(volumeNamePluginDir, kubeletDirPath+"/plugins"),
			newHostPathVolume(volumeNameCSIRootDir, csiRootPath),
			newHostPathVolume(volumeNameDevDir, volumePathDevDir),
			newHostPathVolume(volumeNameSysDir, volumePathSysDir),
			newHostPathVolume(volumeNameProcDir, volumePathProcDir),
		},
		Containers: []corev1.Container{
			{
				Name:  nodeDriverRegistrarContainerName,
				Image: nodeDriverRegistrarContainerImage,
				Args: []string{
					"--v=5",
					"--csi-address=/csi/csi.sock",
					fmt.Sprintf("--kubelet-registration-path=%s", newDirectCSIPluginsSocketDir(kubeletDirPath, name)+"/csi.sock"),
				},
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Env: []corev1.EnvVar{
					{
						Name: kubeNodeNameEnvVar,
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "v1",
								FieldPath:  "spec.nodeName",
							},
						},
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(volumeNameSocketDir, "/csi", false),
					newVolumeMount(volumeNameRegistrationDir, "/registration", false),
				},
				TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
				TerminationMessagePath:   "/var/log/driver-registrar-termination-log",
			},
			{
				Name:  directCSIContainerName,
				Image: directCSIContainerImage,
				Args: []string{
					fmt.Sprintf("--identity=%s", name),
					"--v=5",
					fmt.Sprintf("--endpoint=$(%s)", endpointEnvVarCSI),
					fmt.Sprintf("--node-id=$(%s)", kubeNodeNameEnvVar),
					"--procfs=/hostproc",
					"--driver",
				},
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Env: []corev1.EnvVar{
					{
						Name: kubeNodeNameEnvVar,
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "v1",
								FieldPath:  "spec.nodeName",
							},
						},
					},
					{
						Name:  endpointEnvVarCSI,
						Value: "unix:///csi/csi.sock",
					},
				},
				TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
				TerminationMessagePath:   "/var/log/driver-termination-log",
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(volumeNameSocketDir, "/csi", false),
					newVolumeMount(volumeNameMountpointDir, kubeletDirPath+"/pods", false),
					newVolumeMount(volumeNamePluginDir, kubeletDirPath+"/plugins", true),
					newVolumeMount(volumeNameCSIRootDir, csiRootPath, true),
					newVolumeMount(volumeNameDevDir, "/dev", true),
					newVolumeMount(volumeNameSysDir, "/sys", true),
					newVolumeMount(volumeNameProcDir, "/hostproc", true),
				},
				Ports: []corev1.ContainerPort{
					{
						ContainerPort: 9898,
						Name:          "healthz",
						Protocol:      corev1.ProtocolTCP,
					},
				},
				LivenessProbe: &corev1.Probe{
					FailureThreshold:    5,
					InitialDelaySeconds: 10,
					TimeoutSeconds:      3,
					PeriodSeconds:       2,
					Handler: corev1.Handler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: healthZContainerPortPath,
							Port: intstr.FromString(healthZContainerPortName),
						},
					},
				},
			},
			{
				Name:  livenessProbeContainerName,
				Image: livenessProbeContainerImage,
				Args: []string{
					"--csi-address=/csi/csi.sock",
					"--health-port=9898",
				},
				TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
				TerminationMessagePath:   "/var/log/driver-liveness-termination-log",
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(volumeNameSocketDir, "/csi", false),
				},
			},
		},
	}

	daemonset := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: identity,
			Labels: map[string]string{
				createByLabel: directCSIController,
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: metav1.AddLabelToSelector(&metav1.LabelSelector{}, directCSISelector, generatedSelectorValue),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					Labels: map[string]string{
						directCSISelector: generatedSelectorValue,
					},
				},
				Spec: podSpec,
			},
		},
		Status: appsv1.DaemonSetStatus{},
	}
	return retry(func() error {
		_, err := kClient.AppsV1().DaemonSets(identity).Create(ctx, daemonset, metav1.CreateOptions{})
		return err
	})
}

// Delete Funcs

func RemoveDirectCSINamespace(ctx context.Context, kClient clientset.Interface, identity string) error {
	// Delete Namespace Obj
	return retry(func() error {
		return kClient.CoreV1().Namespaces().Delete(ctx, identity, metav1.DeleteOptions{})
	})
}

func RemoveCSIService(ctx context.Context, kClient clientset.Interface, name, identity string) error {
	return retry(func() error {
		return kClient.CoreV1().Services(identity).Delete(ctx, name, metav1.DeleteOptions{})
	})
}

func RemoveCSIDriver(ctx context.Context, kClient clientset.Interface, name string) error {
	// Delete CSIDriver Obj
	return retry(func() error {
		return kClient.StorageV1beta1().CSIDrivers().Delete(ctx, name, metav1.DeleteOptions{})
	})
}

func RemoveStorageClass(ctx context.Context, kClient clientset.Interface, name string) error {
	// Delete StorageClass Obj
	return retry(func() error {
		return kClient.StorageV1beta1().StorageClasses().Delete(ctx, name, metav1.DeleteOptions{})
	})
}

func RemoveDeployment(ctx context.Context, kClient clientset.Interface, name, identity string) error {
	return retry(func() error {
		return kClient.AppsV1().Deployments(identity).Delete(ctx, name, metav1.DeleteOptions{})
	})
}

func RemoveDaemonSet(ctx context.Context, kClient clientset.Interface, name, identity string) error {
	return retry(func() error {
		return kClient.AppsV1().DaemonSets(identity).Delete(ctx, name, metav1.DeleteOptions{})
	})
}
