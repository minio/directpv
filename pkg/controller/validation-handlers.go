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

package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	failureStatus = "Failure"
	rootPath      = "/"
	xfsFileSystem = "xfs"
)

type validationHandler struct {
}

func parseAdmissionReview(req *http.Request) (admissionv1.AdmissionReview, error) {
	var body []byte
	admissionReview := admissionv1.AdmissionReview{}

	if req.Method != http.MethodPost {
		return admissionReview, errors.New("invalid HTTP Method")
	}

	if req.Body != nil {
		if data, err := io.ReadAll(req.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		return admissionReview, errors.New("request body empty")
	}

	if err := json.Unmarshal(body, &admissionReview); err != nil {
		return admissionReview, err
	}

	return admissionReview, nil
}

func writeSuccessResponse(admissionReview admissionv1.AdmissionReview, w http.ResponseWriter) {
	resp, err := json.Marshal(admissionReview)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(resp); err != nil {
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
	w.(http.Flusher).Flush()
}

func validateRequestedFormat(directCSIDrive directcsi.DirectCSIDrive, admissionReview *admissionv1.AdmissionReview) bool {
	requestedFormat := directCSIDrive.Spec.RequestedFormat

	// Check if the `requestedFormat` field is set
	if requestedFormat == nil {
		return true
	}

	// Drive Status checks
	// (*) Do not allow updates on `Unavailable`/`InUse` drives
	validateDriveStatus := func() bool {
		driveStatus := directCSIDrive.Status.DriveStatus
		switch driveStatus {
		case directcsi.DriveStatusUnavailable:
			admissionReview.Response.Allowed = false
			admissionReview.Response.Result = &metav1.Status{
				Status:  failureStatus,
				Message: "Unavailable drives cannot be added/formatted",
			}
			return false
		case directcsi.DriveStatusInUse:
			admissionReview.Response.Allowed = false
			admissionReview.Response.Result = &metav1.Status{
				Status:  failureStatus,
				Message: "Drives in-use cannot be formatted and added",
			}
			return false
		default:
			return true
		}
	}
	if !validateDriveStatus() {
		return false
	}

	// Mountpoint checks
	// (*) Do not allow updates on root partitions
	// (*) Check if `force` flag is set for unmounting
	validateMountpoint := func() bool {
		mountPoint := directCSIDrive.Status.Mountpoint
		switch mountPoint {
		case "":
			return true
		case rootPath:
			admissionReview.Response.Allowed = false
			admissionReview.Response.Result = &metav1.Status{
				Status:  failureStatus,
				Message: "Root partition'ed drives cannot be added/formatted",
			}
			return false
		default:
			if !requestedFormat.Force {
				admissionReview.Response.Allowed = false
				admissionReview.Response.Result = &metav1.Status{
					Status:  failureStatus,
					Message: "Force flag must be set to unmount and format the drive",
				}
				return false
			}
			return true
		}
	}
	if !validateMountpoint() {
		return false
	}

	// Filesystem validation
	// (*) Allow only "xfs" formatting
	// (*) Check if `force` flag is set for formatting
	validateFS := func() bool {
		requestedFilesystem := requestedFormat.Filesystem
		switch requestedFilesystem {
		case "":
			return true
		case xfsFileSystem:
			if !requestedFormat.Force {
				admissionReview.Response.Allowed = false
				admissionReview.Response.Result = &metav1.Status{
					Status:  failureStatus,
					Message: "Force flag must be set to override the format and remount",
				}
				return false
			}
			return true
		default:
			admissionReview.Response.Allowed = false
			admissionReview.Response.Result = &metav1.Status{
				Status:  failureStatus,
				Message: "DirectCSI supports only xfs filesystem format",
			}
			return false
		}
	}

	return validateFS()
}

/* Validates the following admission rules
   - Check if the fstype in the requestedFormat == "xfs"
   - Check if directCSIOwned is not set to True or requestedFormat is set for root partitions (unavailable drives)
   - Check if requestedFormat is not set for a drive in-use
   - Check if force option is set if the drive has an existing filesystem or mountpoint
*/
func (vh *validationHandler) validateDrive(w http.ResponseWriter, r *http.Request) {

	admissionReview, err := parseAdmissionReview(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse the body: %v", err), http.StatusBadRequest)
		return
	}

	rawObj := admissionReview.Request.Object.Raw

	dcsiDrive := directcsi.DirectCSIDrive{}
	if err := json.Unmarshal(rawObj, &dcsiDrive); err != nil {
		http.Error(w, fmt.Sprintf("could not parse directCSI object: %v", err), http.StatusInternalServerError)
		return
	}

	admissionReview.Response = &admissionv1.AdmissionResponse{
		UID:     admissionReview.Request.UID,
		Allowed: true,
	}

	if !validateRequestedFormat(dcsiDrive, &admissionReview) {
		writeSuccessResponse(admissionReview, w)
		return
	}

	// Add more validations here

	writeSuccessResponse(admissionReview, w)
}

// To-Do: Volume updates other than conditions shouldn't be allowed.
func (vh *validationHandler) validateVolume(w http.ResponseWriter, r *http.Request) {

	admissionReview, err := parseAdmissionReview(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse the body: %v", err), http.StatusBadRequest)
		return
	}

	//
	//
	// Add validation logic
	//
	//

	admissionReview.Response = &admissionv1.AdmissionResponse{
		UID:     admissionReview.Request.UID,
		Allowed: true,
	}

	writeSuccessResponse(admissionReview, w)
}
