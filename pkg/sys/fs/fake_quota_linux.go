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

package fs

import (
	"context"
	"fmt"
)

type FakeFSQuota struct {
	Path             string
	VolumeID         string
	BlockFile        string
	setProjectIDArgs struct {
		projectID uint32
	}
}

func (ffsq FakeFSQuota) GetBlockFile() string {
	return ffsq.BlockFile
}

func (ffsq FakeFSQuota) GetPath() string {
	return ffsq.Path
}

func (ffsq FakeFSQuota) GetVolumeID() string {
	return ffsq.VolumeID
}

func (ffsq *FakeFSQuota) SetQuota(ctx context.Context, limit int64) error {
	return SetFSQuota(ctx, ffsq, limit)
}

func (ffsq *FakeFSQuota) SetProjectID(projectID uint32) error {
	expectedProjectID := getProjectIDHash(ffsq.GetVolumeID())
	if projectID != expectedProjectID {
		return fmt.Errorf("Wrong argument passed to setProjectID. Expected projectID: %v, got: %v", expectedProjectID, projectID)
	}
	ffsq.setProjectIDArgs.projectID = projectID
	return nil
}

func (ffsq *FakeFSQuota) SetProjectQuota(maxBytes uint64, projID uint32) error {
	if maxBytes <= uint64(0) {
		return fmt.Errorf("Invalid argument passed for SetProjectQuota function: maxBytes: %v", maxBytes)
	}
	if projID != ffsq.setProjectIDArgs.projectID {
		return fmt.Errorf("Incorrect project id set. Expected: %v, got: %v", ffsq.setProjectIDArgs.projectID, projID)
	}
	return nil
}

func (ffsq *FakeFSQuota) GetQuota() (result *Dqblk, err error) {
	if ffsq.VolumeID == "" || ffsq.BlockFile == "" {
		return &Dqblk{}, fmt.Errorf("Invalid input for GetQuota function. VolumeID: %v, BlockFile: %v", ffsq.VolumeID, ffsq.BlockFile)
	}
	return &Dqblk{}, nil
}

func NewFakeQuotaer(targetPath, vID, blockFile string) Quotaer {
	return &FakeFSQuota{
		Path:      targetPath,
		VolumeID:  vID,
		BlockFile: blockFile,
	}
}
