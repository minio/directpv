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

// Label represents the label to be set
type Label struct {
	Key    directpvtypes.LabelKey
	Value  directpvtypes.LabelValue
	Remove bool
}

func (l Label) String() string {
	if l.Value == "" {
		return string(l.Key)
	}
	return string(l.Key) + ":" + string(l.Value)
}

// LabelDriveArgs represents the arguments for adding/removing labels on/from the drives
type LabelDriveArgs struct {
	Nodes          []string
	Drives         []string
	DriveStatus    []directpvtypes.DriveStatus
	DriveIDs       []directpvtypes.DriveID
	LabelSelectors map[directpvtypes.LabelKey]directpvtypes.LabelValue
	Quiet          bool
	DryRun         bool
}

// LabelDrives sets/removes labels on/from the drives
func LabelDrives(ctx context.Context, args LabelDriveArgs, labels []Label) error {
	var processed bool
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh := client.NewDriveLister().
		NodeSelector(utils.ToLabelValues(args.Nodes)).
		DriveNameSelector(utils.ToLabelValues(args.Drives)).
		StatusSelector(args.DriveStatus).
		DriveIDSelector(args.DriveIDs).
		LabelSelector(args.LabelSelectors).
		List(ctx)
	for result := range resultCh {
		if result.Err != nil {
			return result.Err
		}
		processed = true
		drive := &result.Drive
		var verb string
		for i := range labels {
			updateFunc := func() (err error) {
				if labels[i].Remove {
					if ok := drive.RemoveLabel(labels[i].Key); !ok {
						return
					}
					verb = "removed from"
				} else {
					if ok := drive.SetLabel(labels[i].Key, labels[i].Value); !ok {
						return
					}
					verb = "set on"
				}
				if !args.DryRun {
					drive, err = client.DriveClient().Update(ctx, drive, metav1.UpdateOptions{})
				}
				if err != nil {
					utils.Eprintf(args.Quiet, true, "%v/%v: %v\n", drive.GetNodeID(), drive.GetDriveName(), err)
				} else if !args.Quiet {
					fmt.Printf("Label '%s' successfully %s %v/%v\n", labels[i].String(), verb, drive.GetNodeID(), drive.GetDriveName())
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
