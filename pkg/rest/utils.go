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

package rest

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/minio/directpv/pkg/device"
	"github.com/minio/directpv/pkg/types"
)

type matchFn func(drive *types.Drive, device *device.Device) bool

func getMatchedDevicesForDrive(drive *types.Drive, devices []*device.Device) ([]*device.Device, []*device.Device) {
	return getMatchedDevices(
		drive,
		devices,
		func(drive *types.Drive, device *device.Device) bool {
			return fsMatcher(drive, device)
		},
	)
}

func fsMatcher(drive *types.Drive, device *device.Device) bool {
	if drive.Status.FSUUID != device.FSUUID {
		return false
	}
	return true
}

func getMatchedDevices(drive *types.Drive, devices []*device.Device, matchFn matchFn) (matchedDevices, unmatchedDevices []*device.Device) {
	for _, device := range devices {
		if matchFn(drive, device) {
			matchedDevices = append(matchedDevices, device)
		} else {
			unmatchedDevices = append(unmatchedDevices, device)
		}
	}
	return matchedDevices, unmatchedDevices
}

func getDeviceNames(devices []*device.Device) string {
	var deviceNames []string
	for _, device := range devices {
		deviceNames = append(deviceNames, device.Name)
	}
	return strings.Join(deviceNames, ", ")
}

func writeFormatMetadata(formatMetadata FormatMetadata, filePath string) error {
	metaDataBytes, err := json.MarshalIndent(formatMetadata, "", "")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, metaDataBytes, 0644)
}

func (d FormatDevice) Path() string {
	return path.Join("/dev", d.Name)
}

func (d FormatDevice) Model() string {
	if d.UDevData == nil {
		return ""
	}
	return d.UDevData["ID_MODEL"]
}

func (d FormatDevice) Vendor() string {
	if d.UDevData == nil {
		return ""
	}
	return d.UDevData["ID_VENDOR"]
}

func getTransport() func() *http.Transport {
	// Keep TLS config.
	tlsConfig := &tls.Config{
		// Can't use SSLv3 because of POODLE and BEAST
		// Can't use TLSv1.0 because of POODLE and BEAST using CBC cipher
		// Can't use TLSv1.1 because of RC4 cipher usage
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true, // FIXME: use trusted CA
	}
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: 30 * time.Second,
		}).DialContext,
		MaxIdleConnsPerHost:   1024,
		IdleConnTimeout:       30 * time.Second,
		ResponseHeaderTimeout: 1 * time.Minute,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 5 * time.Second,
		TLSClientConfig:       tlsConfig,
		// Go net/http automatically unzip if content-type is
		// gzip disable this feature, as we are always interested
		// in raw stream.
		DisableCompression: true,
	}
	return func() *http.Transport {
		return tr
	}
}

// drainBody close non nil response with any response Body.
// convenient wrapper to drain any remaining data on response body.
//
// Subsequently this allows golang http RoundTripper
// to re-use the same connection for future requests.
func drainBody(respBody io.ReadCloser) {
	// Callers should close resp.Body when done reading from it.
	// If resp.Body is not closed, the Client's underlying RoundTripper
	// (typically Transport) may not be able to re-use a persistent TCP
	// connection to the server for a subsequent "keep-alive" request.
	if respBody != nil {
		// Drain any remaining Body and then close the connection.
		// Without this closing connection would disallow re-using
		// the same connection for future uses.
		//  - http://stackoverflow.com/a/17961593/4465767
		defer respBody.Close()
		io.Copy(ioutil.Discard, respBody)
	}
}
