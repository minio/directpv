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

// LabelDriveResult represents the labeled drives
type LabelDriveResult struct {
	NodeID    directpvtypes.NodeID
	DriveName directpvtypes.DriveName
}

// LabelDriveArgs represents the arguments for adding/removing labels on/from the drives
type LabelDriveArgs struct {
	Nodes          []string
	Drives         []string
	DriveStatus    []directpvtypes.DriveStatus
	DriveIDs       []directpvtypes.DriveID
	LabelSelectors map[directpvtypes.LabelKey]directpvtypes.LabelValue
	DryRun         bool
}

// LabelDrives sets/removes labels on/from the drives
func (client *Client) LabelDrives(ctx context.Context, args LabelDriveArgs, labels []Label, log LogFunc) (results []LabelDriveResult, err error) {
	if log == nil {
		log = nullLogger
	}

	for _, label := range labels {
		if label.Key.IsReserved() {
			action := "use"
			if label.Remove {
				action = "remove"
			}
			err = fmt.Errorf("cannot %v reserved key %v", action, label.Key)
			return
		}
	}

	var processed bool
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh := client.NewDriveLister().
		NodeSelector(directpvtypes.ToLabelValues(args.Nodes)).
		DriveNameSelector(directpvtypes.ToLabelValues(args.Drives)).
		StatusSelector(args.DriveStatus).
		DriveIDSelector(args.DriveIDs).
		LabelSelector(args.LabelSelectors).
		List(ctx)
	for result := range resultCh {
		if result.Err != nil {
			err = result.Err
			return
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
					drive, err = client.Drive().Update(ctx, drive, metav1.UpdateOptions{})
				}
				if err != nil {
					log(
						LogMessage{
							Type:             ErrorLogType,
							Err:              err,
							Message:          "unable to " + verb + " label to drive",
							Values:           map[string]any{"node": drive.GetNodeID(), "driveName": drive.GetDriveName()},
							FormattedMessage: fmt.Sprintf("%v/%v: %v\n", drive.GetNodeID(), drive.GetDriveName(), err),
						},
					)
				} else {
					log(
						LogMessage{
							Type:             InfoLogType,
							Message:          "label successfully " + verb + " label to drive",
							Values:           map[string]any{"label": labels[i].String(), "verb": verb, "node": drive.GetNodeID(), "driveName": drive.GetDriveName()},
							FormattedMessage: fmt.Sprintf("Label '%s' successfully %s %v/%v\n", labels[i].String(), verb, drive.GetNodeID(), drive.GetDriveName()),
						},
					)
				}
				results = append(results, LabelDriveResult{
					NodeID:    drive.GetNodeID(),
					DriveName: drive.GetDriveName(),
				})
				return
			}
			retry.RetryOnConflict(retry.DefaultRetry, updateFunc)
		}
	}
	if !processed {
		return nil, ErrNoMatchingResourcesFound
	}
	return
}
