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
	"testing"
)

func TestParseQuotaList(t1 *testing.T) {
	output := `Project quota on /tmp/c333 (/dev/xvdc)
					                   Blocks              
			   Project ID   Used   Soft   Hard Warn/Grace   
			   ---------- --------------------------------- 
			   #0              0      0      0  00 [------]
			   #100            0      0     8T  00 [------]
			   #101            0      0    10M  00 [------]
			   #200           4K      0  20.4M  00 [------]`

	testCases := []struct {
		name      string
		projectID string
	}{
		{
			name:      "test1",
			projectID: "100",
		},
		{
			name:      "test2",
			projectID: "101",
		},
		{
			name:      "test3",
			projectID: "200",
		},
	}

	for _, tt := range testCases {
		t1.Run(tt.name, func(t1 *testing.T) {
			_, err := ParseQuotaList(output, tt.projectID)
			if err != nil {
				t1.Error(err)
			}
		})
	}

}
