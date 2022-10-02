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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/minio-go/v7/pkg/s3utils"
	"github.com/minio/minio-go/v7/pkg/signer"
)

var (
	errEmptyResponse = errors.New("response is empty")
)

const (
	libraryVersion = "1.0"
	// User Agent follows the below style.
	//	DirectPV (OS; ARCH) LIB/VER APP/VER
	libraryUserAgentPrefix = consts.AppPrettyName + " (" + runtime.GOOS + "; " + runtime.GOARCH + ") "
	libraryUserAgent       = libraryUserAgentPrefix + consts.AppName + "/" + libraryVersion
)

type Client struct {
	// parsed endpoint provided by the user
	endpointURL *url.URL
	// Authorization keys
	accessKey string
	secretKey string
	// Indicate whether we are using https or not
	secure bool
	// Needs http client to be initialized
	httpClient *http.Client

	// Advanced functionality
	isTraceEnabled bool
	traceOutput    io.Writer
	// User supplied origin info
	appInfo struct {
		appName    string
		appVersion string
	}
}

// New initiates a new admin client
func New(endpoint, accessKey, secretKey string, secure bool) (*Client, error) {
	clnt, err := privateNew(endpoint, accessKey, secretKey, secure)
	if err != nil {
		return nil, err
	}
	return clnt, nil
}

func privateNew(endpoint, accessKey, secretKey string, secure bool) (*Client, error) {
	endpointURL, err := getEndpointURL(endpoint, secure)
	if err != nil {
		return nil, err
	}

	clnt := new(Client)

	// Set the creds
	clnt.accessKey = accessKey
	clnt.secretKey = secretKey

	// Remember whether we are using https or not
	clnt.secure = secure

	// Save endpoint URL, user agent for future uses.
	clnt.endpointURL = endpointURL

	// Instantiate http client and bucket location cache.
	clnt.httpClient = &http.Client{
		Transport: getDefaultTransport(secure),
	}

	return clnt, nil
}

// SetAppInfo - add application details to user agent.
func (adm *Client) SetAppInfo(appName string, appVersion string) {
	if appName != "" && appVersion != "" {
		adm.appInfo.appName = appName
		adm.appInfo.appVersion = appVersion
	}
}

// SetCustomTransport - set new custom transport.
func (adm *Client) SetCustomTransport(customHTTPTransport http.RoundTripper) {
	// Set this to override default transport
	// ``http.DefaultTransport``.
	//
	// This transport is usually needed for debugging OR to add your
	// own custom TLS certificates on the client transport, for custom
	// CA's and certs which are not part of standard certificate
	// authority follow this example :-
	//
	//   tr := &http.Transport{
	//           TLSClientConfig:    &tls.Config{RootCAs: pool},
	//           DisableCompression: true,
	//   }
	//   api.SetTransport(tr)
	//
	if adm.httpClient != nil {
		adm.httpClient.Transport = customHTTPTransport
	}
}

// TraceOn - enable HTTP tracing.
func (adm *Client) TraceOn(outputStream io.Writer) {
	// if outputStream is nil then default to os.Stdout.
	if outputStream == nil {
		outputStream = os.Stdout
	}
	// Sets a new output stream.
	adm.traceOutput = outputStream

	// Enable tracing.
	adm.isTraceEnabled = true
}

// TraceOff - disable HTTP tracing.
func (adm *Client) TraceOff() {
	// Disable tracing.
	adm.isTraceEnabled = false
}

// Filter out signature value from Authorization header.
func (adm Client) filterSignature(req *http.Request) {
	/// Signature V4 authorization header.

	// Save the original auth.
	origAuth := req.Header.Get("Authorization")
	// Strip out accessKeyID from:
	// Credential=<access-key-id>/<date>/<aws-region>/<aws-service>/aws4_request
	regCred := regexp.MustCompile("Credential=([A-Z0-9]+)/")
	newAuth := regCred.ReplaceAllString(origAuth, "Credential=**REDACTED**/")

	// Strip out 256-bit signature from: Signature=<256-bit signature>
	regSign := regexp.MustCompile("Signature=([[0-9a-f]+)")
	newAuth = regSign.ReplaceAllString(newAuth, "Signature=**REDACTED**")

	// Set a temporary redacted auth
	req.Header.Set("Authorization", newAuth)
}

// dumpHTTP - dump HTTP request and response.
func (adm Client) dumpHTTP(req *http.Request, resp *http.Response) error {
	// Starts http dump.
	_, err := fmt.Fprintln(adm.traceOutput, "---------START-HTTP---------")
	if err != nil {
		return err
	}

	// Filter out Signature field from Authorization header.
	adm.filterSignature(req)

	// Only display request header.
	reqTrace, err := httputil.DumpRequestOut(req, false)
	if err != nil {
		return err
	}

	// Write request to trace output.
	_, err = fmt.Fprint(adm.traceOutput, string(reqTrace))
	if err != nil {
		return err
	}

	// Only display response header.
	var respTrace []byte

	// For errors we make sure to dump response body as well.
	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusPartialContent &&
		resp.StatusCode != http.StatusNoContent {
		respTrace, err = httputil.DumpResponse(resp, true)
		if err != nil {
			return err
		}
	} else {
		// WORKAROUND for https://github.com/golang/go/issues/13942.
		// httputil.DumpResponse does not print response headers for
		// all successful calls which have response ContentLength set
		// to zero. Keep this workaround until the above bug is fixed.
		if resp.ContentLength == 0 {
			var buffer bytes.Buffer
			if err = resp.Header.Write(&buffer); err != nil {
				return err
			}
			respTrace = buffer.Bytes()
			respTrace = append(respTrace, []byte("\r\n")...)
		} else {
			respTrace, err = httputil.DumpResponse(resp, false)
			if err != nil {
				return err
			}
		}
	}
	// Write response to trace output.
	_, err = fmt.Fprint(adm.traceOutput, strings.TrimSuffix(string(respTrace), "\r\n"))
	if err != nil {
		return err
	}

	// Ends the http dump.
	_, err = fmt.Fprintln(adm.traceOutput, "---------END-HTTP---------")
	return err
}

// do - execute http request.
func (adm Client) do(req *http.Request) (*http.Response, error) {
	resp, err := adm.httpClient.Do(req)
	if err != nil {
		// Handle this specifically for now until future Golang versions fix this issue properly.
		if urlErr, ok := err.(*url.Error); ok {
			if strings.Contains(urlErr.Err.Error(), "EOF") {
				return nil, &url.Error{
					Op:  urlErr.Op,
					URL: urlErr.URL,
					Err: errors.New("Connection closed by foreign host " + urlErr.URL + ". Retry again."),
				}
			}
		}
		return nil, err
	}

	// Response cannot be non-nil
	if resp == nil {
		return nil, errEmptyResponse
	}

	// If trace is enabled, dump http request and response.
	if adm.isTraceEnabled {
		err = adm.dumpHTTP(req, resp)
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}

// set User agent.
func (adm Client) setUserAgent(req *http.Request) {
	req.Header.Set("User-Agent", libraryUserAgent)
	if adm.appInfo.appName != "" && adm.appInfo.appVersion != "" {
		req.Header.Set("User-Agent", libraryUserAgent+" "+adm.appInfo.appName+"/"+adm.appInfo.appVersion)
	}
}

// RequestData exposing internal data structure requestData
type RequestData struct {
	CustomHeaders http.Header
	QueryValues   url.Values
	RelPath       string // URL path relative to admin API base endpoint
	Content       []byte
}

// ExecuteMethod - similar to internal method executeMethod() useful
// for writing custom requests.
func (adm Client) ExecuteMethod(ctx context.Context, method string, reqData RequestData) (res *http.Response, err error) {
	return adm.executeMethod(ctx, method, reqData)
}

func (adm Client) executeMethod(ctx context.Context, method string, reqData RequestData) (res *http.Response, err error) {
	defer func() {
		if err != nil {
			// close idle connections before returning, upon error.
			adm.httpClient.CloseIdleConnections()
		}
	}()

	// Create cancel context to control 'newRetryTimer' go routine.
	ctx, cancel := context.WithCancel(ctx)

	// Indicate to our routine to exit cleanly upon return.
	defer cancel()

	// Instantiate a new request.
	var req *http.Request
	req, err = adm.newRequest(ctx, method, reqData)
	if err != nil {
		return nil, err
	}

	// Initiate the request.
	return adm.do(req)
}

// newRequest - instantiate a new HTTP request for a given method.
func (adm Client) newRequest(ctx context.Context, method string, reqData RequestData) (req *http.Request, err error) {
	// If no method is supplied default to 'POST'.
	if method == "" {
		method = "POST"
	}

	// Construct a new target URL.
	targetURL, err := adm.makeTargetURL(reqData)
	if err != nil {
		return nil, err
	}

	// Initialize a new HTTP request for the method.
	req, err = http.NewRequestWithContext(ctx, method, targetURL.String(), nil)
	if err != nil {
		return nil, err
	}

	adm.setUserAgent(req)
	for k, v := range reqData.CustomHeaders {
		req.Header.Set(k, v[0])
	}
	if length := len(reqData.Content); length > 0 {
		req.ContentLength = int64(length)
	}
	req.Body = ioutil.NopCloser(bytes.NewReader(reqData.Content))

	req = signer.SignV4(*req, adm.accessKey, adm.secretKey, "", "")
	return req, nil
}

// makeTargetURL make a new target url.
func (adm Client) makeTargetURL(r RequestData) (*url.URL, error) {
	host := adm.endpointURL.Host
	scheme := adm.endpointURL.Scheme

	// ToDo: Embed versions for a versioned API
	urlStr := scheme + "://" + host + r.RelPath

	// If there are any query values, add them to the end.
	if len(r.QueryValues) > 0 {
		urlStr = urlStr + "?" + s3utils.QueryEncode(r.QueryValues)
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	return u, nil
}
