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
	"errors"
)

// MessageType denotes the type of message
type MessageType string

const (
	// StartMessageType denotes the start message
	StartMessageType MessageType = "start"
	// ProgressMessageType denotes the progress message
	ProgressMessageType MessageType = "progress"
	// EndMessageType denotes the end message
	EndMessageType MessageType = "end"
	// DoneMessageType denotes the done message
	DoneMessageType MessageType = "done"
	// LogMessageType denotes the log message
	LogMessageType MessageType = "log"
)

var errSendProgress = errors.New("unable to send message")

// Component denotes the component that is processed
type Component struct {
	Name string
	Kind string
}

// Message denotes the progress message
type Message struct {
	msgType   MessageType
	component *Component
	steps     int
	step      int
	message   string
	err       error
}

// Type returns the type of the message
func (m Message) Type() MessageType {
	return m.msgType
}

// StartMessage returns the start message content
func (m Message) StartMessage() (steps int) {
	return m.steps
}

// ProgressMessage returns the progress message content
func (m Message) ProgressMessage() (message string, step int, component *Component) {
	return m.message, m.step, m.component
}

// EndMessage returns the end message content
func (m Message) EndMessage() (component *Component, err error) {
	return m.component, m.err
}

// DoneMessage returns the done message content
func (m Message) DoneMessage() (err error) {
	return m.err
}

// LogMessage returns the log message content
func (m Message) LogMessage() string {
	return m.message
}

func newStartMessage(steps int) Message {
	return Message{
		msgType: StartMessageType,
		steps:   steps,
	}
}

func newProgressMessage(message string, step int, component *Component) Message {
	return Message{
		msgType:   ProgressMessageType,
		component: component,
		message:   message,
		step:      step,
	}
}

func newEndMessage(err error, component *Component) Message {
	return Message{
		msgType:   EndMessageType,
		component: component,
		err:       err,
	}
}

func newDoneMessage(err error) Message {
	return Message{
		msgType: DoneMessageType,
		err:     err,
	}
}

func newLogMessage(msg string) Message {
	return Message{
		msgType: LogMessageType,
		message: msg,
	}
}

func sendMessage(ctx context.Context, progressCh chan<- Message, message Message) bool {
	if progressCh == nil {
		return true
	}
	select {
	case <-ctx.Done():
		return false
	case progressCh <- message:
		return true
	}
}
