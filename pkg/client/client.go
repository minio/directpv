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
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"k8s.io/client-go/tools/cache"

	"github.com/spf13/viper"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/utils"
	"k8s.io/klog/v2"
)

// GetClientForNonCoreGroupKindVersions gets client for group/kind of given versions.
func GetClientForNonCoreGroupKindVersions(group, kind string, versions ...string) (rest.Interface, *schema.GroupVersionKind, error) {
	gvk, err := GetGroupKindVersions(group, kind, versions...)
	if err != nil {
		return nil, nil, err
	}

	gv := &schema.GroupVersion{
		Group:   gvk.Group,
		Version: gvk.Version,
	}

	config, err := GetKubeConfig()
	if err != nil {
		klog.Fatalf("could not find client configuration: %v", err)
	}
	klog.V(1).Infof("obtained client config successfully")

	config.GroupVersion = gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	client, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, nil, err
	}
	return client, gvk, nil
}

func getKubeConfig() (*rest.Config, string, error) {
	kubeConfig := viper.GetString("kubeconfig")
	if kubeConfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			klog.Infof("could not find home dir: %v", err)
		}
		kubeConfig = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		if config, err = rest.InClusterConfig(); err != nil {
			return nil, "", err
		}
	}
	config.QPS = float32(MaxThreadCount / 2)
	config.Burst = MaxThreadCount
	return config, "", nil
}

// GetKubeConfig gets kubernetes configuration.
func GetKubeConfig() (*rest.Config, error) {
	config, _, err := getKubeConfig()
	return config, err
}

// GetGroupKindVersions gets group/version/kind of given versions.
func GetGroupKindVersions(group, kind string, versions ...string) (*schema.GroupVersionKind, error) {
	discoveryClient := GetDiscoveryClient()
	apiGroupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		klog.V(3).Infof("could not obtain API group resources: %v", err)
		return nil, err
	}
	restMapper := restmapper.NewDiscoveryRESTMapper(apiGroupResources)
	gk := schema.GroupKind{
		Group: group,
		Kind:  kind,
	}
	mapper, err := restMapper.RESTMapping(gk, versions...)
	if err != nil {
		klog.V(3).Infof("could not find valid restmapping: %v", err)
		return nil, err
	}

	gvk := &schema.GroupVersionKind{
		Group:   mapper.Resource.Group,
		Version: mapper.Resource.Version,
		Kind:    mapper.Resource.Resource,
	}
	return gvk, nil
}

type objectResult struct {
	object runtime.Object
	err    error
}

func logYAML(obj interface{}) error {
	yamlString, err := utils.ToYAML(obj)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n---\n\n", yamlString)
	return nil
}

func processObjects(
	ctx context.Context,
	resultCh <-chan objectResult,
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
		if result.err != nil {
			err = result.err
			break
		}

		if !matchFunc(result.object) {
			continue
		}

		if err = applyFunc(result.object); err != nil {
			break
		}

		if dryRun {
			if err := logYAML(result.object); err != nil {
				klog.Errorf("Unable to convert to YAML. %v", err)
			}
			continue
		}
		if writer != nil {
			if err := utils.WriteObject(writer, result.object); err != nil {
				return err
			}
		}

		breakLoop := false
		select {
		case <-ctx.Done():
			breakLoop = true
		case <-stopCh:
			breakLoop = true
		case objectCh <- result.object:
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

func ProcessVolumes(
	ctx context.Context,
	resultCh <-chan ListVolumeResult,
	matchFunc func(*directcsi.DirectCSIVolume) bool,
	applyFunc func(*directcsi.DirectCSIVolume) error,
	processFunc func(context.Context, *directcsi.DirectCSIVolume) error,
	writer io.Writer,
	dryRun bool,
) error {
	objectCh := make(chan objectResult)
	go func() {
		defer close(objectCh)
		for result := range resultCh {
			var oresult objectResult
			if result.Err != nil {
				oresult.err = result.Err
			} else {
				volume := result.Volume
				oresult.object = &volume
			}

			select {
			case <-ctx.Done():
				return
			case objectCh <- oresult:
			}
		}
	}()

	return processObjects(
		ctx,
		objectCh,
		func(object runtime.Object) bool {
			return matchFunc(object.(*directcsi.DirectCSIVolume))
		},
		func(object runtime.Object) error {
			return applyFunc(object.(*directcsi.DirectCSIVolume))
		},
		func(ctx context.Context, object runtime.Object) error {
			return processFunc(ctx, object.(*directcsi.DirectCSIVolume))
		},
		writer,
		dryRun,
	)
}

func ProcessDrives(
	ctx context.Context,
	resultCh <-chan ListDriveResult,
	matchFunc func(*directcsi.DirectCSIDrive) bool,
	applyFunc func(*directcsi.DirectCSIDrive) error,
	processFunc func(context.Context, *directcsi.DirectCSIDrive) error,
	writer io.Writer,
	dryRun bool,
) error {
	objectCh := make(chan objectResult)
	go func() {
		defer close(objectCh)
		for result := range resultCh {
			var oresult objectResult
			if result.Err != nil {
				oresult.err = result.Err
			} else {
				drive := result.Drive
				oresult.object = &drive
			}

			select {
			case <-ctx.Done():
				return
			case objectCh <- oresult:
			}
		}
	}()

	return processObjects(
		ctx,
		objectCh,
		func(object runtime.Object) bool {
			return matchFunc(object.(*directcsi.DirectCSIDrive))
		},
		func(object runtime.Object) error {
			return applyFunc(object.(*directcsi.DirectCSIDrive))
		},
		func(ctx context.Context, object runtime.Object) error {
			return processFunc(ctx, object.(*directcsi.DirectCSIDrive))
		},
		writer,
		dryRun,
	)
}

func DrivesListerWatcher(nodeID string) cache.ListerWatcher {
	labelSelector := ""
	if nodeID != "" {
		labelSelector = fmt.Sprintf("%s=%s", utils.NodeLabelKey, utils.NewLabelValue(nodeID))
	}

	optionsModifier := func(options *metav1.ListOptions) {
		options.LabelSelector = labelSelector
	}

	return cache.NewFilteredListWatchFromClient(
		GetLatestDirectCSIRESTClient(),
		"DirectCSIDrives",
		"",
		optionsModifier,
	)
}

func VolumesListerWatcher(nodeID string) cache.ListerWatcher {
	labelSelector := ""
	if nodeID != "" {
		labelSelector = fmt.Sprintf("%s=%s", utils.NodeLabelKey, utils.NewLabelValue(nodeID))
	}

	optionsModifier := func(options *metav1.ListOptions) {
		options.LabelSelector = labelSelector
	}

	return cache.NewFilteredListWatchFromClient(
		GetLatestDirectCSIRESTClient(),
		"DirectCSIVolumes",
		"",
		optionsModifier,
	)
}
