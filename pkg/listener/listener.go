// This file is part of MinIO Direct CSI
// Copyright (c) 2020 MinIO, Inc.
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
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	// objectstorage
	v1beta1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta1"
	"github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// k8s api
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	// k8s client
	"k8s.io/apimachinery/pkg/types"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	// logging
	"github.com/golang/glog"
)

type addFunc func(ctx context.Context, obj interface{}) error
type updateFunc func(ctx context.Context, old, new interface{}) error
type deleteFunc func(ctx context.Context, obj interface{}) error

type addOp struct {
	Object  interface{}
	AddFunc *addFunc
	Indexer cache.Indexer

	Key string
}

func (a addOp) String() string {
	return a.Key
}

type updateOp struct {
	OldObject  interface{}
	NewObject  interface{}
	UpdateFunc *updateFunc
	Indexer    cache.Indexer

	Key string
}

func (u updateOp) String() string {
	return u.Key
}

type deleteOp struct {
	Object     interface{}
	DeleteFunc *deleteFunc
	Indexer    cache.Indexer

	Key string
}

func (d deleteOp) String() string {
	return d.Key
}

type DirectCSIController struct {
	LeaseDuration time.Duration
	RenewDeadline time.Duration
	RetryPeriod   time.Duration

	// Controller
	ResyncPeriod time.Duration
	queue        workqueue.RateLimitingInterface
	threadiness  int

	// Listeners
	DirectCSIVolumeListener DirectCSIVolumeListener
	DirectCSIDriveListener  DirectCSIDriveListener

	// leader election
	leaderLock string
	identity   string

	// internal
	initialized     bool
	directcsiClient clientset.Interface
	kubeClient      kubeclientset.Interface

	lockerLock sync.Mutex
	locker     map[types.UID]*sync.Mutex
	opMap      *sync.Map
}

func NewDefaultDirectCSIController(identity string, leaderLockName string, threads int) (*DirectCSIController, error) {
	rateLimit := workqueue.NewItemExponentialFailureRateLimiter(100*time.Millisecond, 30*time.Second)
	return NewDirectCSIController(identity, leaderLockName, threads, rateLimit)
}

func NewDirectCSIController(identity string, leaderLockName string, threads int, limiter workqueue.RateLimiter) (*DirectCSIController, error) {
	var err error
	directcsiClient := utils.GetDirectClientset()
	kubeClient := utils.GetKubeClient()

	id := identity
	if id == "" {
		id, err = os.Hostname()
		if err != nil {
			return nil, err
		}
	}

	return &DirectCSIController{
		identity:        id,
		kubeClient:      kubeClient,
		directcsiClient: directcsiClient,
		initialized:     false,
		leaderLock:      leaderLockName,
		queue:           workqueue.NewRateLimitingQueue(limiter),
		threadiness:     threads,

		ResyncPeriod:  30 * time.Second,
		LeaseDuration: 60 * time.Second,
		RenewDeadline: 15 * time.Second,
		RetryPeriod:   5 * time.Second,

		opMap: &sync.Map{},
	}, nil
}

// Run - runs the controller. Note that ctx must be cancellable i.e. ctx.Done() should not return nil
func (c *DirectCSIController) Run(ctx context.Context) error {
	if !c.initialized {
		fmt.Errorf("Uninitialized controller. Atleast 1 listener should be added")
	}

	ns := func() string {
		if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
			return ns
		}

		if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
			if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
				return ns
			}
		}
		return "default"
	}()

	sanitize := func(n string) string {
		re := regexp.MustCompile("[^a-zA-Z0-9-]")
		name := strings.ToLower(re.ReplaceAllString(n, "-"))
		if name[len(name)-1] == '-' {
			// name must not end with '-'
			name = name + "X"
		}
		return name
	}

	leader := sanitize(fmt.Sprintf("%s/%s", c.leaderLock, c.identity))
	id, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("error getting the default leader identity: %v", err)
	}

	recorder := record.NewBroadcaster()
	recorder.StartRecordingToSink(&corev1.EventSinkImpl{Interface: c.kubeClient.CoreV1().Events(ns)})
	eRecorder := recorder.NewRecorder(scheme.Scheme, v1.EventSource{Component: leader})

	rlConfig := resourcelock.ResourceLockConfig{
		Identity:      sanitize(id),
		EventRecorder: eRecorder,
	}

	l, err := resourcelock.New(resourcelock.LeasesResourceLock, ns, leader, c.kubeClient.CoreV1(), c.kubeClient.CoordinationV1(), rlConfig)
	if err != nil {
		return err
	}

	leaderConfig := leaderelection.LeaderElectionConfig{
		Lock:            l,
		ReleaseOnCancel: true,
		LeaseDuration:   c.LeaseDuration,
		RenewDeadline:   c.RenewDeadline,
		RetryPeriod:     c.RetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				glog.V(2).Info("became leader, starting")
				c.runController(ctx)
			},
			OnStoppedLeading: func() {
				glog.Fatal("stopped leading")
			},
			OnNewLeader: func(identity string) {
				glog.V(3).Infof("new leader detected, current leader: %s", identity)
			},
		},
	}

	leaderelection.RunOrDie(ctx, leaderConfig)
	return nil // should never reach here
}

func (c *DirectCSIController) runWorker(ctx context.Context) {
	for c.processNextItem(ctx) {
	}
}

func (c *DirectCSIController) processNextItem(ctx context.Context) bool {
	// Wait until there is a new item in the working queue
	uuidInterface, quit := c.queue.Get()
	if quit {
		return false
	}

	uuid := uuidInterface.(types.UID)
	defer c.queue.Done(uuid)

	op, ok := c.opMap.Load(uuid)
	if !ok {
		panic("unreachable code")
	}

	// Ensure that multiple operations on different versions of the same object
	// do not happen in parallel
	c.OpLock(uuid)
	defer c.OpUnlock(uuid)

	var err error

	switch o := op.(type) {
	case addOp:
		add := *o.AddFunc
		objMeta := o.Object.(metav1.Object)
		name := objMeta.GetName()
		err = add(ctx, o.Object)
		if err == nil {
			o.Indexer.Add(o.Object)
		} else {
			glog.Errorf("Error adding %s: %v", name, err)
		}
	case updateOp:
		update := *o.UpdateFunc
		objMeta := o.OldObject.(metav1.Object)
		name := objMeta.GetName()
		err = update(ctx, o.OldObject, o.NewObject)
		if err == nil {
			o.Indexer.Update(o.NewObject)
		} else {
			glog.Errorf("Error updating %s: %v", name, err)
		}
	case deleteOp:
		delete := *o.DeleteFunc
		objMeta := o.Object.(metav1.Object)
		name := objMeta.GetName()
		err = delete(ctx, o.Object)
		if err == nil {
			o.Indexer.Delete(o.Object)
		} else {
			glog.Errorf("Error deleting %s: %v", name, err)
		}
	default:
		panic("unknown item in queue")
	}

	// Handle the error if something went wrong
	c.handleErr(err, uuid)
	return true
}

func (c *DirectCSIController) OpLock(op types.UID) {
	c.GetOpLock(op).Lock()
}

func (c *DirectCSIController) OpUnlock(op types.UID) {
	c.GetOpLock(op).Unlock()
}

func (c *DirectCSIController) GetOpLock(op types.UID) *sync.Mutex {
	lockKey := op
	c.lockerLock.Lock()
	defer c.lockerLock.Unlock()

	if c.locker == nil {
		c.locker = map[types.UID]*sync.Mutex{}
	}

	if _, ok := c.locker[lockKey]; !ok {
		c.locker[lockKey] = &sync.Mutex{}
	}
	return c.locker[lockKey]
}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *DirectCSIController) handleErr(err error, uuid types.UID) {
	if err == nil {
		c.opMap.Delete(uuid)
		return
	}
	c.queue.AddRateLimited(uuid)
}

func (c *DirectCSIController) runController(ctx context.Context) {
	controllerFor := func(name string, objType runtime.Object, add addFunc, update updateFunc, delete deleteFunc) {
		indexer := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, cache.Indexers{})
		resyncPeriod := c.ResyncPeriod

		lw := cache.NewListWatchFromClient(c.directcsiClient.DirectV1beta1().RESTClient(), name, "", fields.Everything())
		cfg := &cache.Config{
			Queue: cache.NewDeltaFIFOWithOptions(cache.DeltaFIFOOptions{
				KnownObjects:          indexer,
				EmitDeltaTypeReplaced: false,
			}),
			ListerWatcher:    lw,
			ObjectType:       objType,
			FullResyncPeriod: resyncPeriod,
			RetryOnError:     true,
			Process: func(obj interface{}) error {
				for _, d := range obj.(cache.Deltas) {
					switch d.Type {
					case cache.Sync, cache.Replaced, cache.Added, cache.Updated:
						if old, exists, err := indexer.Get(d.Object); err == nil && exists {
							key, err := cache.MetaNamespaceKeyFunc(d.Object)
							if err != nil {
								panic(err)
							}

							if reflect.DeepEqual(d.Object, old) {
								return nil
							}

							uuid := d.Object.(metav1.Object).GetUID()

							c.opMap.Store(uuid, updateOp{
								OldObject:  old,
								NewObject:  d.Object,
								UpdateFunc: &update,
								Key:        key,
								Indexer:    indexer,
							})
							c.queue.Add(uuid)
						} else {
							key, err := cache.MetaNamespaceKeyFunc(d.Object)
							if err != nil {
								panic(err)
							}
							uuid := d.Object.(metav1.Object).GetUID()

							// If an update to the k8s object happens before add has succeeded
							if op, ok := c.opMap.Load(uuid); ok {
								if _, ok := op.(updateOp); ok {
									err := fmt.Errorf("cannot add already added object: %s", key)
									return err
								}
							}

							c.opMap.Store(uuid, addOp{
								Object:  d.Object,
								AddFunc: &add,
								Key:     key,
								Indexer: indexer,
							})
							c.queue.Add(uuid)
						}
					case cache.Deleted:
						key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(d.Object)
						if err != nil {
							panic(err)
						}

						uuid := d.Object.(metav1.Object).GetUID()
						c.opMap.Store(uuid, deleteOp{
							Object:     d.Object,
							DeleteFunc: &delete,
							Key:        key,
							Indexer:    indexer,
						})
						c.queue.Add(uuid)
					}
				}
				return nil
			},
		}
		ctrlr := cache.New(cfg)

		defer utilruntime.HandleCrash()
		defer c.queue.ShutDown()

		glog.V(3).Infof("Starting %s controller", name)
		go ctrlr.Run(ctx.Done())

		if !cache.WaitForCacheSync(ctx.Done(), ctrlr.HasSynced) {
			utilruntime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
			return
		}

		for i := 0; i < c.threadiness; i++ {
			go c.runWorker(ctx)
		}

		<-ctx.Done()
		glog.V(3).Infof("Stopping %s controller", name)
	}

	if c.DirectCSIVolumeListener != nil {
		c.DirectCSIVolumeListener.InitializeKubeClient(c.kubeClient)
		c.DirectCSIVolumeListener.InitializeDirectCSIClient(c.directcsiClient)
		addFunc := func(ctx context.Context, obj interface{}) error {
			return c.DirectCSIVolumeListener.Add(ctx, obj.(*v1beta1.DirectCSIVolume))
		}
		updateFunc := func(ctx context.Context, old interface{}, new interface{}) error {
			return c.DirectCSIVolumeListener.Update(ctx, old.(*v1beta1.DirectCSIVolume), new.(*v1beta1.DirectCSIVolume))
		}
		deleteFunc := func(ctx context.Context, obj interface{}) error {
			return c.DirectCSIVolumeListener.Delete(ctx, obj.(*v1beta1.DirectCSIVolume))
		}
		go controllerFor("DirectCSIVolumes", &v1beta1.DirectCSIVolume{}, addFunc, updateFunc, deleteFunc)
	}
	if c.DirectCSIDriveListener != nil {
		c.DirectCSIDriveListener.InitializeKubeClient(c.kubeClient)
		c.DirectCSIDriveListener.InitializeDirectCSIClient(c.directcsiClient)
		addFunc := func(ctx context.Context, obj interface{}) error {
			return c.DirectCSIDriveListener.Add(ctx, obj.(*v1beta1.DirectCSIDrive))
		}
		updateFunc := func(ctx context.Context, old interface{}, new interface{}) error {
			return c.DirectCSIDriveListener.Update(ctx, old.(*v1beta1.DirectCSIDrive), new.(*v1beta1.DirectCSIDrive))
		}
		deleteFunc := func(ctx context.Context, obj interface{}) error {
			return c.DirectCSIDriveListener.Delete(ctx, obj.(*v1beta1.DirectCSIDrive))
		}
		go controllerFor("DirectCSIDrives", &v1beta1.DirectCSIDrive{}, addFunc, updateFunc, deleteFunc)
	}

	<-ctx.Done()
}
