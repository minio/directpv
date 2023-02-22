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

package listener

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/time/rate"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

const (
	resyncPeriod   = 10 * time.Minute
	queueBaseDelay = 100 * time.Millisecond
	queueMaxDelay  = 10 * time.Minute
	threadiness    = 40
)

// Event represents a controller event.
type Event struct {
	Type      EventType
	Key       string
	Object    runtime.Object
	OldObject runtime.Object
}

// EventHandler represents an interface for controller event handler.
type EventHandler interface {
	ObjectType() runtime.Object
	ListerWatcher() cache.ListerWatcher
	Handle(ctx context.Context, event Event) error
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

// Controller object
type Controller struct {
	name        string
	handler     EventHandler
	queue       workqueue.RateLimitingInterface
	informer    cache.SharedIndexInformer
	threadiness int
}

func newRateLimitingQueue() workqueue.RateLimitingInterface {
	return workqueue.NewRateLimitingQueue(
		workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemExponentialFailureRateLimiter(queueBaseDelay, queueMaxDelay),
			&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
		),
	)
}

// New creates a new controller for the provided handler
func New(ctx context.Context, name string, handler EventHandler) *Controller {
	informer := cache.NewSharedIndexInformer(
		handler.ListerWatcher(),
		handler.ObjectType(),
		resyncPeriod,
		cache.Indexers{},
	)

	queue := newRateLimitingQueue()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			klog.V(5).InfoS("Add event", "key", key, "error", err)
			if err == nil {
				queue.Add(Event{AddEvent, key, obj.(runtime.Object), nil})
			}
		},
		UpdateFunc: func(old, new interface{}) {
			oldKey, oldErr := cache.MetaNamespaceKeyFunc(old)
			newKey, newErr := cache.MetaNamespaceKeyFunc(new)
			klog.V(5).InfoS("Update event", "oldKey", oldKey, "oldErr", oldErr, "newKey", newKey, "newErr", newErr)
			queue.Add(Event{UpdateEvent, oldKey, new.(runtime.Object), old.(runtime.Object)})
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			klog.V(5).InfoS("Delete event", "key", key, "error", err)
			queue.Add(Event{DeleteEvent, key, obj.(runtime.Object), nil})
		},
	})

	return &Controller{
		name:        name,
		informer:    informer,
		queue:       queue,
		threadiness: threadiness,
		handler:     handler,
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
	for i := 0; i < c.threadiness; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	<-ctx.Done()
	klog.Infof("Stopping %s controller", c.name)
}

// HasSynced is required for the cache.Controller interface.
func (c *Controller) HasSynced() bool {
	return c.informer.HasSynced()
}

// LastSyncResourceVersion is required for the cache.Controller interface.
func (c *Controller) LastSyncResourceVersion() string {
	return c.informer.LastSyncResourceVersion()
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextItem(ctx) {
		// continue looping
	}
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

func (c *Controller) processItem(ctx context.Context, event Event) error {
	obj, exists, err := c.informer.GetIndexer().GetByKey(event.Key)
	if err != nil {
		return fmt.Errorf("error fetching object with key %s from store; %v", event.Key, err)
	}
	if exists {
		event.Object = obj.(runtime.Object)
	}
	return c.handler.Handle(ctx, event)
}
