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

package utils

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"

	runtime "k8s.io/apimachinery/pkg/runtime"

	"k8s.io/klog/v2"
)

type objectResult struct {
	object runtime.Object
	err    error
}

func processObjects(
	ctx context.Context,
	resultCh <-chan objectResult,
	matchFunc func(runtime.Object) bool,
	applyFunc func(runtime.Object) error,
	processFunc func(context.Context, runtime.Object) error,
	writer io.Writer,
	dryRun bool,
) error {
	stopCh := make(chan struct{})
	var stopChMu int32
	closeStopCh := func() {
		if atomic.AddInt32(&stopChMu, 1) == 1 {
			close(stopCh)
		}
	}
	defer closeStopCh()

	objectCh := make(chan runtime.Object)
	var wg sync.WaitGroup

	// Start utils.MaxThreadCount workers.
	var errs []error
	for i := 0; i < MaxThreadCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-stopCh:
					return
				case object, ok := <-objectCh:
					if !ok {
						return
					}
					if err := processFunc(ctx, object); err != nil {
						errs = append(errs, err)
						defer closeStopCh()
						return
					}
				}
			}
		}()
	}

	var err error
	for result := range resultCh {
		if result.err != nil {
			err = result.err
			break
		}

		if !matchFunc(result.object) {
			continue
		}

		if err = applyFunc(result.object); err != nil {
			break
		}

		if dryRun {
			if err := LogYAML(result.object); err != nil {
				klog.Errorf("Unable to convert to YAML. %v", err)
			}
			continue
		}
		if err := WriteObject(writer, result.object); err != nil {
			return err
		}

		breakLoop := false
		select {
		case <-ctx.Done():
			breakLoop = true
		case <-stopCh:
			breakLoop = true
		case objectCh <- result.object:
		}

		if breakLoop {
			break
		}
	}

	close(objectCh)
	wg.Wait()

	if err != nil {
		return err
	}

	msgs := []string{}
	for _, err := range errs {
		msgs = append(msgs, err.Error())
	}
	if msg := strings.Join(msgs, "; "); msg != "" {
		return errors.New(msg)
	}

	return nil
}

func ProcessVolumes(
	ctx context.Context,
	resultCh <-chan ListVolumeResult,
	matchFunc func(*directcsi.DirectCSIVolume) bool,
	applyFunc func(*directcsi.DirectCSIVolume) error,
	processFunc func(context.Context, *directcsi.DirectCSIVolume) error,
	writer io.Writer,
	dryRun bool,
) error {
	objectCh := make(chan objectResult)
	go func() {
		defer close(objectCh)
		for result := range resultCh {
			var oresult objectResult
			if result.Err != nil {
				oresult.err = result.Err
			} else {
				volume := result.Volume
				oresult.object = &volume
			}

			select {
			case <-ctx.Done():
				return
			case objectCh <- oresult:
			}
		}
	}()

	return processObjects(
		ctx,
		objectCh,
		func(object runtime.Object) bool {
			return matchFunc(object.(*directcsi.DirectCSIVolume))
		},
		func(object runtime.Object) error {
			return applyFunc(object.(*directcsi.DirectCSIVolume))
		},
		func(ctx context.Context, object runtime.Object) error {
			return processFunc(ctx, object.(*directcsi.DirectCSIVolume))
		},
		writer,
		dryRun,
	)
}

func ProcessDrives(
	ctx context.Context,
	resultCh <-chan ListDriveResult,
	matchFunc func(*directcsi.DirectCSIDrive) bool,
	applyFunc func(*directcsi.DirectCSIDrive) error,
	processFunc func(context.Context, *directcsi.DirectCSIDrive) error,
	writer io.Writer,
	dryRun bool,
) error {
	objectCh := make(chan objectResult)
	go func() {
		defer close(objectCh)
		for result := range resultCh {
			var oresult objectResult
			if result.Err != nil {
				oresult.err = result.Err
			} else {
				drive := result.Drive
				oresult.object = &drive
			}

			select {
			case <-ctx.Done():
				return
			case objectCh <- oresult:
			}
		}
	}()

	return processObjects(
		ctx,
		objectCh,
		func(object runtime.Object) bool {
			return matchFunc(object.(*directcsi.DirectCSIDrive))
		},
		func(object runtime.Object) error {
			return applyFunc(object.(*directcsi.DirectCSIDrive))
		},
		func(ctx context.Context, object runtime.Object) error {
			return processFunc(ctx, object.(*directcsi.DirectCSIDrive))
		},
		writer,
		dryRun,
	)
}
