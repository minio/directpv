/*
 * This file is part of MinIO Direct CSI
 * Copyright (C) 2020, MinIO, Inc.
 *
 * This code is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License, version 3,
 * as published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License, version 3,
 * along with this program.  If not, see <http://www.gnu.org/licenses/>
 *
 */

package utils

import (
	"github.com/golang/glog"
)

type LogLevel int

const (
	LogLevelInvalid LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelDebug
)

var _debug = glog.V(3).Infof
var _warn = glog.V(2).Infof
var _info = glog.V(1).Infof

var _err = glog.Errorf
var _fatal = glog.Fatalf
