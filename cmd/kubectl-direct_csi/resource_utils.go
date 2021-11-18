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

package main

import (
	"context"
	"strings"

	"github.com/minio/direct-csi/pkg/client"
	"k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func getDrivesByIds(ctx context.Context, ids []string) <-chan client.ListDriveResult {
	resultCh := make(chan client.ListDriveResult)
	go func() {
		defer close(resultCh)
		directClient := client.GetDirectCSIClient()
		for _, id := range ids {
			driveName := strings.TrimSpace(id)
			d, err := directClient.DirectCSIDrives().Get(ctx, driveName, metav1.GetOptions{})
			if err != nil {
				if !errors.IsNotFound(err) {
					klog.ErrorS(err, "could not get drive", driveName)
					return
				}
				klog.Errorf("No resource of %s found by the name %s", bold("DirectCSIDrive"), driveName)
				continue
			}
			resultCh <- client.ListDriveResult{Drive: *d}
		}
	}()
	return resultCh
}
