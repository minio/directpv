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
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/retry"
)

const nodeListTimeout = 2 * time.Minute

// DiscoverArgs represets the arguments for discovery
type DiscoverArgs struct {
	Nodes         []string
	Drives        []string
	PrintProgress bool
}

// DiscoverDevices discovers and fetches the devices present in the cluster
func DiscoverDevices(ctx context.Context, args DiscoverArgs) (map[directpvtypes.NodeID][]types.Device, error) {
	if err := client.SyncNodes(ctx); err != nil {
		return nil, err
	}

	nodes, err := client.NewNodeLister().
		NodeSelector(utils.ToLabelValues(args.Nodes)).
		Get(ctx)
	if err != nil {
		return nil, err
	}
	if len(args.Nodes) != 0 && len(nodes) == 0 {
		suffix := ""
		if len(args.Nodes) > 1 {
			suffix = "s"
		}
		return nil, fmt.Errorf("node%v %v not found", suffix, strings.Join(args.Nodes, ","))
	}

	var teaProgram *tea.Program
	var wg sync.WaitGroup
	if args.PrintProgress {
		m := newProgressModel(false)
		teaProgram = tea.NewProgram(m)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := teaProgram.Run(); err != nil {
				fmt.Println("error running program:", err)
				os.Exit(1)
			}
		}()
	}
	resultMap, err := discoverDevices(ctx, nodes, args.Drives, teaProgram)
	if err != nil {
		return nil, err
	}
	if teaProgram != nil {
		teaProgram.Send(progressNotification{
			done: true,
			err:  err,
		})
		wg.Wait()
	}
	return resultMap, nil
}

func discoverDevices(ctx context.Context, nodes []types.Node, drives []string, teaProgram *tea.Program) (devices map[directpvtypes.NodeID][]types.Device, err error) {
	var nodeNames []string
	nodeClient := client.NodeClient()
	totalNodeCount := len(nodes)
	discoveryProgressMap := make(map[string]progressLog, totalNodeCount)
	for i := range nodes {
		nodeNames = append(nodeNames, nodes[i].Name)
		updateFunc := func() error {
			node, err := nodeClient.Get(ctx, nodes[i].Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			node.Spec.Refresh = true
			if _, err := nodeClient.Update(ctx, node, metav1.UpdateOptions{TypeMeta: types.NewNodeTypeMeta()}); err != nil {
				return err
			}
			if teaProgram != nil {
				discoveryProgressMap[node.Name] = progressLog{
					log: fmt.Sprintf("Discovering node '%v'", node.Name),
				}
				teaProgram.Send(progressNotification{
					progressLogs: toProgressLogs(discoveryProgressMap),
				})
			}
			return nil
		}
		if err = retry.RetryOnConflict(retry.DefaultRetry, updateFunc); err != nil {
			return nil, err
		}
	}

	ctx, cancel := context.WithTimeout(ctx, nodeListTimeout)
	defer cancel()

	eventCh, stop, err := client.NewNodeLister().
		NodeSelector(utils.ToLabelValues(nodeNames)).
		Watch(ctx)
	if err != nil {
		return nil, err
	}
	defer stop()

	devices = map[directpvtypes.NodeID][]types.Device{}
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			if event.Err != nil {
				err = event.Err
				return
			}
			switch event.Type {
			case watch.Modified, watch.Added:
				node := event.Item
				if !node.Spec.Refresh {
					devices[directpvtypes.NodeID(node.Name)] = node.GetDevicesByNames(drives)
					if teaProgram != nil {
						discoveryProgressMap[node.Name] = progressLog{
							log:  fmt.Sprintf("Discovered node '%v'", node.Name),
							done: true,
						}
						teaProgram.Send(progressNotification{
							progressLogs: toProgressLogs(discoveryProgressMap),
						})
					}
				}
				if len(devices) >= len(nodes) {
					return
				}
			case watch.Deleted:
				return
			default:
			}
		case <-ctx.Done():
			err = fmt.Errorf("unable to complete the discovery; %v", ctx.Err())
			return
		}
	}
}
