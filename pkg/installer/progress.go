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

import "sync/atomic"

const (
	totalTasks = 5
)

var perTaskWeightage = 1.0 / totalTasks

// Task represents the task event.
type Task struct {
	TotalSteps     int
	StepsCompleted int
	Done           bool
}

func newTask(totalSteps, stepsCompleted int, done bool) *Task {
	return &Task{
		TotalSteps:     totalSteps,
		StepsCompleted: stepsCompleted,
		Done:           done,
	}
}

// ProgressPercent denotes the progress of task.
func (t Task) ProgressPercent() float64 {
	return float64(t.StepsCompleted) * t.perStepWeightage()
}

func (t Task) perStepWeightage() float64 {
	return perTaskWeightage / float64(t.TotalSteps)
}

// Event denotes the progress message event.
type Event struct {
	Message string
	Persist bool
	Err     error
	Task    *Task
}

// Progress keeps track of the progress information.
type Progress struct {
	TotalTasks int
	EventCh    chan Event
	Done       bool
	isClosed   int32
}

// Component indicates the components that are processed.
type Component struct {
	Name string
	Kind string
}

// NewProgress returns an instance of the progress.
func NewProgress() *Progress {
	return &Progress{
		EventCh:    make(chan Event),
		TotalTasks: totalTasks,
	}
}

// Close closes the notifyCh.
func (p *Progress) Close() {
	if atomic.AddInt32(&p.isClosed, 1) == 1 {
		close(p.EventCh)
	}
}
