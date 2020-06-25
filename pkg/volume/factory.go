package volume

import (
	"os"
	"path/filepath"
)

var vf = &vFactory{}

type vFactory struct {
	Paths []string
	LastAssigned int
}

func Initialize(paths []string) {
	vf.Paths = paths
}

func Provision(volumeID string) (string, error) {
	next := vf.LastAssigned + 1
	next = next % len(vf.Paths)

	nextPath := vf.Paths[next]
	
	if err := os.MkdirAll(filepath.Join(nextPath, volumeID), 0755); err != nil {
		return "", err
	}
	vf.LastAssigned = next

	return filepath.Join(nextPath, volumeID), nil
}
