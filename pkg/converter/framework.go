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

package converter

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/munnerz/goautoneg"

	"k8s.io/klog/v2"

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

const (
	versionV1Alpha1 = "direct.csi.min.io/v1alpha1"
	versionV1Beta1  = "direct.csi.min.io/v1beta1"
	versionV1Beta2  = "direct.csi.min.io/v1beta2"
	versionV1Beta3  = "direct.csi.min.io/v1beta3"
	versionV1Beta4  = "direct.csi.min.io/v1beta4"
)

var supportedVersions = []string{
	versionV1Alpha1,
	versionV1Beta1,
	versionV1Beta2,
	versionV1Beta3,
	versionV1Beta4,
} // ordered

type crdKind string

const (
	driveCRDKind  crdKind = "DirectCSIDrive"
	volumeCRDKind crdKind = "DirectCSIVolume"
)

// convertFunc is the user defined function for any conversion. The code in this file is a
// template that can be use for any CR conversion given this function.
type convertFunc func(Object *unstructured.Unstructured, version string) (*unstructured.Unstructured, metav1.Status)

// conversionResponseFailureWithMessagef is a helper function to create an AdmissionResponse
// with a formatted embedded error message.
func conversionResponseFailureWithMessagef(msg string, params ...interface{}) *v1.ConversionResponse {
	return &v1.ConversionResponse{
		Result: metav1.Status{
			Message: fmt.Sprintf(msg, params...),
			Status:  metav1.StatusFailure,
		},
	}

}

func statusErrorWithMessage(msg string, params ...interface{}) metav1.Status {
	return metav1.Status{
		Message: fmt.Sprintf(msg, params...),
		Status:  metav1.StatusFailure,
	}
}

func statusSucceed() metav1.Status {
	return metav1.Status{
		Status: metav1.StatusSuccess,
	}
}

// doConversion converts the requested object given the conversion function and returns a conversion response.
// failures will be reported as Reason in the conversion response.
func doConversion(convertRequest *v1.ConversionRequest, convert convertFunc) *v1.ConversionResponse {
	var convertedObjects []runtime.RawExtension
	for _, obj := range convertRequest.Objects {
		cr := unstructured.Unstructured{}
		if err := cr.UnmarshalJSON(obj.Raw); err != nil {
			klog.Error(err)
			return conversionResponseFailureWithMessagef("failed to unmarshall object (%v) with error: %v", string(obj.Raw), err)
		}
		convertedCR, status := convert(&cr, convertRequest.DesiredAPIVersion)
		if status.Status != metav1.StatusSuccess {
			klog.Error(status.String())
			return &v1.ConversionResponse{
				Result: status,
			}
		}
		convertedCR.SetAPIVersion(convertRequest.DesiredAPIVersion)
		convertedObjects = append(convertedObjects, runtime.RawExtension{Object: convertedCR})
	}
	return &v1.ConversionResponse{
		ConvertedObjects: convertedObjects,
		Result:           statusSucceed(),
	}
}

func serve(w http.ResponseWriter, r *http.Request, convert convertFunc) {
	var body []byte
	if r.Body != nil {
		if data, err := io.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	contentType := r.Header.Get("Content-Type")
	serializer := getInputSerializer(contentType)
	if serializer == nil {
		msg := fmt.Sprintf("invalid Content-Type header `%s`", contentType)
		klog.Errorf(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	klog.V(5).Infof("handling request: %v", body)
	convertReview := v1.ConversionReview{}
	if _, _, err := serializer.Decode(body, nil, &convertReview); err != nil {
		klog.Error(err)
		convertReview.Response = conversionResponseFailureWithMessagef("failed to deserialize body (%v) with error %v", string(body), err)
	} else {
		convertReview.Response = doConversion(convertReview.Request, convert)
		convertReview.Response.UID = convertReview.Request.UID
	}
	klog.V(5).Info(fmt.Sprintf("sending response: %v", convertReview.Response))

	// reset the request, it is not needed in a response.
	convertReview.Request = &v1.ConversionRequest{}

	accept := r.Header.Get("Accept")
	outSerializer := getOutputSerializer(accept)
	if outSerializer == nil {
		msg := fmt.Sprintf("invalid accept header `%s`", accept)
		klog.Errorf(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	err := outSerializer.Encode(&convertReview, w)
	if err != nil {
		klog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func serveDriveConversion(w http.ResponseWriter, r *http.Request) {
	serve(w, r, convertDriveCRD)
}

func serveVolumeConversion(w http.ResponseWriter, r *http.Request) {
	serve(w, r, convertVolumeCRD)
}

type mediaType struct {
	Type, SubType string
}

var scheme = runtime.NewScheme()
var serializers = map[mediaType]runtime.Serializer{
	{"application", "json"}: json.NewSerializerWithOptions(json.DefaultMetaFactory, scheme, scheme, json.SerializerOptions{}),
	{"application", "yaml"}: json.NewSerializerWithOptions(json.DefaultMetaFactory, scheme, scheme, json.SerializerOptions{Yaml: true}),
}

func getInputSerializer(contentType string) runtime.Serializer {
	parts := strings.SplitN(contentType, "/", 2)
	if len(parts) != 2 {
		return nil
	}
	return serializers[mediaType{parts[0], parts[1]}]
}

func getOutputSerializer(accept string) runtime.Serializer {
	if len(accept) == 0 {
		return serializers[mediaType{"application", "json"}]
	}

	clauses := goautoneg.ParseAccept(accept)
	for _, clause := range clauses {
		for k, v := range serializers {
			switch {
			case clause.Type == k.Type && clause.SubType == k.SubType,
				clause.Type == k.Type && clause.SubType == "*",
				clause.Type == "*" && clause.SubType == "*":
				return v
			}
		}
	}

	return nil
}
