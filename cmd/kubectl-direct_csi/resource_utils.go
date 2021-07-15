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
	"fmt"
	"strings"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	"github.com/minio/direct-csi/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func getDrives(ctx context.Context, nodes []string, drives []string, accessTiers []string) <-chan directcsi.DirectCSIDrive {
	driveCh := make(chan directcsi.DirectCSIDrive)

	labelSelector := func() string {
		selector := ""

		drivesSelector := ""
		drivesKey := utils.DrivePathLabel

		drivesValue := strings.Join(utils.FmapStringSlice(drives, utils.SanitizeDrivePath), ",")
		if len(drives) > 0 {
			drivesSelector = fmt.Sprintf("%s in (%s)", drivesKey, drivesValue)
			selector = drivesSelector
		}

		nodesSelector := ""
		nodesKey := utils.NodeLabel
		nodesValue := strings.Join(utils.FmapStringSlice(nodes, utils.SanitizeLabelV), ",")
		if len(nodes) > 0 {
			nodesSelector = fmt.Sprintf("%s in (%s)", nodesKey, nodesValue)
			if selector != "" {
				selector = selector + ","
			}
			selector = selector + nodesSelector
		}

		accessTiersSelector := ""
		accessTiersKey := utils.AccessTierLabel
		accessTiersValue := strings.Join(utils.FmapStringSlice(accessTiers, utils.SanitizeLabelV), ",")
		if len(accessTiers) > 0 {
			accessTiersSelector = fmt.Sprintf("%s in (%s)", accessTiersKey, accessTiersValue)
			if selector != "" {
				selector = selector + ","
			}
			selector = selector + accessTiersSelector
		}

		return selector
	}()
	go func() {
		defer close(driveCh)
		cont := ""
		klog.V(5).InfoS("Listing DirectCSIDrives", "limit", utils.MaxThreadCount, "selectors", labelSelector)

		directClient := utils.GetDirectCSIClient()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				drives, err := directClient.DirectCSIDrives().List(ctx, metav1.ListOptions{
					Limit:         int64(utils.MaxThreadCount),
					LabelSelector: labelSelector,
					Continue:      cont,
				})
				if err != nil {
					klog.ErrorS(err, "could not list drives", "selectors", labelSelector)
					return
				}
				for _, d := range drives.Items {
					driveCh <- d
				}

				if drives.Continue == "" {
					return
				}
				cont = drives.Continue
			}
		}
	}()
	return driveCh
}

func getDrivesByIds(ctx context.Context, ids []string) <-chan directcsi.DirectCSIDrive {
	driveCh := make(chan directcsi.DirectCSIDrive)
	go func() {
		defer close(driveCh)
		directClient := utils.GetDirectCSIClient()
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
			driveCh <- *d
		}
	}()
	return driveCh
}
