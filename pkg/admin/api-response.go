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
	"encoding/json"
	"net/http"
	"strconv"

	"k8s.io/klog/v2"
)

// writeSuccessResponseJSON writes success headers and response if any,
// with content-type set to `application/json`.
func writeSuccessResponse(w http.ResponseWriter, response []byte) {
	writeResponse(w, http.StatusOK, response)
}

// writeSuccessResponse writes error response with the provided statusCode
// with content-type set to `application/json`.
func writeErrorResponse(w http.ResponseWriter, statusCode int, apiErr apiError) {
	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}
	var responseBytes []byte
	var err error
	responseBytes, err = json.Marshal(apiErr)
	if err != nil {
		klog.Errorf("couldn't marshal the apierror %v: %v", apiErr, err)
		responseBytes = []byte(apiErr.Description)
	}
	writeResponse(w, statusCode, responseBytes)
}

// writeResponse writes the response bytes to the response writer
// with content-type set to `application/json`
func writeResponse(w http.ResponseWriter, statusCode int, response []byte) {
	if statusCode == 0 || statusCode < 100 || statusCode > 999 {
		klog.Errorf("invalid WriteHeader code %v", statusCode)
		statusCode = http.StatusInternalServerError
	}
	w.Header().Add("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(response)))
	w.WriteHeader(statusCode)
	if response != nil {
		w.Write(response)
	}
}
