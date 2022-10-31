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
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"path"
	"strconv"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"k8s.io/klog/v2"
)

func getHeaderValue(headers http.Header, key string) (value string, err error) {
	values, found := headers[http.CanonicalHeaderKey(key)]
	if !found {
		err = fmt.Errorf("header key %v must be provided", key)
		return
	}
	if len(values) == 0 || values[0] == "" {
		err = fmt.Errorf("value to header key %v must be provided", key)
		return
	}
	value = values[0]
	return
}

func drainBodyHandler(handler func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		handler(w, r)
		io.Copy(ioutil.Discard, r.Body)
		r.Body.Close()
	}
}

func authHandler(handler func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		contentLength, err := getHeaderValue(r.Header, "Content-Length")
		if err != nil {
			w.WriteHeader(http.StatusLengthRequired)
			w.Write(apiErrorf("invalid content-length header; %v", err))
			return
		}
		size, err := strconv.ParseUint(contentLength, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(apiErrorf("invalid content-length; %v", err))
			return
		}
		if size > 5*1024*1024 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(apiErrorf("content-length too big; supports 5 MiB"))
			return
		}

		contentSha256, err := getHeaderValue(r.Header, "x-amz-content-sha256")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(apiErrorf("invalid x-amz-content-sha256 header; %v", err))
			return
		}
		if contentSha256 != "UNSIGNED-PAYLOAD" {
			body := &bytes.Buffer{}
			written, err := io.CopyN(body, r.Body, int64(size))
			if err != nil || written != int64(size) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write(apiErrorf("body too small; %v", err))
				return
			}
			computed := sha256Hash(body.Bytes())
			if contentSha256 != computed {
				w.WriteHeader(http.StatusBadRequest)
				w.Write(apiErrorf("x-amz-content-sha256 mismatch; passed: %v, computed: %v", contentSha256, computed))
				return
			}
			r.Body = io.NopCloser(body)
		}

		cred, err := getCredentialFromSecrets(context.Background())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(apiErrorf("credential: %v", err))
			return
		}

		r.Header.Add("host", r.Host)
		if err := checkSignV4CSI(r.Header, r.URL.EscapedPath(), cred, contentSha256); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write(apiErrorf("signature mismatch; %v", err))
			return
		}

		handler(w, r)
	}
}

func postHandler(handler func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write(apiErrorf("method %v not allowed", r.Method))
			return
		}

		handler(w, r)
	}
}

type httpHandler interface {
	ListDevicesHandler(res http.ResponseWriter, req *http.Request)
	FormatDevicesHandler(res http.ResponseWriter, req *http.Request)
}

type nodeHTTPHandler struct {
	rpc *nodeRPCServer
}

func (handler *nodeHTTPHandler) ListDevicesHandler(w http.ResponseWriter, r *http.Request) {
	var request NodeListDevicesRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(apiErrorf("invalid node list devices request; %v", err))
		return
	}

	response, err := handler.rpc.ListDevices(&request)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(apiErrorf("rpc error: %v", err))
		return
	}

	body, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(apiErrorf("json: %v", err))
		return
	}

	w.Write(body)
}

func (handler *nodeHTTPHandler) FormatDevicesHandler(w http.ResponseWriter, r *http.Request) {
	var request NodeFormatDevicesRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(apiErrorf("invalid node format devices request; %v", err))
		return
	}

	response, err := handler.rpc.FormatDevices(&request)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(apiErrorf("rpc error: %v", err))
		return
	}

	body, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(apiErrorf("json: %v", err))
		return
	}

	w.Write(body)
}

type adminHTTPHandler struct {
	rpc *rpcServer
}

func (handler *adminHTTPHandler) ListDevicesHandler(w http.ResponseWriter, r *http.Request) {
	var request ListDevicesRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(apiErrorf("invalid list devices request; %v", err))
		return
	}

	response, err := handler.rpc.ListDevices(&request)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(apiErrorf("rpc error: %v", err))
		return
	}

	body, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(apiErrorf("json: %v", err))
		return
	}

	w.Write(body)
}

func (handler *adminHTTPHandler) FormatDevicesHandler(w http.ResponseWriter, r *http.Request) {
	var request FormatDevicesRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(apiErrorf("invalid format devices request; %v", err))
		return
	}

	response, err := handler.rpc.FormatDevices(&request)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(apiErrorf("rpc error: %v", err))
		return
	}

	body, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(apiErrorf("json: %v", err))
		return
	}

	w.Write(body)
}

func startHTTPServer(ctx context.Context, port int, certFile, keyFile string, handler httpHandler) error {
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc(devicesListPath, drainBodyHandler(postHandler(authHandler(handler.ListDevicesHandler))))
	mux.HandleFunc(devicesFormatPath, drainBodyHandler(postHandler(authHandler(handler.FormatDevicesHandler))))

	server := &http.Server{
		TLSConfig: &tls.Config{
			Certificates:       []tls.Certificate{certificate},
			InsecureSkipVerify: true,
		},
		// FIXME: Implement GetCertificate
		Handler: mux,
	}

	config := net.ListenConfig{}
	listener, err := config.Listen(ctx, "tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		return err
	}

	for {
		klog.V(3).Infof("Serving at port :%v", port)
		if err = server.ServeTLS(listener, "", ""); err != nil {
			return err
		}
	}
}

// StartServer starts admin server.
func StartServer(ctx context.Context, port int) error {
	return startHTTPServer(
		ctx,
		port,
		path.Join(consts.AdminServerCertsPath, consts.PublicCertFileName),
		path.Join(consts.AdminServerCertsPath, consts.PrivateKeyFileName),
		&adminHTTPHandler{newRPCServer()},
	)
}

// StartNodeAPIServer starts node API server.
func StartNodeAPIServer(ctx context.Context, port int, identity string, nodeID directpvtypes.NodeID, rack, zone, region string) error {
	server, err := newNodeRPCServer(
		ctx,
		nodeID,
		map[string]string{
			string(directpvtypes.TopologyDriverIdentity): identity,
			string(directpvtypes.TopologyDriverRack):     rack,
			string(directpvtypes.TopologyDriverZone):     zone,
			string(directpvtypes.TopologyDriverRegion):   region,
			string(directpvtypes.TopologyDriverNode):     string(nodeID),
		},
	)
	if err != nil {
		return err
	}

	return startHTTPServer(
		ctx,
		port,
		path.Join(consts.NodeAPIServerCertsPath, consts.PublicCertFileName),
		path.Join(consts.NodeAPIServerCertsPath, consts.PrivateKeyFileName),
		&nodeHTTPHandler{server},
	)
}
