// This file is part of MinIO DirectPV
// Copyright (c) 2024 MinIO, Inc.
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

package admin

import (
	"context"
	"fmt"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

// LabelVolumeArgs represents the arguments for adding/removing labels on/from the volumes
type LabelVolumeArgs struct {
	Nodes          []string
	Drives         []string
	DriveIDs       []string
	PodNames       []string
	PodNamespaces  []string
	VolumeStatus   []directpvtypes.VolumeStatus
	VolumeNames    []string
	LabelSelectors map[directpvtypes.LabelKey]directpvtypes.LabelValue
	Quiet          bool
	DryRun         bool
}

// LabelVolumes sets/removes labels on/from the volumes
func LabelVolumes(ctx context.Context, args LabelVolumeArgs, labels []Label) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	var processed bool
	resultCh := client.NewVolumeLister().
		NodeSelector(utils.ToLabelValues(args.Nodes)).
		DriveNameSelector(utils.ToLabelValues(args.Drives)).
		DriveIDSelector(utils.ToLabelValues(args.DriveIDs)).
		PodNameSelector(utils.ToLabelValues(args.PodNames)).
		PodNSSelector(utils.ToLabelValues(args.PodNamespaces)).
		StatusSelector(args.VolumeStatus).
		VolumeNameSelector(args.VolumeNames).
		LabelSelector(args.LabelSelectors).
		List(ctx)
	for result := range resultCh {
		if result.Err != nil {
			return result.Err
		}
		var verb string
		processed = true
		volume := &result.Volume
		for i := range labels {
			updateFunc := func() (err error) {
				if labels[i].Remove {
					if ok := volume.RemoveLabel(labels[i].Key); !ok {
						return
					}
					verb = "removed from"
				} else {
					if ok := volume.SetLabel(labels[i].Key, labels[i].Value); !ok {
						return
					}
					verb = "set on"
				}
				if !args.DryRun {
					volume, err = client.VolumeClient().Update(ctx, volume, metav1.UpdateOptions{})
				}
				if err != nil {
					utils.Eprintf(args.Quiet, true, "%v: %v\n", volume.Name, err)
				} else if !args.Quiet {
					fmt.Printf("Label '%s' successfully %s %v\n", labels[i].String(), verb, volume.Name)
				}
				return
			}
			retry.RetryOnConflict(retry.DefaultRetry, updateFunc)
		}
	}
	if !processed {
		return ErrNoMatchingResourcesFound
	}
	return nil
}
