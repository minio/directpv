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
	v1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

var (
	eventBroadcaster record.EventBroadcaster
	eventRecorder    record.EventRecorder
)

func initEvent(kubeClient kubernetes.Interface) {
	eventBroadcaster = record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(
		&corev1.EventSinkImpl{
			Interface: kubeClient.CoreV1().Events(""),
		},
	)
	eventRecorder = eventBroadcaster.NewRecorder(
		Scheme, v1.EventSource{Component: "directcsi-controller"},
	)
}

// Eventf raises kubernetes events.
func Eventf(object runtime.Object, eventType, reason, messageFmt string, args ...interface{}) {
	eventRecorder.Eventf(object, eventType, reason, messageFmt, args...)
}
