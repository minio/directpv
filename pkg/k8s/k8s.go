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
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

// GetKubeConfig gets kubernetes configuration from command line argument,
// ~/.kube/config or in-cluster configuration.
func GetKubeConfig() (*rest.Config, error) {
	kubeconfigPath := viper.GetString("kubeconfig")
	if kubeconfigPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			klog.ErrorS(err, "unable to find user home directory")
		}
		kubeconfigPath = path.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		if config, err = rest.InClusterConfig(); err != nil {
			return nil, err
		}
	}

	config.QPS = float32(MaxThreadCount / 2)
	config.Burst = MaxThreadCount
	return config, nil
}

// GetGroupVersionKind gets group/version/kind of given versions.
func GetGroupVersionKind(group, kind string, versions ...string) (*schema.GroupVersionKind, error) {
	apiGroupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		klog.ErrorS(err, "unable to get API group resources")
		return nil, err
	}
	restMapper := restmapper.NewDiscoveryRESTMapper(apiGroupResources)
	mapper, err := restMapper.RESTMapping(
		schema.GroupKind{
			Group: group,
			Kind:  kind,
		},
		versions...,
	)
	if err != nil {
		return nil, err
	}

	return &schema.GroupVersionKind{
		Group:   mapper.Resource.Group,
		Version: mapper.Resource.Version,
		Kind:    mapper.Resource.Resource,
	}, nil
}

// GetClientForNonCoreGroupVersionKind gets client for group/kind of given versions.
func GetClientForNonCoreGroupVersionKind(group, kind string, versions ...string) (rest.Interface, *schema.GroupVersionKind, error) {
	gvk, err := GetGroupVersionKind(group, kind, versions...)
	if err != nil {
		return nil, nil, err
	}

	config := *kubeConfig
	config.GroupVersion = &schema.GroupVersion{
		Group:   gvk.Group,
		Version: gvk.Version,
	}
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, nil, err
	}

	return client, gvk, nil
}

// IsCondition checks whether type/status/reason/message in conditions or not.
func IsCondition(conditions []metav1.Condition, ctype string, status metav1.ConditionStatus, reason, message string) bool {
	for i := range conditions {
		if conditions[i].Type == ctype &&
			conditions[i].Status == status &&
			conditions[i].Reason == reason &&
			conditions[i].Message == message {
			return true
		}
	}
	return false
}

// IsConditionStatus checks whether type/status in conditions or not.
func IsConditionStatus(conditions []metav1.Condition, ctype string, status metav1.ConditionStatus) bool {
	for i := range conditions {
		if conditions[i].Type == ctype && conditions[i].Status == status {
			return true
		}
	}
	return false
}

// UpdateCondition updates type/status/reason/message of conditions matched by condition type.
func UpdateCondition(conditions []metav1.Condition, ctype string, status metav1.ConditionStatus, reason, message string) {
	for i := range conditions {
		if conditions[i].Type == ctype {
			conditions[i].Status = status
			conditions[i].Reason = reason
			conditions[i].Message = message
			conditions[i].LastTransitionTime = metav1.Now()
			break
		}
	}
}

// MatchTrueConditions matches whether types and status list are in a true conditions or not.
func MatchTrueConditions(conditions []metav1.Condition, types, statusList []string) bool {
	for i := range types {
		types[i] = strings.ToLower(types[i])
	}
	for i := range statusList {
		statusList[i] = strings.ToLower(statusList[i])
	}

	statusMatches := 0
	for _, condition := range conditions {
		ctype := strings.ToLower(condition.Type)
		if condition.Status == metav1.ConditionTrue && utils.StringIn(types, ctype) && utils.StringIn(statusList, ctype) {
			statusMatches++
		}
	}

	return statusMatches == len(statusList)
}

// BoolToConditionStatus converts boolean value to condition status.
func BoolToConditionStatus(val bool) metav1.ConditionStatus {
	if val {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}

// RemoveFinalizer removes finalizer in object meta.
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

// SanitizeResourceName - Sanitize given name to a valid kubernetes name format.
// RegEx for a kubernetes name is
//
//	([a-z0-9][-a-z0-9]*)?[a-z0-9]
//
// with a max length of 253
//
// WARNING: This function will truncate to 253 bytes if the input is longer
func SanitizeResourceName(name string) string {
	if len(name) > 253 {
		name = name[:253]
	}

	result := []rune(strings.ToLower(name))
	for i, r := range result {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
		default:
			if i == 0 {
				result[i] = '0'
			} else {
				result[i] = '-'
			}
		}
	}

	return string(result)
}

type ObjectResult struct {
	Object runtime.Object
	Err    error
}

func logYAML(obj interface{}) error {
	yamlString, err := utils.ToYAML(obj)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n---\n\n", yamlString)
	return nil
}

func ProcessObjects(
	ctx context.Context,
	resultCh <-chan ObjectResult,
	matchFunc func(runtime.Object) bool,
	applyFunc func(runtime.Object) error,
	processFunc func(context.Context, runtime.Object) error,
	writer io.Writer,
	dryRun bool,
) error {
	stopCh := make(chan struct{})
	var stopChMu int32
	closeStopCh := func() {
		if atomic.AddInt32(&stopChMu, 1) == 1 {
			close(stopCh)
		}
	}
	defer closeStopCh()

	objectCh := make(chan runtime.Object)
	var wg sync.WaitGroup

	// Start MaxThreadCount workers.
	var errs []error
	for i := 0; i < MaxThreadCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-stopCh:
					return
				case object, ok := <-objectCh:
					if !ok {
						return
					}
					if err := processFunc(ctx, object); err != nil {
						errs = append(errs, err)
						defer closeStopCh()
						return
					}
				}
			}
		}()
	}

	var err error
	for result := range resultCh {
		if result.Err != nil {
			err = result.Err
			break
		}

		if !matchFunc(result.Object) {
			continue
		}

		if err = applyFunc(result.Object); err != nil {
			break
		}

		if dryRun {
			if err := logYAML(result.Object); err != nil {
				klog.ErrorS(err, "unable log object as YAML string")
			}
			continue
		}
		if writer != nil {
			if err := utils.WriteObject(writer, result.Object); err != nil {
				return err
			}
		}

		breakLoop := false
		select {
		case <-ctx.Done():
			breakLoop = true
		case <-stopCh:
			breakLoop = true
		case objectCh <- result.Object:
		}

		if breakLoop {
			break
		}
	}

	close(objectCh)
	wg.Wait()

	if err != nil {
		return err
	}

	msgs := []string{}
	for _, err := range errs {
		msgs = append(msgs, err.Error())
	}
	if msg := strings.Join(msgs, "; "); msg != "" {
		return errors.New(msg)
	}

	return nil
}
