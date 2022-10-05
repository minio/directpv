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

package admin

import (
	"net/http"

	"k8s.io/klog/v2"
)

// readinessHandler - Checks if the process is up. Always returns success.
func readinessHandler(w http.ResponseWriter, r *http.Request) {
	klog.V(5).Infof("Received readiness request %v", r)
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}
