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
	"encoding/json"
	"testing"
)

func TestFindDrives(t *testing.T) {
	ctx := context.Background()

	drives, err := FindDrives(ctx)
	if err != nil {
		t.Fatal(err)
	}
	drivesJSON, _ := json.MarshalIndent(drives, "", " ")
	t.Error(string(drivesJSON))

	for _, d := range drives {
		parts, err := d.FindPartitions()
		if err != nil {
			t.Fatal(err)
		}
		partsJSON, _ := json.MarshalIndent(parts, "", " ")
		t.Error(string(partsJSON))
	}
}
