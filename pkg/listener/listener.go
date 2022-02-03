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
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/minio/directpv/pkg/utils"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	"golang.org/x/time/rate"
)

func getNamespace() string {
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}

	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); ns != "" {
			return ns
		}
	}

	return "default"
}

func newRateLimitingQueue() workqueue.RateLimitingInterface {
	return workqueue.NewRateLimitingQueue(
		workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemExponentialFailureRateLimiter(100*time.Millisecond, 10*time.Minute),
			&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
		),
	)
}

// EventHandler is type of any compatible event handler.
type EventHandler interface {
	ListerWatcher() cache.ListerWatcher
	KubeClient() kubernetes.Interface
	Name() string
	ObjectType() runtime.Object
	Handle(ctx context.Context, args EventArgs) error
}

// EventType denotes type of event.
type EventType int

// Event types.
const (
	AddEvent EventType = iota + 1
	UpdateEvent
	DeleteEvent
)

func (et EventType) String() string {
	switch et {
	case AddEvent:
		return "Add"
	case UpdateEvent:
		return "Update"
	case DeleteEvent:
		return "Delete"
	}
	return ""
}

// EventArgs denotes event arguments.
type EventArgs struct {
	Event     EventType
	Key       string
	Object    interface{}
	OldObject interface{}
}

// Listener is event listener.
type Listener struct {
	handler EventHandler

	LeaseDuration time.Duration
	RenewDeadline time.Duration
	RetryPeriod   time.Duration

	// Controller
	ResyncPeriod time.Duration
	queue        workqueue.RateLimitingInterface
	threadiness  int

	// leader election
	leaderLock string
	identity   string

	locker      map[types.UID]*sync.Mutex
	lockerMutex sync.Mutex
	eventMap    *sync.Map
	indexer     cache.Indexer
}

// NewListener creates new listener with provided event handler.
func NewListener(handler EventHandler, identity, leaderLock string, threadiness int) *Listener {
	if identity == "" {
		identity = leaderLock
	}

	return &Listener{
		handler: handler,

		LeaseDuration: 1 * time.Minute,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   5 * time.Second,

		ResyncPeriod: 1 * time.Minute,
		queue:        newRateLimitingQueue(),
		threadiness:  threadiness,

		leaderLock: leaderLock,
		identity:   identity,

		locker:   map[types.UID]*sync.Mutex{},
		eventMap: &sync.Map{},
		indexer:  cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{}),
	}
}

func (listener *Listener) getOpLock(lockKey types.UID) *sync.Mutex {
	listener.lockerMutex.Lock()
	defer listener.lockerMutex.Unlock()

	if _, found := listener.locker[lockKey]; !found {
		listener.locker[lockKey] = &sync.Mutex{}
	}
	return listener.locker[lockKey]
}

func (listener *Listener) opLock(lockKey types.UID) {
	listener.getOpLock(lockKey).Lock()
}

func (listener *Listener) opUnlock(lockKey types.UID) {
	listener.getOpLock(lockKey).Unlock()
}

func (listener *Listener) handleErr(uuid types.UID, err error) {
	if err == nil {
		listener.eventMap.Delete(uuid)
		return
	}

	listener.queue.AddRateLimited(uuid)
}

func (listener *Listener) processNextItem(ctx context.Context) bool {
	// Wait until there is a new item in the working queue
	value, quit := listener.queue.Get()
	if quit {
		return false
	}
	uuid := value.(types.UID)

	defer listener.queue.Done(uuid)

	value, found := listener.eventMap.Load(uuid)
	if !found {
		return true
	}
	args := value.(EventArgs)

	// Ensure that multiple operations on different versions of the same object
	// do not happen in parallel
	listener.opLock(uuid)
	defer listener.opUnlock(uuid)
	err := listener.handler.Handle(ctx, args)
	switch args.Event {
	case AddEvent:
		if err := listener.indexer.Add(args.Object); err != nil {
			klog.Error(err)
		}
	case UpdateEvent:
		if err := listener.indexer.Update(args.Object); err != nil {
			klog.Error(err)
		}
	case DeleteEvent:
		if err == nil {
			if err := listener.indexer.Delete(args.Object); err != nil {
				klog.Error(err)
			}
			listener.eventMap.Delete(uuid)
		}
	}

	// Handle the error if something went wrong
	listener.handleErr(uuid, err)
	return true
}

func (listener *Listener) runWorker(ctx context.Context) {
	for listener.processNextItem(ctx) {
	}
}

func (listener *Listener) controllerLoop(ctx context.Context) {
	deltaToEventArgs := func(delta cache.Delta, key string) *EventArgs {
		if !delta.Object.(metav1.Object).GetDeletionTimestamp().IsZero() {
			return &EventArgs{
				Event:  DeleteEvent,
				Key:    key,
				Object: delta.Object,
			}
		}

		if oldObject, exists, err := listener.indexer.Get(delta.Object); err == nil && exists {
			if reflect.DeepEqual(delta.Object, oldObject) {
				return nil
			}

			return &EventArgs{
				Event:     UpdateEvent,
				Key:       key,
				Object:    delta.Object,
				OldObject: oldObject,
			}
		}

		return &EventArgs{
			Event:  AddEvent,
			Key:    key,
			Object: delta.Object,
		}
	}

	processFunc := func(obj interface{}) error {
		for _, delta := range obj.(cache.Deltas) {
			switch delta.Type {
			case cache.Sync, cache.Replaced, cache.Added, cache.Updated:
			default:
				continue
			}

			uuid := delta.Object.(metav1.Object).GetUID()
			key, err := cache.MetaNamespaceKeyFunc(delta.Object)
			if err != nil {
				panic(err)
			}

			args := deltaToEventArgs(delta, key)
			if args == nil {
				return nil
			}

			if args.Event == AddEvent {
				value, loaded := listener.eventMap.LoadOrStore(uuid, *args)
				// If add event args is not stored, error out if existing args is update event.
				if loaded && value.(EventArgs).Event == UpdateEvent {
					return fmt.Errorf("cannot add already added object: %s", key)
				}
			} else {
				listener.eventMap.Store(uuid, *args)
			}

			listener.queue.Add(uuid)
		}

		return nil
	}

	config := &cache.Config{
		Queue: cache.NewDeltaFIFOWithOptions(
			cache.DeltaFIFOOptions{
				KnownObjects:          listener.indexer,
				EmitDeltaTypeReplaced: true,
			},
		),
		ListerWatcher:    listener.handler.ListerWatcher(),
		ObjectType:       listener.handler.ObjectType(),
		FullResyncPeriod: listener.ResyncPeriod,
		RetryOnError:     true,
		Process:          processFunc,
	}

	ctrlr := cache.New(config)

	defer utilruntime.HandleCrash()
	defer listener.queue.ShutDown()

	klog.V(3).Infof("Starting %v controller", listener.handler.Name())
	go ctrlr.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), ctrlr.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < listener.threadiness; i++ {
		go wait.UntilWithContext(ctx, listener.runWorker, time.Second)
	}

	<-ctx.Done()
	klog.V(3).Infof("Stopping %v controller", listener.handler.Name())
}

func (listener *Listener) runController(ctx context.Context) {
	go listener.controllerLoop(ctx)
	<-ctx.Done()
}

// Run starts this listener.
func (listener *Listener) Run(ctx context.Context) error {
	id, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("error getting the default leader identity: %w", err)
	}

	namespace := getNamespace()
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(
		&corev1.EventSinkImpl{
			Interface: listener.handler.KubeClient().CoreV1().Events(namespace),
		},
	)

	leader := utils.SanitizeKubeResourceName(fmt.Sprintf("%s/%s", listener.leaderLock, listener.identity))
	resourceLock, err := resourcelock.New(
		resourcelock.LeasesResourceLock,
		namespace,
		leader,
		listener.handler.KubeClient().CoreV1(),
		listener.handler.KubeClient().CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity:      utils.SanitizeKubeResourceName(id),
			EventRecorder: eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{
				Component: leader,
			}),
		},
	)
	if err != nil {
		return err
	}

	leaderConfig := leaderelection.LeaderElectionConfig{
		Lock:          resourceLock,
		LeaseDuration: listener.LeaseDuration,
		RenewDeadline: listener.RenewDeadline,
		RetryPeriod:   listener.RetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				klog.V(2).Info("became leader, starting")
				listener.runController(ctx)
			},
			OnStoppedLeading: func() {
				klog.Errorf("stopped leading")
			},
			OnNewLeader: func(identity string) {
				klog.V(3).Infof("new leader detected, current leader: %s", identity)
			},
		},
	}

	leaderelection.RunOrDie(ctx, leaderConfig)
	return nil // should never reach here
}
