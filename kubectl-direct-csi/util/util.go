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
	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	k8sretry "k8s.io/client-go/util/retry"

	"github.com/golang/glog"
	"github.com/minio/kubectl-direct-csi/util/randomstring"
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
		name,
		volumeSource,
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
			glog.Errorf("CSIDriver creation failed: %v", err)
			return err
		}
	}
	return nil
}

func getConf() (*rest.Config, error) {
	kubeConfig := viper.GetString("kubeconfig")
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			glog.Fatalf("could not find client configuration: %v", err)
		}
		glog.Infof("obtained client config successfully")
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

// GetCRDClient provides k8s client for CRDs
func GetCRDClient() apiextensions.CustomResourceDefinitionInterface {
	conf, err := getConf()
	if err != nil {
		glog.Fatalf("could not initialize kubeclient: %v", err)
	}
	crdClient, err := apiextensions.NewForConfig(conf)
	if err != nil {
		glog.Fatalf("could not initialize kubeclient: %v", err)
	}
	return crdClient.CustomResourceDefinitions()
}

func CreateCSIService(ctx context.Context, kClient clientset.Interface, name string) error {
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
			Namespace: name,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{csiPort},
			Selector: map[string]string{
				createByLabel: directCSIController,
			},
		},
	}

	return retry(func() error {
		_, err := kClient.CoreV1().Services(name).Create(ctx, svc, metav1.CreateOptions{})
		return err
	})
}

func CreateCSIDriver(ctx context.Context, kClient clientset.Interface, name string) error {
	podInfoOnMount := false
	attachRequired := false
	csiDriver := &storagev1.CSIDriver{
		metav1.TypeMeta{
			Kind:       "CSIDriver",
			APIVersion: "storage.k8s.io/v1",
		},
		metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				createByLabel: directCSIController,
			},
		},
		storagev1.CSIDriverSpec{
			PodInfoOnMount: &podInfoOnMount,
			AttachRequired: &attachRequired,
			VolumeLifecycleModes: []storagev1.VolumeLifecycleMode{
				storagev1.VolumeLifecyclePersistent,
				storagev1.VolumeLifecycleEphemeral,
			},
		},
	}

	// Create CSIDriver Obj
	return retry(func() error {
		_, err := kClient.StorageV1().CSIDrivers().Create(ctx, csiDriver, metav1.CreateOptions{})
		return err
	})
}

func CreateDirectCSINamespace(ctx context.Context, kClient clientset.Interface, name string) error {
	ns := &corev1.Namespace{
		metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				createByLabel: directCSIController,
			},
		},
		corev1.NamespaceSpec{
			Finalizers: []corev1.FinalizerName{},
		},
		corev1.NamespaceStatus{},
	}

	// Create Namespace Obj
	return retry(func() error {
		_, err := kClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		return err
	})
}

func CreateStorageClass(ctx context.Context, kClient clientset.Interface, name string) error {
	allowExpansion := false
	bindingMode := storagev1.VolumeBindingImmediate
	allowedTopologies := []corev1.TopologySelectorTerm{}
	retainPolicy := corev1.PersistentVolumeReclaimRetain

	// Create StorageClass for the new driver
	storageClass := &storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: "storage.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				createByLabel: directCSIController,
			},
		},
		AllowVolumeExpansion: &allowExpansion,
		VolumeBindingMode:    &bindingMode,
		AllowedTopologies:    allowedTopologies,
		ReclaimPolicy:        &retainPolicy,
	}

	return retry(func() error {
		_, err := kClient.StorageV1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{})
		return err
	})
}

func CreateDeployment(ctx context.Context, kClient clientset.Interface, name, identity, kubeletDirPath, csiRootPath string) error {
	generatedSelectorValue := generateSanitizedUniqueNameFrom(name)
	var replicas int32 = 3
	privileged := false
	podSpec := corev1.PodSpec{
		ServiceAccountName: name,
		Volumes: []corev1.Volume{
			newHostPathVolume(volumeNameSocketDir, newDirectCSIPluginsSocketDir(kubeletDirPath, name)),
		},
		Containers: []corev1.Container{
			{
				Name:  CSIProvisionerContainerName,
				Image: CSIProvisionerContainerImage,
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
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Ports: []corev1.ContainerPort{
					{
						Name:          healthZContainerPortName,
						ContainerPort: healthZContainerPort,
						Protocol:      healthZContainerPortProtocol,
					},
				},
			},
			{
				Name:  DirectCSIContainerName,
				Image: DirectCSIContainerImage,
				Args: []string{
					fmt.Sprintf("--identity=%s", identity),
					"--v=5",
					"--csi-address=/csi/csi.sock",
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
		metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		metav1.ObjectMeta{
			Name:      name,
			Namespace: name,
			Labels: map[string]string{
				createByLabel: directCSIController,
			},
		},
		appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: metav1.AddLabelToSelector(&metav1.LabelSelector{}, directCSISelector, generatedSelectorValue),
			Template: corev1.PodTemplateSpec{
				metav1.ObjectMeta{
					Name: name,
					Labels: map[string]string{
						directCSISelector: generatedSelectorValue,
					},
				},
				podSpec,
			},
		},
		appsv1.DeploymentStatus{},
	}
	return retry(func() error {
		_, err := kClient.AppsV1().Deployments(name).Create(ctx, deployment, metav1.CreateOptions{})
		return err
	})
}

func CreateDaemonSet(ctx context.Context, kClient clientset.Interface, name, identity, kubeletDirPath, csiRootPath string) error {
	generatedSelectorValue := generateSanitizedUniqueNameFrom(name)

	privileged := false
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
					fmt.Sprintf("--kubelet-registration-path=$(%s)", kubeletDirPath+"/plugins/direct-csi-min-io/csi.sock"),
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
				Name:  DirectCSIContainerName,
				Image: DirectCSIContainerImage,
				Args: []string{
					fmt.Sprintf("--identity=%s", identity),
					"--v=5",
					"--csi-address=/csi/csi.sock",
					fmt.Sprintf("--endpoint=$(%s)", endpointEnvVarCSI),
					fmt.Sprintf("--node-id=$(%s)", kubeNodeNameEnvVar),
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
					newVolumeMount(volumeNameCSIRootDir, csiRootPath, false),
					newVolumeMount(volumeNameDevDir, "/dev", false),
					newVolumeMount(volumeNameSysDir, "/sys", false),
					newVolumeMount(volumeNameProcDir, "/proc", false),
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
		metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		metav1.ObjectMeta{
			Name:      name,
			Namespace: name,
			Labels: map[string]string{
				createByLabel: directCSIController,
			},
		},
		appsv1.DaemonSetSpec{
			Selector: metav1.AddLabelToSelector(&metav1.LabelSelector{}, directCSISelector, generatedSelectorValue),
			Template: corev1.PodTemplateSpec{
				metav1.ObjectMeta{
					Name: name,
					Labels: map[string]string{
						directCSISelector: generatedSelectorValue,
					},
				},
				podSpec,
			},
		},
		appsv1.DaemonSetStatus{},
	}
	return retry(func() error {
		_, err := kClient.AppsV1().DaemonSets(name).Create(ctx, daemonset, metav1.CreateOptions{})
		return err
	})
}
