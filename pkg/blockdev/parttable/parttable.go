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

package parttable

import (
	"errors"
)

var (
	// ErrPartTableNotFound denotes partition table not found error.
	ErrPartTableNotFound = errors.New("partition table not found")

	// ErrCancelled denotes canceled by context error.
	ErrCancelled = errors.New("canceled by context")
)

// PartType denotes partition type.
type PartType int

// Partition types.
const (
	Primary PartType = iota + 1
	Extended
	Logical
)

func (pt PartType) String() string {
	switch pt {
	case Primary:
		return "primary"
	case Extended:
		return "extended"
	case Logical:
		return "logical"
	default:
		return ""
	}
}

// Partition denotes partition information.
type Partition struct {
	Number int
	UUID   string
	Type   PartType
}

// PartTable denotes partition table.
type PartTable interface {
	UUID() string
	Type() string
	Partitions() map[int]*Partition
}
