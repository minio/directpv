// This file is part of MinIO DirectPV
// Copyright (c) 2022 MinIO, Inc.
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

package client

import (
	"context"

	directv1beta5 "github.com/minio/directpv/pkg/legacy/apis/direct.csi.min.io/v1beta5"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListDriveResult denotes list of drive result.
type ListDriveResult struct {
	Drive directv1beta5.DirectCSIDrive
	Err   error
}

// ListDrives returns channel to loop through drive items.
func ListDrives(ctx context.Context) <-chan ListDriveResult {
	resultCh := make(chan ListDriveResult)
	go func() {
		defer close(resultCh)

		send := func(result ListDriveResult) bool {
			select {
			case <-ctx.Done():
				return false
			case resultCh <- result:
				return true
			}
		}

		options := metav1.ListOptions{Limit: 1000}
		for {
			result, err := DriveClient().List(ctx, options)
			if err != nil {
				if !apierrors.IsNotFound(err) {
					send(ListDriveResult{Err: err})
				}
				return
			}

			for _, item := range result.Items {
				if !send(ListDriveResult{Drive: item}) {
					return
				}
			}

			if result.Continue == "" {
				return
			}

			options.Continue = result.Continue
		}
	}()

	return resultCh
}

// ListVolumeResult denotes list of volume result.
type ListVolumeResult struct {
	Volume directv1beta5.DirectCSIVolume
	Err    error
}

// ListVolumes returns channel to loop through volume items.
func ListVolumes(ctx context.Context) <-chan ListVolumeResult {
	resultCh := make(chan ListVolumeResult)
	go func() {
		defer close(resultCh)

		send := func(result ListVolumeResult) bool {
			select {
			case <-ctx.Done():
				return false
			case resultCh <- result:
				return true
			}
		}

		options := metav1.ListOptions{Limit: 1000}
		for {
			result, err := VolumeClient().List(ctx, options)
			if err != nil {
				if !apierrors.IsNotFound(err) {
					send(ListVolumeResult{Err: err})
				}
				return
			}

			for _, item := range result.Items {
				if !send(ListVolumeResult{Volume: item}) {
					return
				}
			}

			if result.Continue == "" {
				return
			}

			options.Continue = result.Continue
		}
	}()

	return resultCh
}
