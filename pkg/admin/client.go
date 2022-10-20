// This file is part of MinIO DirectPV
// Copyright (c) 2022 MinIO, Inc.
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
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

const (
	devicesListPath   = "/devices/list"
	devicesFormatPath = "/devices/format"
	contentType       = "application/json"
)

func newRequest(url *url.URL, data []byte, cred *Credential) (*http.Request, error) {
	request, err := http.NewRequest(http.MethodPost, url.String(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	date := time.Now().UTC()
	contentSha256 := sha256Hash(data)

	headers := http.Header{}
	headers.Add("host", url.Host)
	headers.Add("content-length", fmt.Sprintf("%v", len(data)))
	headers.Add("content-type", contentType)
	headers.Add("x-amz-date", date.Format(iso8601UTCLayout))
	headers.Add("x-amz-content-sha256", contentSha256)

	headers.Add("Authorization", SignV4CSI(headers, url.EscapedPath(), cred, contentSha256, date))
	request.Header = headers

	return request, nil
}

type nodeClient struct {
	url    *url.URL
	client *http.Client
}

func (c *nodeClient) ListDevices(devices []string, formatAllowed, formatDenied bool) (results []Device, err error) {
	data, err := json.Marshal(NodeListDevicesRequest{
		Devices:       devices,
		FormatAllowed: formatAllowed,
		FormatDenied:  formatDenied,
	})
	if err != nil {
		return nil, err
	}

	cred, err := getCredentialFromSecrets(context.Background())
	if err != nil {
		return nil, err
	}

	request, err := newRequest(c.url.JoinPath(devicesListPath), data, cred)
	if err != nil {
		return nil, err
	}

	r, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}

	if r.StatusCode < 200 || r.StatusCode > 299 {
		var errResp ErrorResponse
		if err := json.NewDecoder(r.Body).Decode(&errResp); err != nil {
			return nil, err
		}

		return nil, errors.New(errResp.Error)
	}

	var resp NodeListDevicesResponse
	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		return nil, err
	}

	return resp.Devices, nil
}

func (c *nodeClient) FormatDevices(devices []FormatDevice) (results []FormatResult, err error) {
	data, err := json.Marshal(NodeFormatDevicesRequest{
		Devices: devices,
	})
	if err != nil {
		return nil, err
	}

	cred, err := getCredentialFromSecrets(context.Background())
	if err != nil {
		return nil, err
	}

	request, err := newRequest(c.url.JoinPath(devicesFormatPath), data, cred)
	if err != nil {
		return nil, err
	}

	r, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}

	if r.StatusCode < 200 || r.StatusCode > 299 {
		var errResp ErrorResponse
		if err := json.NewDecoder(r.Body).Decode(&errResp); err != nil {
			return nil, err
		}

		return nil, errors.New(errResp.Error)
	}

	var resp NodeFormatDevicesResponse
	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		return nil, err
	}

	return resp.Devices, nil
}

type Client struct {
	url    *url.URL
	client *http.Client
}

func NewClient(url *url.URL) *Client {
	if url.Path == "" {
		url.Path = "/"
	}

	return &Client{
		url: url,
		client: &http.Client{
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				DialContext:           (&net.Dialer{Timeout: 1 * time.Minute}).DialContext,
				MaxIdleConnsPerHost:   1024,
				IdleConnTimeout:       1 * time.Minute,
				ResponseHeaderTimeout: 1 * time.Minute,
				TLSHandshakeTimeout:   15 * time.Second,
				ExpectContinueTimeout: 3 * time.Second,
				TLSClientConfig: &tls.Config{
					// Can't use SSLv3 because of POODLE and BEAST
					// Can't use TLSv1.0 because of POODLE and BEAST using CBC cipher
					// Can't use TLSv1.1 because of RC4 cipher usage
					MinVersion:         tls.VersionTLS12,
					InsecureSkipVerify: true, // FIXME: use trusted CA
				},
			},
		},
	}
}

func (c *Client) ListDevices(req *ListDevicesRequest, cred *Credential) (*ListDevicesResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	request, err := newRequest(c.url.JoinPath(devicesListPath), data, cred)
	if err != nil {
		return nil, err
	}

	r, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}

	if r.StatusCode < 200 || r.StatusCode > 299 {
		var errResp ErrorResponse
		if err := json.NewDecoder(r.Body).Decode(&errResp); err != nil {
			return nil, err
		}

		return nil, errors.New(errResp.Error)
	}

	var resp ListDevicesResponse
	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) FormatDevices(req *FormatDevicesRequest, cred *Credential) (*FormatDevicesResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	request, err := newRequest(c.url.JoinPath(devicesFormatPath), data, cred)
	if err != nil {
		return nil, err
	}

	r, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}

	if r.StatusCode < 200 || r.StatusCode > 299 {
		var errResp ErrorResponse
		if err := json.NewDecoder(r.Body).Decode(&errResp); err != nil {
			return nil, err
		}

		return nil, errors.New(errResp.Error)
	}

	var resp FormatDevicesResponse
	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
