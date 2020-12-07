// This file is part of MinIO Direct CSI
// Copyright (c) 2020 MinIO, Inc.
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

package dev

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
)

type BlockDevice struct {
	Major   string
	Minor   string
	Devname string
	Devtype string
}

func FindDrives(ctx context.Context) ([]BlockDevice, error) {
	const head = "/sys/devices"
	drives := []BlockDevice{}

	if err := filepath.Walk(head, func(path string, info os.FileInfo, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if strings.HasPrefix(info.Name(), "loop") {
			return filepath.SkipDir
		}
		if info.Name() != "uevent" {
			return nil
		}
		drive, err := parseUevent(path)
		if err != nil {
			glog.V(10).Info(err)
			return nil
		}
		if drive.Devtype != "disk" {
			return nil
		}
		subsystem, err := subsystem(path)
		if err != nil {
			glog.V(10).Info(err)
			return nil
		}
		if subsystem != "block" {
			return nil
		}
		drives = append(drives, drive)
		return nil
	}); err != nil {
		return nil, err
	}
	return drives, nil
}

func subsystem(path string) (string, error) {
	dir := filepath.Dir(path)
	link, err := os.Readlink(filepath.Join(dir, "subsystem"))
	if err != nil {
		return "", err
	}
	return filepath.Base(link), nil
}

func parseUevent(path string) (BlockDevice, error) {
	if filepath.Base(path) != "uevent" {
		return BlockDevice{}, fmt.Errorf("not a uevent file")
	}

	uevent, err := ioutil.ReadFile(path)
	if err != nil {
		return BlockDevice{}, err
	}

	uev := string(uevent)
	var major, minor, devname, devtype string

	lines := strings.Split(uev, "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		cleanLine := strings.TrimSpace(line)
		parts := strings.Split(cleanLine, "=")
		if len(parts) != 2 {
			return BlockDevice{}, fmt.Errorf("uevent file format not supported: %s", path)
		}
		key := parts[0]
		value := parts[1]
		switch key {
		case "MAJOR":
			major = value
		case "MINOR":
			minor = value
		case "DEVNAME":
			devname = value
		case "DEVTYPE":
			devtype = value
		default:
			return BlockDevice{}, fmt.Errorf("uevent file format not supported: %s", path)
		}
	}
	return BlockDevice{
		Major:   major,
		Minor:   minor,
		Devname: devname,
		Devtype: devtype,
	}, nil
}
