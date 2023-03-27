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
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

const (
	queueBaseDelay = 100 * time.Millisecond
	queueMaxDelay  = 10 * time.Minute
)

// Event represents a controller event.
type Event struct {
	Type   EventType
	Key    types.UID
	Object runtime.Object
}

// EventHandler represents an interface for controller event handler.
type EventHandler interface {
	ObjectType() runtime.Object
	ListerWatcher() cache.ListerWatcher
	Handle(ctx context.Context, event EventType, object runtime.Object) error
}

// EventType denotes type of event.
type EventType string

// Event types.
const (
	AddEvent    EventType = "Add"
	UpdateEvent EventType = "Update"
	DeleteEvent EventType = "Delete"
)

// Controller object
type Controller struct {
	name          string
	handler       EventHandler
	queue         workqueue.RateLimitingInterface
	informer      cache.SharedIndexInformer
	workerThreads int
	// locking
	locker      map[types.UID]*sync.Mutex
	lockerMutex sync.Mutex
}

// New creates a new controller for the provided handler
func New(name string, handler EventHandler, workers int, resyncPeriod time.Duration) *Controller {
	informer := cache.NewSharedIndexInformer(
		handler.ListerWatcher(),
		handler.ObjectType(),
		resyncPeriod,
		cache.Indexers{},
	)

	queue := workqueue.NewRateLimitingQueue(
		workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemExponentialFailureRateLimiter(queueBaseDelay, queueMaxDelay),
			&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
		),
	)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				klog.ErrorS(err, "unable to process an ADD event")
			} else {
				queue.Add(Event{AddEvent, types.UID(key), obj.(runtime.Object)})
			}
		},
		UpdateFunc: func(old, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(old)
			if err != nil {
				klog.ErrorS(err, "unable to process an UPDATE event")
			} else {
				queue.Add(Event{UpdateEvent, types.UID(key), new.(runtime.Object)})
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				klog.ErrorS(err, "unable to process a DELETE event")
			} else {
				queue.Add(Event{DeleteEvent, types.UID(key), obj.(runtime.Object)})
			}
		},
	})

	return &Controller{
		name:          name,
		informer:      informer,
		queue:         queue,
		workerThreads: workers,
		handler:       handler,
		locker:        map[types.UID]*sync.Mutex{},
	}
}

// Run starts the controller
func (c *Controller) Run(ctx context.Context) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	go c.informer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), c.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	klog.Infof("%s controller synced and ready", c.name)
	for i := 0; i < c.workerThreads; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	<-ctx.Done()
	klog.Infof("Stopping %s controller", c.name)
}

// HasSynced is required for the cache.Controller interface.
func (c *Controller) HasSynced() bool {
	return c.informer.HasSynced()
}

func (c *Controller) runWorker(ctx context.Context) {
	//revive:disable:empty-block
	for c.processNextItem(ctx) {
		// continue looping
	}
	//revive:enable:empty-block
}

func (c *Controller) processNextItem(ctx context.Context) bool {
	event, quit := c.queue.Get()
	if quit {
		return false
	}

	defer c.queue.Done(event)

	if err := c.processItem(ctx, event.(Event)); err != nil {
		c.queue.AddRateLimited(event)
		utilruntime.HandleError(err)
	} else {
		c.queue.Forget(event)
	}

	return true
}

func (c *Controller) getLock(lockKey types.UID) *sync.Mutex {
	c.lockerMutex.Lock()
	defer c.lockerMutex.Unlock()

	if _, found := c.locker[lockKey]; !found {
		c.locker[lockKey] = &sync.Mutex{}
	}
	return c.locker[lockKey]
}

func (c *Controller) lock(lockKey types.UID) {
	c.getLock(lockKey).Lock()
}

func (c *Controller) unlock(lockKey types.UID) {
	c.getLock(lockKey).Unlock()
}

func (c *Controller) processItem(ctx context.Context, event Event) error {
	// Ensure that multiple operations on different versions of the same objects
	// do not happen in parallel
	c.lock(event.Key)
	defer c.unlock(event.Key)
	obj, exists, err := c.informer.GetIndexer().GetByKey(string(event.Key))
	if err != nil {
		return fmt.Errorf("error fetching object with key %s from store; %v", event.Key, err)
	}
	if exists {
		event.Object = obj.(runtime.Object)
	}
	return c.handler.Handle(ctx, event.Type, event.Object)
}
