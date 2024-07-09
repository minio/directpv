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

// LogType represents the type of the log
type LogType int

const (
	// UnknownLogType denotes the dummy LogType
	UnknownLogType LogType = iota
	// ErrorLogType denotes the error LogType
	ErrorLogType
	// InfoLogType denotes the non-error info LogType
	InfoLogType
)

// LogMessage represents the log message
type LogMessage struct {
	Type             LogType
	Err              error
	Code             string
	Message          string
	Values           map[string]any
	FormattedMessage string
}

// LogFunc represents the logger
type LogFunc func(LogMessage)

func nullLogger(_ LogMessage) {}
