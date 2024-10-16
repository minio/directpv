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

package installer

import (
	"context"
	"path"

	"github.com/minio/directpv/pkg/k8s"
	"k8s.io/klog/v2"
)

func newPluginsSocketDir(kubeletDir, name string) string {
	return path.Join(kubeletDir, "plugins", k8s.SanitizeResourceName(name))
}

func sendDoneMessage(ctx context.Context, progressCh chan<- Message, err error) (sent bool) {
	sent = sendMessage(ctx, progressCh, newDoneMessage(err))
	if !sent && err != nil {
		klog.Error(err)
	}
	return
}

func sendStartMessage(ctx context.Context, progressCh chan<- Message, totalSteps int) bool {
	return sendMessage(ctx, progressCh, newStartMessage(totalSteps))
}

func sendEndMessage(ctx context.Context, progressCh chan<- Message, err error) (sent bool) {
	sent = sendMessage(ctx, progressCh, newEndMessage(err, nil))
	if !sent && err != nil {
		klog.Error(err)
	}
	return
}

func sendProgressMessage(ctx context.Context, progressCh chan<- Message, message string, step int, component *Component) bool {
	return sendMessage(ctx, progressCh, newProgressMessage(message, step, component))
}

func sendLogMessage(ctx context.Context, progressCh chan<- Message, msg string) bool {
	return sendMessage(ctx, progressCh, newLogMessage(msg))
}

func namespaceComponent(name string) *Component {
	return &Component{
		Name: name,
		Kind: "Namespace",
	}
}

func serviceAccountComponent(name string) *Component {
	return &Component{
		Name: name,
		Kind: "ServiceAccount",
	}
}

func clusterRoleBindingComponent(name string) *Component {
	return &Component{
		Name: name,
		Kind: "ClusterRoleBinding",
	}
}

func clusterRoleComponent(name string) *Component {
	return &Component{
		Name: name,
		Kind: "ClusterRole",
	}
}

func roleBindingComponent(name string) *Component {
	return &Component{
		Name: name,
		Kind: "RoleBinding",
	}
}

func roleComponent(name string) *Component {
	return &Component{
		Name: name,
		Kind: "Role",
	}
}

func crdComponent(name string) *Component {
	return &Component{
		Name: name,
		Kind: "CustomResourceDefinition",
	}
}

func csiDriverComponent(name string) *Component {
	return &Component{
		Name: name,
		Kind: "CSIDriver",
	}
}

func storageClassComponent(name string) *Component {
	return &Component{
		Name: name,
		Kind: "StorageClass",
	}
}

func daemonsetComponent(name string) *Component {
	return &Component{
		Name: name,
		Kind: "Daemonset",
	}
}

func deploymentComponent(name string) *Component {
	return &Component{
		Name: name,
		Kind: "Deployment",
	}
}

func migrateLog(ctx context.Context, args *Args, errMsg string, showInProgress bool) error {
	switch {
	case args.ProgressCh != nil:
		if showInProgress {
			if !sendLogMessage(ctx, args.ProgressCh, errMsg) {
				return errSendProgress
			}
		}
	case !args.Quiet && !args.DryRun:
		klog.Error(errMsg)
	}

	if args.ObjectWriter != nil {
		_, err := args.ObjectWriter.Write([]byte(errMsg))
		return err
	}

	return nil
}
