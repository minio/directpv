// This file is part of MinIO Direct CSI
// Copyright (c) 2021 MinIO, Inc.
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

package node

import (
	"context"
	"errors"
	"time"

	"github.com/minio/directpv/pkg/mount"
	"github.com/minio/directpv/pkg/volume"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

func monitorDirectCSIMounts(ctx context.Context, nodeID string) error {
	backoff := &wait.Backoff{
		Steps:    4,
		Duration: 10 * time.Second,
		Factor:   5.0,
		Jitter:   0.1,
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		listener, err := mount.StartListener(ctx)
		if err == nil {
			var key string
			var event *mount.Event
			for {
				select {
				case <-ctx.Done():
					return nil
				default:
					key, event, err = listener.Get(ctx)
					if err == nil {
						// verify volume mounts
						volume.ProcessMountEvent(ctx, nodeID, key, event)
					}
				}
				if err != nil {
					break
				}
			}
		}

		if err != nil {
			klog.Error(err)
		}

		if listener != nil {
			listener.Close()
		}
		ticker.Reset(backoff.Step())
		select {
		case <-ctx.Done():
			return errors.New("canceled by context")
		case <-ticker.C:
		}
	}
}
