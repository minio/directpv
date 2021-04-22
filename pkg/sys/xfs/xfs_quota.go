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

package xfs

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"

	"github.com/golang/glog"
	simd "github.com/minio/sha256-simd"
)

var (
	ErrProjNotFound = errors.New("xfs project not found")
)

type XFSQuota struct {
	Path      string
	ProjectID string
}

type XFSVolumeStats struct {
	AvailableBytes int64
	TotalBytes     int64
	UsedBytes      int64
}

func getProjectIDHash(id string) string {
	h := simd.Sum256([]byte(id))
	b := binary.LittleEndian.Uint32(h[:8])
	return strconv.FormatUint(uint64(b), 10)
}

// SetQuota creates a projectID and sets the hardlimit for the path
func (xfsq *XFSQuota) SetQuota(ctx context.Context, limit int64) error {

	_, err := xfsq.GetVolumeStats(ctx)
	// error getting quota value
	if err != nil && err != ErrProjNotFound {
		return err
	}
	// this means quota has already been set
	if err == nil {
		return nil
	}

	limitInStr := strconv.FormatInt(limit, 10)
	pid := getProjectIDHash(xfsq.ProjectID)

	glog.V(3).Infof("setting prjquota proj_id=%s path=%s", pid, xfsq.Path)

	cmd := exec.CommandContext(ctx, "xfs_quota", "-x", "-c", fmt.Sprintf("project -d 0 -s -p %s %s", xfsq.Path, pid))
	out, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("could not set prjquota proj_id=%s path=%s err=%v", pid, xfsq.Path, err)
		return fmt.Errorf("SetQuota failed for %s with error: (%v), output: (%s)", xfsq.ProjectID, err, out)
	}

	cmd = exec.CommandContext(ctx, "xfs_quota", "-x", "-c", fmt.Sprintf("limit -p bhard=%s %s", limitInStr, pid), xfsq.Path)
	out, err = cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("could not set prjquota proj_id=%s path=%s err=%v", pid, xfsq.Path, err)
		return fmt.Errorf("xfs_quota failed with error: %v, output: %s", err, out)
	}
	glog.V(3).Infof("prjquota set successfully proj_id=%s path=%s", pid, xfsq.Path)

	return nil
}

func dehumanize(size string) (float64, error) {
	if size == "0" {
		return 0.0, nil
	}
	f, unit := size[:len(size)-1], size[len(size)-1:]
	num, err := strconv.ParseFloat(f, 64)
	if err != nil {
		return 0.0, err
	}

	suffixes := map[string]int{"": 0, "K": 1, "M": 2, "G": 3, "T": 4, "P": 5, "E": 6, "Z": 7}
	suffix := "B"
	var prefix string

	if strings.HasSuffix(unit, suffix) {
		prefix = unit[:len(unit)-1]
	} else {
		prefix = unit
	}
	prefix = strings.ToUpper(prefix)

	if s, ok := suffixes[prefix]; ok {
		value := num * math.Pow(1024.0, float64(s))
		return value, nil
	} else {
		return 0.0, fmt.Errorf("Unknown unit: '%s'", prefix)
	}
}

// GetVolumeStats - Reads the xfs_quota report
func (xfsq *XFSQuota) GetVolumeStats(ctx context.Context) (XFSVolumeStats, error) {
	cmd := exec.CommandContext(ctx, "xfs_quota", "-x", "-c", fmt.Sprint("report -h"), xfsq.Path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return XFSVolumeStats{}, fmt.Errorf("GetVolumeStats failed with error: %v, output: %s", err, out)
	}
	output := string(out)
	pid := getProjectIDHash(xfsq.ProjectID)
	return ParseQuotaList(output, pid)
}

// ParseQuotaList - Parses the quota output and extracts the volume stats
func ParseQuotaList(output, projectID string) (XFSVolumeStats, error) {
	var usedInBytes, totalInBytes int64
	var pErr error
	var f float64
	lines := strings.Split(output, "\n")
	prjFound := false
	for _, l := range lines {
		line := strings.TrimSpace(l)
		if len(line) == 0 {
			continue
		}
		if !strings.HasPrefix(line, "#"+projectID) {
			continue
		}
		prjFound = true

		splits := strings.Split(line, " ")
		var values []string
		for _, split := range splits {
			tSplit := strings.TrimSpace(split)
			if tSplit == "" {
				continue
			}
			values = append(values, tSplit)
		}

		if values[0] == "#"+projectID {
			f, pErr = dehumanize(values[1])
			if pErr != nil {
				return XFSVolumeStats{}, fmt.Errorf("Error while reading xfs limits: %v", pErr)
			}
			usedInBytes = int64(f)
			f, pErr = dehumanize(values[3])
			if pErr != nil {
				return XFSVolumeStats{}, fmt.Errorf("Error while reading xfs limits: %v", pErr)
			}
			totalInBytes = int64(f)
			break
		}
		break
	}

	if !prjFound {
		return XFSVolumeStats{}, ErrProjNotFound
	}
	return XFSVolumeStats{
		AvailableBytes: totalInBytes - usedInBytes,
		TotalBytes:     totalInBytes,
		UsedBytes:      usedInBytes,
	}, nil
}
