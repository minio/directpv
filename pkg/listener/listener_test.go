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

package listener

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/consts"
	pkgtypes "github.com/minio/directpv/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

const (
	nodeID = "test-node"
	GiB    = 1024 * 1024 * 1024
)

func init() {
	client.FakeInit()
}

type testEventHandler struct {
	t          *testing.T
	handleFunc func(args EventArgs) error
}

func (handler *testEventHandler) ListerWatcher() cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return client.VolumeClient().List(context.TODO(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return client.VolumeClient().Watch(context.TODO(), options)
		},
	}
}

func (handler *testEventHandler) Name() string {
	return "volume"
}

func (handler *testEventHandler) ObjectType() runtime.Object {
	return &pkgtypes.Volume{}
}

func (handler *testEventHandler) Handle(ctx context.Context, args EventArgs) error {
	return handler.handleFunc(args)
}

func startTestController(ctx context.Context, t *testing.T, handler *testEventHandler, threadiness int) {
	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}

	listener := NewListener(handler, "test-volume-controller", hostname, threadiness)
	go func() {
		if err := listener.Run(ctx); err != nil && err != errLeaderElectionDied {
			panic(err)
		}
	}()
}

type condition struct {
	ctype  directpvtypes.VolumeConditionType
	status metav1.ConditionStatus
	reason directpvtypes.VolumeConditionReason
}

func (c condition) toCondition() metav1.Condition {
	return metav1.Condition{
		Type:               string(c.ctype),
		Status:             c.status,
		Reason:             string(c.reason),
		LastTransitionTime: metav1.Now(),
	}
}

func newVolume(name, uid string, capacity int64, conditions []condition) *pkgtypes.Volume {
	var metaConditions []metav1.Condition
	for _, c := range conditions {
		metaConditions = append(metaConditions, c.toCondition())
	}

	return &pkgtypes.Volume{
		TypeMeta: pkgtypes.NewVolumeTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Finalizers: []string{
				string(consts.VolumeFinalizerPurgeProtection),
			},
			UID: types.UID(uid),
		},
		Status: pkgtypes.VolumeStatus{
			NodeName:      nodeID,
			HostPath:      "hostpath",
			DriveName:     "test-drive",
			TotalCapacity: capacity,
			Conditions:    metaConditions,
		},
	}
}

func getHandleFunc(t *testing.T, event EventType, volumes ...*pkgtypes.Volume) (<-chan struct{}, func(EventArgs) error) {
	doneCh := make(chan struct{})
	i := 0
	errOccured := false
	return doneCh, func(args EventArgs) (err error) {
		defer func() {
			i++
			if errOccured || i == len(volumes) {
				close(doneCh)
			}
		}()

		if args.Event != event {
			errOccured = true
			t.Fatalf("expected: %v, got: %v", event, args.Event)
		}

		volume := args.Object.(*pkgtypes.Volume)
		if !reflect.DeepEqual(volumes[i], volume) {
			errOccured = true
			t.Fatalf("received volume is not equal to volumes[%v]", i)
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

func TestListener(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	testHandler := &testEventHandler{
		t: t,
	}

	// Sync
	volumes := []*pkgtypes.Volume{
		newVolume("test-volume-1", "1", 2*GiB, []condition{
			{directpvtypes.VolumeConditionTypeStaged, metav1.ConditionTrue, directpvtypes.VolumeConditionReasonInUse},
			{directpvtypes.VolumeConditionTypePublished, metav1.ConditionFalse, directpvtypes.VolumeConditionReasonNotInUse},
			{directpvtypes.VolumeConditionTypeReady, metav1.ConditionTrue, directpvtypes.VolumeConditionReasonReady},
		}),
		newVolume("test-volume-2", "2", 3*GiB, []condition{
			{directpvtypes.VolumeConditionTypeStaged, metav1.ConditionFalse, directpvtypes.VolumeConditionReasonInUse},
			{directpvtypes.VolumeConditionTypePublished, metav1.ConditionFalse, directpvtypes.VolumeConditionReasonNotInUse},
			{directpvtypes.VolumeConditionTypeReady, metav1.ConditionTrue, directpvtypes.VolumeConditionReasonReady},
		}),
	}

	clientset := pkgtypes.NewExtFakeClientset(clientsetfake.NewSimpleClientset(toRuntimeObjects(volumes...)...))
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	doneCh, handleFunc := getHandleFunc(t, AddEvent, volumes...)
	testHandler.handleFunc = handleFunc
	startTestController(ctx, t, testHandler, 1)
	<-doneCh

	// Update
	volumes = []*pkgtypes.Volume{
		newVolume("test-volume-1", "1", 4*GiB, []condition{
			{directpvtypes.VolumeConditionTypeStaged, metav1.ConditionTrue, directpvtypes.VolumeConditionReasonInUse},
			{directpvtypes.VolumeConditionTypePublished, metav1.ConditionFalse, directpvtypes.VolumeConditionReasonNotInUse},
			{directpvtypes.VolumeConditionTypeReady, metav1.ConditionTrue, directpvtypes.VolumeConditionReasonReady},
		}),
		newVolume("test-volume-1", "1", 6*GiB, []condition{
			{directpvtypes.VolumeConditionTypeStaged, metav1.ConditionTrue, directpvtypes.VolumeConditionReasonInUse},
			{directpvtypes.VolumeConditionTypePublished, metav1.ConditionFalse, directpvtypes.VolumeConditionReasonNotInUse},
			{directpvtypes.VolumeConditionTypeReady, metav1.ConditionTrue, directpvtypes.VolumeConditionReasonReady},
		}),
	}
	doneCh, handleFunc = getHandleFunc(t, UpdateEvent, volumes[1])
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

	// Delete
	volumes = []*pkgtypes.Volume{
		newVolume("test-volume-1", "1", 4*GiB, []condition{
			{directpvtypes.VolumeConditionTypeStaged, metav1.ConditionTrue, directpvtypes.VolumeConditionReasonInUse},
			{directpvtypes.VolumeConditionTypePublished, metav1.ConditionFalse, directpvtypes.VolumeConditionReasonNotInUse},
			{directpvtypes.VolumeConditionTypeReady, metav1.ConditionTrue, directpvtypes.VolumeConditionReasonReady},
		}),
	}
	now := metav1.Now()
	volumes[0].DeletionTimestamp = &now
	doneCh, handleFunc = getHandleFunc(t, DeleteEvent, volumes...)
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
		newVolume("test-volume-1", "1", 512*GiB, []condition{
			{directpvtypes.VolumeConditionTypeStaged, metav1.ConditionTrue, directpvtypes.VolumeConditionReasonInUse},
			{directpvtypes.VolumeConditionTypePublished, metav1.ConditionFalse, directpvtypes.VolumeConditionReasonNotInUse},
			{directpvtypes.VolumeConditionTypeReady, metav1.ConditionTrue, directpvtypes.VolumeConditionReasonReady},
		}),
	}

	stopCh := make(chan struct{})
	raiseRetryErr := true
	testHandler.handleFunc = func(args EventArgs) (err error) {
		if raiseRetryErr {
			raiseRetryErr = false
			return errors.New("retry again")
		}

		defer close(stopCh)
		if args.Event != AddEvent {
			t.Fatalf("expected: %v, got: %v", AddEvent, args.Event)
		}

		volume := args.Object.(*pkgtypes.Volume)
		if !reflect.DeepEqual(volumes[0], volume) {
			t.Fatalf("received volume is not equal to volumes[0]")
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
}

func TestListenerParallel(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	testHandler := &testEventHandler{
		t: t,
	}

	clientset := pkgtypes.NewExtFakeClientset(clientsetfake.NewSimpleClientset(
		toRuntimeObjects(newVolume("test-volume-1", "1", 2*GiB, []condition{
			{directpvtypes.VolumeConditionTypeStaged, metav1.ConditionTrue, directpvtypes.VolumeConditionReasonInUse},
			{directpvtypes.VolumeConditionTypePublished, metav1.ConditionFalse, directpvtypes.VolumeConditionReasonNotInUse},
			{directpvtypes.VolumeConditionTypeReady, metav1.ConditionTrue, directpvtypes.VolumeConditionReasonReady},
		}))...,
	))
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	doneCh := make(chan struct{})
	stopCh := make(chan struct{})
	testHandler.handleFunc = func(args EventArgs) (err error) {
		volume := args.Object.(*pkgtypes.Volume)
		if volume.Status.TotalCapacity == 512*GiB {
			close(stopCh)
		}
		if volume.Status.TotalCapacity > 1*GiB {
			return errors.New("no space left on the device")
		}

		defer close(doneCh)
		if args.Event != UpdateEvent {
			t.Fatalf("expected: %v, got: %v", UpdateEvent, args.Event)
		}

		if volume.Status.TotalCapacity != 1*GiB {
			t.Fatalf("TotalCapacity: expected: %v, got: %v", GiB, volume.Status.TotalCapacity)
		}

		return nil
	}
	startTestController(ctx, t, testHandler, 40)

	_, err := client.VolumeClient().Update(
		ctx,
		newVolume("test-volume-1", "1", 512*GiB, []condition{
			{directpvtypes.VolumeConditionTypeStaged, metav1.ConditionTrue, directpvtypes.VolumeConditionReasonInUse},
			{directpvtypes.VolumeConditionTypePublished, metav1.ConditionFalse, directpvtypes.VolumeConditionReasonNotInUse},
			{directpvtypes.VolumeConditionTypeReady, metav1.ConditionTrue, directpvtypes.VolumeConditionReasonReady},
		}),
		metav1.UpdateOptions{
			TypeMeta: pkgtypes.NewVolumeTypeMeta(),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	<-stopCh

	_, err = client.VolumeClient().Update(
		ctx,
		newVolume("test-volume-1", "1", 1*GiB, []condition{
			{directpvtypes.VolumeConditionTypeStaged, metav1.ConditionTrue, directpvtypes.VolumeConditionReasonInUse},
			{directpvtypes.VolumeConditionTypePublished, metav1.ConditionFalse, directpvtypes.VolumeConditionReasonNotInUse},
			{directpvtypes.VolumeConditionTypeReady, metav1.ConditionTrue, directpvtypes.VolumeConditionReasonReady},
		}),
		metav1.UpdateOptions{
			TypeMeta: pkgtypes.NewVolumeTypeMeta(),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	<-doneCh
}
