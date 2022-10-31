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

package client

import (
	"github.com/minio/directpv/pkg/consts"
	apicorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

// EventType denotes kubernetes event type.
type EventType string

// Enum values of EventType type.
const (
	EventTypeNormal  EventType = EventType(apicorev1.EventTypeNormal)
	EventTypeWarning EventType = EventType(apicorev1.EventTypeWarning)
)

// EventReason denotes kubernetes event reason.
type EventReason string

// Enum values of EventReason type.
const (
	EventReasonStageVolume             EventReason = "StageVolume"
	EventReasonVolumeMoved             EventReason = "VolumeMoved"
	EventReasonMetrics                 EventReason = "Metrics"
	EventReasonVolumeProvisioned       EventReason = "VolumeProvisioned"
	EventReasonVolumeAdded             EventReason = "VolumeAdded"
	EventReasonVolumeReleased          EventReason = "VolumeReleased"
	EventReasonDriveMountError         EventReason = "DriveHasMountError"
	EventReasonDriveMounted            EventReason = "DriveMounted"
	EventReasonDriveHasMultipleMatches EventReason = "DriveHasMultipleMatches"
	EventReasonDriveIOError            EventReason = "DriveHasIOError"
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
		Scheme, apicorev1.EventSource{Component: consts.ControllerName},
	)
}

// Eventf raises kubernetes events.
func Eventf(object runtime.Object, eventType EventType, reason EventReason, messageFmt string, args ...interface{}) {
	eventRecorder.Eventf(object, string(eventType), string(reason), messageFmt, args...)
}
