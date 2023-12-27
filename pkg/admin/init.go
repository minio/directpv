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
	"errors"
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

var defaultListTimeout = 2 * time.Minute

// ErrNoDrivesAvailableToInit denotes that no drives are selected to initialize
var ErrNoDrivesAvailableToInit = errors.New("no drives are provided to init")

// InitResult denotes the status after initialization attempt
type InitResult struct {
	RequestID string
	NodeID    directpvtypes.NodeID
	Failed    bool
	Devices   []types.InitDeviceResult
}

// InitDevicesArgs represents the arguments used for initializing devices
type InitDevicesArgs struct {
	InitConfig    *InitConfig
	PrintProgress bool
	ListTimeout   time.Duration
}

// Validate the args
func (args *InitDevicesArgs) Validate() error {
	if args.InitConfig == nil {
		return errors.New("initconfig is not provided")
	}
	if args.ListTimeout == 0 {
		args.ListTimeout = defaultListTimeout
	}
	return nil
}

// InitDevices creates InitRequest objects and waits until it gets initialized
func InitDevices(ctx context.Context, args InitDevicesArgs) ([]InitResult, error) {
	if err := args.Validate(); err != nil {
		return nil, fmt.Errorf("unable to validate args; %v", err)
	}
	initRequests, requestID := args.InitConfig.ToInitRequestObjects()
	if len(initRequests) == 0 {
		return nil, ErrNoDrivesAvailableToInit
	}
	defer func() {
		labelMap := map[directpvtypes.LabelKey][]directpvtypes.LabelValue{
			directpvtypes.RequestIDLabelKey: utils.ToLabelValues([]string{requestID}),
		}
		client.InitRequestClient().DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
			LabelSelector: directpvtypes.ToLabelSelector(labelMap),
		})
	}()
	var teaProgram *tea.Program
	var wg sync.WaitGroup
	if args.PrintProgress {
		m := newProgressModel(true)
		teaProgram = tea.NewProgram(m)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := teaProgram.Run(); err != nil {
				fmt.Println("error running program:", err)
				return
			}
		}()
	}
	results, err := initDevices(ctx, initRequests, requestID, teaProgram, args.ListTimeout)
	if err != nil && teaProgram == nil {
		return nil, err
	}
	if teaProgram != nil {
		teaProgram.Send(progressNotification{
			done: true,
			err:  err,
		})
		wg.Wait()
	}
	return results, nil
}

func initDevices(ctx context.Context, initRequests []types.InitRequest, requestID string, teaProgram *tea.Program, listTimeout time.Duration) (results []InitResult, err error) {
	totalReqCount := len(initRequests)
	totalTasks := totalReqCount * 2
	var completedTasks int
	initProgressMap := make(map[string]progressLog, totalReqCount)
	for i := range initRequests {
		initReq, err := client.InitRequestClient().Create(ctx, &initRequests[i], metav1.CreateOptions{TypeMeta: types.NewInitRequestTypeMeta()})
		if err != nil {
			return nil, err
		}
		if teaProgram != nil {
			completedTasks++
			initProgressMap[initReq.Name] = progressLog{
				log: fmt.Sprintf("Processing initialization request '%s' for node '%v'", initReq.Name, initReq.GetNodeID()),
			}
			teaProgram.Send(progressNotification{
				progressLogs: toProgressLogs(initProgressMap),
				percent:      float64(completedTasks) / float64(totalTasks),
			})
		}
	}
	ctx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()

	eventCh, stop, err := client.NewInitRequestLister().
		RequestIDSelector(utils.ToLabelValues([]string{requestID})).
		Watch(ctx)
	if err != nil {
		return nil, err
	}
	defer stop()

	results = []InitResult{}
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
				initReq := event.Item
				if initReq.Status.Status != directpvtypes.InitStatusPending {
					results = append(results, InitResult{
						RequestID: initReq.Name,
						NodeID:    initReq.GetNodeID(),
						Devices:   initReq.Status.Results,
						Failed:    initReq.Status.Status == directpvtypes.InitStatusError,
					})
					if teaProgram != nil {
						completedTasks++
						initProgressMap[initReq.Name] = progressLog{
							log:  fmt.Sprintf("Processed initialization request '%s' for node '%v'", initReq.Name, initReq.GetNodeID()),
							done: true,
						}
						teaProgram.Send(progressNotification{
							progressLogs: toProgressLogs(initProgressMap),
							percent:      float64(completedTasks) / float64(totalTasks),
						})
					}
				}
				if len(results) >= totalReqCount {
					return
				}
			case watch.Deleted:
				return
			default:
			}
		case <-ctx.Done():
			err = fmt.Errorf("unable to initialize devices; %v", ctx.Err())
			return
		}
	}
}
