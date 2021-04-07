// +build !linux

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

package loopback

import "errors"

// dummy stubs kept for compilation purposes

var errNotALoopDevice = errors.New("Not a loop device")
var backFileSize = 100 * oneMB

type LoopInfo struct {
	Device         uint64
	INode          uint64
	RDevice        uint64
	Offset         uint64
	SizeLimit      uint64
	Number         uint32
	EncryptType    uint32
	EncryptKeySize uint32
	Flags          uint32
	FileName       [NameSize]byte
	CryptName      [NameSize]byte
	EncryptKey     [KeySize]byte
	Init           [2]uint64
}

func getInfo(fd uintptr) (LoopInfo, error) {
	return LoopInfo{}, errNotALoopDevice
}

func RemoveLoopDevice(loopPath string) error {
	return errNotALoopDevice
}

func CreateLoopbackDevice() (string, error) {
	return "", errNotALoopDevice
}
