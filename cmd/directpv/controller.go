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

package main

import (
	"context"
	"os"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/csi/controller"
	pkgidentity "github.com/minio/directpv/pkg/csi/identity"
	"github.com/minio/directpv/pkg/jobs"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"
)

var controllerCmd = &cobra.Command{
	Use:           consts.ControllerServerName,
	Short:         "Start controller server.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		return startController(c.Context())
	},
}

func startController(ctx context.Context) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	capabilities := pkgidentity.GetDefaultPluginCapabilities()
	capabilities = append(capabilities, &csi.PluginCapability{
		Type: &csi.PluginCapability_VolumeExpansion_{
			VolumeExpansion: &csi.PluginCapability_VolumeExpansion{
				Type: csi.PluginCapability_VolumeExpansion_ONLINE,
			},
		},
	})
	idServer, err := pkgidentity.NewServer(identity, Version, capabilities)
	if err != nil {
		return err
	}
	klog.V(3).Infof("Identity server started")

	ctrlServer := controller.NewServer()
	klog.V(3).Infof("Controller server started")

	errCh := make(chan error)

	go func() {
		if err := runServers(ctx, csiEndpoint, idServer, ctrlServer, nil); err != nil {
			klog.ErrorS(err, "unable to start GRPC servers")
			errCh <- err
		}
	}()

	go func() {
		if err := serveReadinessEndpoint(ctx); err != nil {
			klog.ErrorS(err, "unable to start readiness endpoint")
			errCh <- err
		}
	}()

	go func() {
		runJobsController(ctx)
	}()

	return <-errCh
}

func runJobsController(ctx context.Context) {
	podName := os.Getenv("HOSTNAME")
	if podName == "" {
		klog.V(5).Info("unable to get the pod name from env; defaulting to pod name: directpv-controller")
		podName = "directpv-controller"
	}
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      consts.AppName + "-jobs-controller",
			Namespace: consts.AppNamespace,
		},
		Client: k8s.KubeClient().CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: podName,
		},
	}
	// start the leader election code loop
	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   60 * time.Second,
		RenewDeadline:   15 * time.Second,
		RetryPeriod:     5 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				klog.Info("started leading")
				jobs.StartController(ctx)
			},
			OnStoppedLeading: func() {
				klog.Infof("leader lost")
			},
		},
	})
}
