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

package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"

	directv1alpha1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	FailureStatus = "Failure"
	SuccessStatus = "Success"
)

type ValidationHandler struct {
}

func parseAdmissionReview(req *http.Request) (admissionv1.AdmissionReview, error) {
	var body []byte
	admissionReview := admissionv1.AdmissionReview{}

	if req.Method != http.MethodPost {
		return admissionReview, errors.New("Invalid HTTP Method")
	}

	if req.Body != nil {
		if data, err := ioutil.ReadAll(req.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		return admissionReview, errors.New("Request body empty")
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

func validateRequestedFormat(directCSIDrive directv1alpha1.DirectCSIDrive, admissionReview *admissionv1.AdmissionReview) bool {
	requestedFormat := directCSIDrive.Spec.RequestedFormat

	if reflect.DeepEqual(requestedFormat, directv1alpha1.RequestedFormat{}) {
		return true
	}

	if directCSIDrive.Status.DriveStatus == directv1alpha1.DriveStatusUnavailable || directCSIDrive.Status.Mountpoint == "/" {
		admissionReview.Response.Allowed = false
		admissionReview.Response.Result = &metav1.Status{
			Status:  FailureStatus,
			Message: "Root partition'ed drives cannot be added/formatted",
		}
		return false
	}

	if directCSIDrive.Status.DriveStatus == directv1alpha1.DriveStatusInUse {
		admissionReview.Response.Allowed = false
		admissionReview.Response.Result = &metav1.Status{
			Status:  FailureStatus,
			Message: "Drives in-use cannot be formatted and added",
		}
		return false
	}

	if requestedFormat.Filesystem != "" && requestedFormat.Filesystem != "xfs" {
		admissionReview.Response.Allowed = false
		admissionReview.Response.Result = &metav1.Status{
			Status:  FailureStatus,
			Message: "DirectCSI supports only xfs filesystem format",
		}
		return false
	}

	if directCSIDrive.Status.Filesystem != "" || directCSIDrive.Status.Mountpoint != "" {
		if !requestedFormat.Force {
			admissionReview.Response.Allowed = false
			admissionReview.Response.Result = &metav1.Status{
				Status:  FailureStatus,
				Message: "Force flag must be set to override the format and remount",
			}
			return false
		}
	}

	return true
}

/* Validates the following admission rules
   - Check if the fstype in the requestedFormat == "xfs"
   - Check if directCSIOwned is not set to True or requestedFormat is set for root partitions (unavailable drives)
   - Check if requestedFormat is not set for a drive in-use
   - Check if force option is set if the drive has an existing filesystem or mountpoint
*/
func (vh *ValidationHandler) validateDrive(w http.ResponseWriter, r *http.Request) {

	admissionReview, err := parseAdmissionReview(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse the body: %v", err), http.StatusBadRequest)
		return
	}

	rawObj := admissionReview.Request.Object.Raw

	dcsiDrive := directv1alpha1.DirectCSIDrive{}
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

	writeSuccessResponse(admissionReview, w)
}

// To-Do: Volume updates other than conditions shouldn't be allowed.
func (vh *ValidationHandler) validateVolume(w http.ResponseWriter, r *http.Request) {

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
