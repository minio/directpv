// This file is part of MinIO Kubernetes Cloud
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

package centralcontroller

import (
	"context"
	"fmt"
	"os"
	"time"

	directcsiv1alpha1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	"github.com/minio/direct-csi/pkg/util"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"

	"github.com/golang/glog"
)

type Controller struct {
	Identity string

	// Leader election
	LeaderLock string

	LeaseDuration time.Duration
	RenewDeadline time.Duration
	RetryPeriod   time.Duration

	// Controller
	ResyncPeriod time.Duration
}

func (c *Controller) Run(ctx context.Context) error {
	ns := util.GetNamespace()
	lock := util.Sanitize(c.LeaderLock)
	id, err := os.Hostname()
	if err != nil {
		return err
	}
	kClient := util.GetKubeClientOrDie()

	// Create the DirectCSI Namespace
	if err := createDirectCSINamespace(ctx, kClient, DirectCSINS); err != nil {
		return err
	}

	recorder := record.NewBroadcaster()
	recorder.StartRecordingToSink(&corev1.EventSinkImpl{Interface: kClient.CoreV1().Events(ns)})
	eRecorder := recorder.NewRecorder(scheme.Scheme, v1.EventSource{Component: fmt.Sprintf("%s/%s", id, lock)})

	rlConfig := resourcelock.ResourceLockConfig{
		Identity:      id,
		EventRecorder: eRecorder,
	}

	l, err := resourcelock.New(resourcelock.LeasesResourceLock, ns, lock, kClient.CoreV1(), kClient.CoordinationV1(), rlConfig)
	if err != nil {
		return err
	}

	leaderConfig := leaderelection.LeaderElectionConfig{
		Lock:          l,
		LeaseDuration: c.LeaseDuration,
		RenewDeadline: c.RenewDeadline,
		RetryPeriod:   c.RetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				glog.V(2).Info("became leader, starting")
				c.RunController(ctx)
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

func (c *Controller) RunController(ctx context.Context) {
	dClient := util.GetDirectCSIClientOrDie()

	indexer := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, cache.Indexers{})
	lw := cache.NewListWatchFromClient(dClient.DirectV1alpha1().RESTClient(), "StorageTopologies", "", fields.Everything())
	objType := &directcsiv1alpha1.StorageTopology{}
	resyncPeriod := c.ResyncPeriod

	fifo := cache.NewDeltaFIFOWithOptions(cache.DeltaFIFOOptions{
		KnownObjects:          indexer,
		EmitDeltaTypeReplaced: true,
	})

	cfg := &cache.Config{
		Queue:            fifo,
		ListerWatcher:    lw,
		ObjectType:       objType,
		FullResyncPeriod: resyncPeriod,
		RetryOnError:     true,
		Process: func(obj interface{}) error {
			for _, d := range obj.(cache.Deltas) {
				switch d.Type {
				case cache.Sync, cache.Replaced, cache.Added, cache.Updated:
					if old, exists, err := indexer.Get(d.Object); err == nil && exists {
						if err := indexer.Update(d.Object); err != nil {
							return err
						}
						return c.OnUpdate(ctx,
							old.(*directcsiv1alpha1.StorageTopology),
							d.Object.(*directcsiv1alpha1.StorageTopology))
					} else {
						if err := indexer.Add(d.Object); err != nil {
							return err
						}
						return c.OnAdd(ctx, d.Object.(*directcsiv1alpha1.StorageTopology))
					}
				case cache.Deleted:
					if err := indexer.Delete(d.Object); err != nil {
						return err
					}
					return c.OnDelete(ctx, d.Object.(*directcsiv1alpha1.StorageTopology))
				}
			}
			return nil
		},
	}
	ctrlr := cache.New(cfg)

	stopper := make(chan struct{})
	defer close(stopper)

	ctrlr.Run(stopper)
	<-stopper
}
