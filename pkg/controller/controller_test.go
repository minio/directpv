// This file is part of MinIO DirectPV
// Copyright (c) 2023 MinIO, Inc.
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

package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/dustin/go-humanize"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	pkgtypes "github.com/minio/directpv/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

const (
	nodeID = "test-node"
)

func init() {
	client.FakeInit()
}

type testEventHandler struct {
	t          *testing.T
	handleFunc func(eventType EventType, object runtime.Object) error
}

func (handler *testEventHandler) ListerWatcher() cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.LabelSelector = fmt.Sprintf("%s=%s", directpvtypes.NodeLabelKey, nodeID)
			return client.VolumeClient().List(context.TODO(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			if options.SendInitialEvents != nil && *options.SendInitialEvents {
				return nil, errors.New("fake client does not support SendInitialEvents")
			}
			options.LabelSelector = fmt.Sprintf("%s=%s", directpvtypes.NodeLabelKey, nodeID)
			return client.VolumeClient().Watch(context.TODO(), options)
		},
	}
}

func (handler *testEventHandler) ObjectType() runtime.Object {
	return &pkgtypes.Volume{}
}

func (handler *testEventHandler) Handle(_ context.Context, eventType EventType, object runtime.Object) error {
	return handler.handleFunc(eventType, object)
}

func startTestController(ctx context.Context, handler *testEventHandler) {
	controller := New("test-volume-controller", handler, 1, 10*time.Second)
	go controller.Run(ctx)
}

func newVolume(name, uid string, capacity int64) *pkgtypes.Volume {
	volume := pkgtypes.NewVolume(
		name,
		uid,
		nodeID,
		"sda",
		"sda",
		capacity,
	)
	volume.UID = types.UID(uid)
	volume.Status.DataPath = "datapath"
	return volume
}

func getHandleFunc(t *testing.T, expectedEventType EventType, volumesMap map[string]*pkgtypes.Volume) (<-chan struct{}, func(EventType, runtime.Object) error) {
	doneCh := make(chan struct{})
	processed := 0
	errOccured := false
	return doneCh, func(eventType EventType, object runtime.Object) (err error) {
		defer func() {
			processed++
			if errOccured || processed == len(volumesMap) {
				close(doneCh)
			}
		}()

		volume := object.(*pkgtypes.Volume)

		if expectedEventType != eventType {
			errOccured = true
			t.Fatalf("expected: %v, got: %v", expectedEventType, eventType)
		}

		if !reflect.DeepEqual(volumesMap[volume.Name], volume) {
			errOccured = true
			t.Fatalf("received volume is not equal to volumesMap[%v]", volume.Name)
		}

		return nil
	}
}

func toRuntimeObjects(volumes ...*pkgtypes.Volume) (objects []runtime.Object) {
	for _, volume := range volumes {
		objects = append(objects, volume)
	}
	return objects
}

func TestController(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancelFunc()

	testHandler := &testEventHandler{
		t: t,
	}

	volumes := []*pkgtypes.Volume{
		newVolume("test-volume-1", "1", 2*humanize.GiByte),
		newVolume("test-volume-2", "2", 3*humanize.GiByte),
	}

	volumesMap := map[string]*pkgtypes.Volume{
		"test-volume-1": volumes[0],
		"test-volume-2": volumes[1],
	}

	clientset := pkgtypes.NewExtFakeClientset(clientsetfake.NewSimpleClientset(toRuntimeObjects(volumes...)...))
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	doneCh, handleFunc := getHandleFunc(t, AddEvent, volumesMap)
	testHandler.handleFunc = handleFunc
	startTestController(ctx, testHandler)
	<-doneCh

	volumes = []*pkgtypes.Volume{
		newVolume("test-volume-1", "1", 4*humanize.GiByte),
		newVolume("test-volume-2", "1", 6*humanize.GiByte),
	}
	volumesMap = map[string]*pkgtypes.Volume{
		"test-volume-1": volumes[0],
		"test-volume-2": volumes[1],
	}
	doneCh, handleFunc = getHandleFunc(t, UpdateEvent, volumesMap)
	testHandler.handleFunc = handleFunc
	for _, volume := range volumes {
		_, err := client.VolumeClient().Update(
			ctx,
			volume,
			metav1.UpdateOptions{
				TypeMeta: pkgtypes.NewVolumeTypeMeta(),
			},
		)
		if err != nil {
			t.Fatal(err)
		}
	}
	<-doneCh

	// Retry on error
	volumes = []*pkgtypes.Volume{
		newVolume("test-volume-1", "1", 4*humanize.GiByte),
	}
	volumesMap = map[string]*pkgtypes.Volume{
		"test-volume-1": volumes[0],
	}
	stopCh := make(chan struct{})
	raiseRetryErr := true
	testHandler.handleFunc = func(eventType EventType, object runtime.Object) (err error) {
		if raiseRetryErr {
			raiseRetryErr = false
			return errors.New("returning an error to test controller retry")
		}

		defer close(stopCh)
		if eventType != UpdateEvent {
			t.Fatalf("expected: %v, got: %v", UpdateEvent, eventType)
		}

		volume := object.(*pkgtypes.Volume)
		if !reflect.DeepEqual(volumesMap[volume.Name], volume) {
			t.Fatalf("received volume is not equal to volumesMap[%s]", volume.Name)
		}
		return nil
	}

	for _, volume := range volumes {
		_, err := client.VolumeClient().Update(
			ctx,
			volume,
			metav1.UpdateOptions{
				TypeMeta: pkgtypes.NewVolumeTypeMeta(),
			},
		)
		if err != nil {
			t.Fatal(err)
		}
	}
	<-stopCh

	// Delete
	volumes = []*pkgtypes.Volume{
		newVolume("test-volume-1", "1", 4*humanize.GiByte),
		newVolume("test-volume-2", "1", 6*humanize.GiByte),
	}
	volumesMap = map[string]*pkgtypes.Volume{
		"test-volume-1": volumes[0],
		"test-volume-2": volumes[1],
	}
	doneCh, handleFunc = getHandleFunc(t, DeleteEvent, volumesMap)
	testHandler.handleFunc = handleFunc
	for _, volume := range volumes {
		if err := client.VolumeClient().Delete(
			ctx,
			volume.Name,
			metav1.DeleteOptions{
				TypeMeta: pkgtypes.NewVolumeTypeMeta(),
			},
		); err != nil {
			t.Fatal(err)
		}
	}
	<-doneCh
}
