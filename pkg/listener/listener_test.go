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

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/clientset"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

const (
	nodeID = "test-node"
	GiB    = 1024 * 1024 * 1024
)

type testEventHandler struct {
	t               *testing.T
	kubeClient      kubernetes.Interface
	directCSIClient clientset.Interface
	handleFunc      func(args EventArgs) error
}

func (handler *testEventHandler) ListerWatcher() cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return handler.directCSIClient.DirectV1beta4().DirectCSIVolumes().List(context.TODO(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return handler.directCSIClient.DirectV1beta4().DirectCSIVolumes().Watch(context.TODO(), options)
		},
	}
}

func (handler *testEventHandler) KubeClient() kubernetes.Interface {
	return handler.kubeClient
}

func (handler *testEventHandler) Name() string {
	return "volume"
}

func (handler *testEventHandler) ObjectType() runtime.Object {
	return &directcsi.DirectCSIVolume{}
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
		if err := listener.Run(ctx); err != nil {
			panic(err)
		}
	}()
}

type condition struct {
	ctype  directcsi.DirectCSIVolumeCondition
	status metav1.ConditionStatus
	reason directcsi.DirectCSIVolumeReason
}

func (c condition) toCondition() metav1.Condition {
	return metav1.Condition{
		Type:               string(c.ctype),
		Status:             c.status,
		Reason:             string(c.reason),
		LastTransitionTime: metav1.Now(),
	}
}

func newDirectCSIVolume(name, uid string, capacity int64, conditions []condition) *directcsi.DirectCSIVolume {
	var metaConditions []metav1.Condition
	for _, c := range conditions {
		metaConditions = append(metaConditions, c.toCondition())
	}

	return &directcsi.DirectCSIVolume{
		TypeMeta: utils.DirectCSIVolumeTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Finalizers: []string{
				string(directcsi.DirectCSIVolumeFinalizerPurgeProtection),
			},
			UID: types.UID(uid),
		},
		Status: directcsi.DirectCSIVolumeStatus{
			NodeName:      nodeID,
			HostPath:      "hostpath",
			Drive:         "test-drive",
			TotalCapacity: capacity,
			Conditions:    metaConditions,
		},
	}
}

func getHandleFunc(t *testing.T, event EventType, volumes ...*directcsi.DirectCSIVolume) (<-chan struct{}, func(EventArgs) error) {
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

		volume := args.Object.(*directcsi.DirectCSIVolume)
		if !reflect.DeepEqual(volumes[i], volume) {
			errOccured = true
			t.Fatalf("received volume is not equal to volumes[%v]", i)
		}
		return nil
	}
}

func toRuntimeObjects(volumes ...*directcsi.DirectCSIVolume) (objects []runtime.Object) {
	for _, volume := range volumes {
		objects = append(objects, volume)
	}
	return objects
}

func TestListener(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	testHandler := &testEventHandler{
		kubeClient: kubernetesfake.NewSimpleClientset(),
		t:          t,
	}

	// Sync
	volumes := []*directcsi.DirectCSIVolume{
		newDirectCSIVolume("test-volume-1", "1", 2*GiB, []condition{
			{directcsi.DirectCSIVolumeConditionStaged, metav1.ConditionTrue, directcsi.DirectCSIVolumeReasonInUse},
			{directcsi.DirectCSIVolumeConditionPublished, metav1.ConditionFalse, directcsi.DirectCSIVolumeReasonNotInUse},
			{directcsi.DirectCSIVolumeConditionReady, metav1.ConditionTrue, directcsi.DirectCSIVolumeReasonReady},
		}),
		newDirectCSIVolume("test-volume-2", "2", 3*GiB, []condition{
			{directcsi.DirectCSIVolumeConditionStaged, metav1.ConditionFalse, directcsi.DirectCSIVolumeReasonInUse},
			{directcsi.DirectCSIVolumeConditionPublished, metav1.ConditionFalse, directcsi.DirectCSIVolumeReasonNotInUse},
			{directcsi.DirectCSIVolumeConditionReady, metav1.ConditionTrue, directcsi.DirectCSIVolumeReasonReady},
		}),
	}
	testHandler.directCSIClient = clientsetfake.NewSimpleClientset(toRuntimeObjects(volumes...)...)

	doneCh, handleFunc := getHandleFunc(t, AddEvent, volumes...)
	testHandler.handleFunc = handleFunc
	startTestController(ctx, t, testHandler, 1)
	<-doneCh

	// Update
	volumes = []*directcsi.DirectCSIVolume{
		newDirectCSIVolume("test-volume-1", "1", 4*GiB, []condition{
			{directcsi.DirectCSIVolumeConditionStaged, metav1.ConditionTrue, directcsi.DirectCSIVolumeReasonInUse},
			{directcsi.DirectCSIVolumeConditionPublished, metav1.ConditionFalse, directcsi.DirectCSIVolumeReasonNotInUse},
			{directcsi.DirectCSIVolumeConditionReady, metav1.ConditionTrue, directcsi.DirectCSIVolumeReasonReady},
		}),
		newDirectCSIVolume("test-volume-1", "1", 6*GiB, []condition{
			{directcsi.DirectCSIVolumeConditionStaged, metav1.ConditionTrue, directcsi.DirectCSIVolumeReasonInUse},
			{directcsi.DirectCSIVolumeConditionPublished, metav1.ConditionFalse, directcsi.DirectCSIVolumeReasonNotInUse},
			{directcsi.DirectCSIVolumeConditionReady, metav1.ConditionTrue, directcsi.DirectCSIVolumeReasonReady},
		}),
	}
	doneCh, handleFunc = getHandleFunc(t, UpdateEvent, volumes[1])
	testHandler.handleFunc = handleFunc
	for _, volume := range volumes {
		_, err := testHandler.directCSIClient.DirectV1beta4().DirectCSIVolumes().Update(
			ctx,
			volume,
			metav1.UpdateOptions{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
			},
		)
		if err != nil {
			t.Fatal(err)
		}
	}
	<-doneCh

	// Delete
	volumes = []*directcsi.DirectCSIVolume{
		newDirectCSIVolume("test-volume-1", "1", 4*GiB, []condition{
			{directcsi.DirectCSIVolumeConditionStaged, metav1.ConditionTrue, directcsi.DirectCSIVolumeReasonInUse},
			{directcsi.DirectCSIVolumeConditionPublished, metav1.ConditionFalse, directcsi.DirectCSIVolumeReasonNotInUse},
			{directcsi.DirectCSIVolumeConditionReady, metav1.ConditionTrue, directcsi.DirectCSIVolumeReasonReady},
		}),
	}
	now := metav1.Now()
	volumes[0].DeletionTimestamp = &now
	doneCh, handleFunc = getHandleFunc(t, DeleteEvent, volumes...)
	testHandler.handleFunc = handleFunc
	for _, volume := range volumes {
		_, err := testHandler.directCSIClient.DirectV1beta4().DirectCSIVolumes().Update(
			ctx,
			volume,
			metav1.UpdateOptions{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
			},
		)
		if err != nil {
			t.Fatal(err)
		}
	}
	<-doneCh

	// Retry on error
	volumes = []*directcsi.DirectCSIVolume{
		newDirectCSIVolume("test-volume-1", "1", 512*GiB, []condition{
			{directcsi.DirectCSIVolumeConditionStaged, metav1.ConditionTrue, directcsi.DirectCSIVolumeReasonInUse},
			{directcsi.DirectCSIVolumeConditionPublished, metav1.ConditionFalse, directcsi.DirectCSIVolumeReasonNotInUse},
			{directcsi.DirectCSIVolumeConditionReady, metav1.ConditionTrue, directcsi.DirectCSIVolumeReasonReady},
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

		volume := args.Object.(*directcsi.DirectCSIVolume)
		if !reflect.DeepEqual(volumes[0], volume) {
			t.Fatalf("received volume is not equal to volumes[0]")
		}
		return nil
	}

	for _, volume := range volumes {
		_, err := testHandler.directCSIClient.DirectV1beta4().DirectCSIVolumes().Update(
			ctx,
			volume,
			metav1.UpdateOptions{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
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
		kubeClient: kubernetesfake.NewSimpleClientset(),
		t:          t,
	}
	testHandler.directCSIClient = clientsetfake.NewSimpleClientset(
		toRuntimeObjects(newDirectCSIVolume("test-volume-1", "1", 2*GiB, []condition{
			{directcsi.DirectCSIVolumeConditionStaged, metav1.ConditionTrue, directcsi.DirectCSIVolumeReasonInUse},
			{directcsi.DirectCSIVolumeConditionPublished, metav1.ConditionFalse, directcsi.DirectCSIVolumeReasonNotInUse},
			{directcsi.DirectCSIVolumeConditionReady, metav1.ConditionTrue, directcsi.DirectCSIVolumeReasonReady},
		}))...,
	)

	doneCh := make(chan struct{})
	stopCh := make(chan struct{})
	testHandler.handleFunc = func(args EventArgs) (err error) {
		volume := args.Object.(*directcsi.DirectCSIVolume)
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

	_, err := testHandler.directCSIClient.DirectV1beta4().DirectCSIVolumes().Update(
		ctx,
		newDirectCSIVolume("test-volume-1", "1", 512*GiB, []condition{
			{directcsi.DirectCSIVolumeConditionStaged, metav1.ConditionTrue, directcsi.DirectCSIVolumeReasonInUse},
			{directcsi.DirectCSIVolumeConditionPublished, metav1.ConditionFalse, directcsi.DirectCSIVolumeReasonNotInUse},
			{directcsi.DirectCSIVolumeConditionReady, metav1.ConditionTrue, directcsi.DirectCSIVolumeReasonReady},
		}),
		metav1.UpdateOptions{
			TypeMeta: utils.DirectCSIVolumeTypeMeta(),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	<-stopCh

	_, err = testHandler.directCSIClient.DirectV1beta4().DirectCSIVolumes().Update(
		ctx,
		newDirectCSIVolume("test-volume-1", "1", 1*GiB, []condition{
			{directcsi.DirectCSIVolumeConditionStaged, metav1.ConditionTrue, directcsi.DirectCSIVolumeReasonInUse},
			{directcsi.DirectCSIVolumeConditionPublished, metav1.ConditionFalse, directcsi.DirectCSIVolumeReasonNotInUse},
			{directcsi.DirectCSIVolumeConditionReady, metav1.ConditionTrue, directcsi.DirectCSIVolumeReasonReady},
		}),
		metav1.UpdateOptions{
			TypeMeta: utils.DirectCSIVolumeTypeMeta(),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	<-doneCh
}
