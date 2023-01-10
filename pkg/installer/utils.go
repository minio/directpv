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
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/minio/directpv/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

func mustGetYAML(i interface{}) string {
	data, err := yaml.Marshal(i)
	if err != nil {
		klog.Fatalf("unable to marshal object to YAML; %w", err)
	}
	return fmt.Sprintf("%v\n---\n", string(data))
}

func newHostPathVolume(name, path string) corev1.Volume {
	hostPathType := corev1.HostPathDirectoryOrCreate
	volumeSource := corev1.VolumeSource{
		HostPath: &corev1.HostPathVolumeSource{
			Path: path,
			Type: &hostPathType,
		},
	}

	return corev1.Volume{
		Name:         name,
		VolumeSource: volumeSource,
	}
}

func newPluginsSocketDir(kubeletDir, name string) string {
	return path.Join(kubeletDir, "plugins", k8s.SanitizeResourceName(name))
}

func newVolumeMount(name, path string, mountPropogation corev1.MountPropagationMode, readOnly bool) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:             name,
		ReadOnly:         readOnly,
		MountPath:        path,
		MountPropagation: &mountPropogation,
	}
}

func getRandSuffix() string {
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		klog.Fatalf("unable to generate random bytes; %v", err)
	}
	return strings.ToLower(base32.StdEncoding.EncodeToString(b)[:5])
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

func podSecurityPolicyComponent(name string) *Component {
	return &Component{
		Name: name,
		Kind: "PodSecurityPolicy",
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
	return writeToAuditFile(args.auditWriter, errMsg)
}

func writeToAuditFile(writer io.Writer, message string) error {
	if writer == nil {
		return nil
	}
	log := fmt.Sprintf("\n%s\n---\n", message)
	_, err := io.WriteString(writer, log)
	return err
}
