//go:build linux

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

package mount

type EventType string

const (
	Attached EventType = "attached"
	Detached EventType = "detached"
	Modified EventType = "modified"
)

type info struct {
	previousMountInfo map[string][]Info
	currentMountInfo  map[string][]Info
}

type Event struct {
	mountInfo Info
	eventType EventType
}

func (e *Event) MountInfo() Info {
	return e.mountInfo
}

func (e *Event) Type() EventType {
	return e.eventType
}
