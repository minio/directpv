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

	directcsi "github.com/minio/directpv/pkg/legacy/apis/direct.csi.min.io/v1beta5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListDriveResult is for listing drive results
type ListDriveResult struct {
	Drive directcsi.DirectCSIDrive
	Err   error
}

// ListDrives is for listing drives
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
			result, err := directCSIDriveClient.List(ctx, options)
			if err != nil {
				send(ListDriveResult{Err: err})
				return
			}

			for _, item := range result.Items {
				switch item.Status.DriveStatus {
				case directcsi.DriveStatusReady, directcsi.DriveStatusInUse:
					if !send(ListDriveResult{Drive: item}) {
						return
					}
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

// ListVolumeResult is for list volumes results
type ListVolumeResult struct {
	Volume directcsi.DirectCSIVolume
	Err    error
}

// ListVolumes is for listing volumes
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
			result, err := directCSIVolumeClient.List(ctx, options)
			if err != nil {
				send(ListVolumeResult{Err: err})
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
