// This file is part of MinIO Kubernetes Cloud
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

package volume

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/golang/glog"
)

var vf = &vFactory{}
var provisionerLock = &sync.Mutex{}

type vFactory struct {
	Paths        []string
	LastAssigned int
}

func InitializeFactory(paths []string) {
	vf.Paths = paths
	vf.LastAssigned = -1
}

func Provision(volumeID string) (string, error) {
	provisionerLock.Lock()
	defer provisionerLock.Unlock()

	if len(vf.Paths) == 0 {
		return "", fmt.Errorf("no base paths provided for directs")
	}
	next := vf.LastAssigned + 1
	next = next % len(vf.Paths)

	nextPath := vf.Paths[next]
	glog.V(15).Infof("[%s] using direct storage: BasePaths[%d] = %s", volumeID, next, nextPath)

	if err := os.MkdirAll(filepath.Join(nextPath, volumeID), 0755); err != nil {
		return "", err
	}
	vf.LastAssigned = next

	return filepath.Join(nextPath, volumeID), nil
}

func Unprovision(path string) error {
	return os.RemoveAll(path)
}
