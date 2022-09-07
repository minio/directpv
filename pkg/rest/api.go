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
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/ellipsis"
	"github.com/minio/directpv/pkg/matcher"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

const (
	apiPort                 = "30443"
	apiCertPath             = "/tmp/certs/api-cert.pem"
	apiKeyPath              = "/tmp/certs/api-key.pem"
	nodeCA                  = "/tmp/certs/node-ca.crt"
	drivesList              = "/drives/list"
	drivesFormat            = "/drives/format"
	directPVNamespace       = "directpv-min-io"
	directPVNodeServiceName = "directpv-min-io"
)

var (
	errNoSubsetsFound   = errors.New("no subsets found for the node service")
	errNoEndpointsFound = errors.New("no endpoints found for the node service")
)

// ServeAPIServer starts the DirectPV API server
func ServeAPIServer(ctx context.Context) error {
	certs, err := tls.LoadX509KeyPair(apiCertPath, apiKeyPath)
	if err != nil {
		klog.Errorf("Filed to load key pair for the DirectPV API server: %v", err)
		return err
	}

	// Create a secure http server
	server := &http.Server{
		TLSConfig: &tls.Config{
			Certificates:       []tls.Certificate{certs},
			InsecureSkipVerify: true,
		},
	}

	// define http server and server handler
	mux := http.NewServeMux()
	mux.HandleFunc(drivesList, listDrivesHandler)
	mux.HandleFunc(drivesFormat, formatDrivesHandler)
	server.Handler = mux

	lc := net.ListenConfig{}
	listener, lErr := lc.Listen(ctx, "tcp", fmt.Sprintf(":%v", apiPort))
	if lErr != nil {
		return lErr
	}

	go func() {
		klog.V(3).Infof("Starting DirectPV API server in port: %s", apiPort)
		if err := server.ServeTLS(listener, "", ""); err != nil {
			klog.Errorf("Failed to listen and serve DirectPV API server: %v", err)
		}
	}()

	return nil
}

func listDrivesHandler(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		klog.Errorf("couldn't read the request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var req GetDevicesRequest
	if err = json.Unmarshal(data, &req); err != nil {
		klog.Errorf("couldn't parse the request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	deviceInfo, err := listDrives(context.Background(), req)
	if err != nil {
		klog.Errorf("couldn't get the drive list: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Marshal API response
	jsonBytes, err := json.Marshal(GetDevicesResponse{
		DeviceInfo: deviceInfo,
	})
	if err != nil {
		klog.Errorf("Couldn't marshal the format status: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(jsonBytes)))
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
	w.WriteHeader(http.StatusOK)
}

func listDrives(ctx context.Context, req GetDevicesRequest) (map[NodeName][]Device, error) {
	var nodes []string
	var err error
	if len(req.Nodes) > 0 {
		nodes, err = ellipsis.Expand(string(req.Nodes))
		if err != nil {
			return nil, fmt.Errorf("couldn't expand the node selector %v: %v", req.Nodes, err)
		}
	}
	endpointsMap, err := getNodeEndpoints(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't get the node endpoints: %v", err)
	}
	httpClient := &http.Client{
		Transport: getTransport()(),
	}
	reqBody, err := json.Marshal(GetDevicesRequest{
		Drives:   req.Drives,
		Statuses: req.Statuses,
	})
	if err != nil {
		return nil, fmt.Errorf("errror while marshalling the request: %v", err)
	}
	var devices = make(map[NodeName][]Device)
	var mutex = &sync.RWMutex{}
	var wg sync.WaitGroup
	for node, ip := range endpointsMap {
		if len(nodes) > 0 && !matcher.StringIn(nodes, node) {
			continue
		}
		wg.Add(1)
		go func(node, ip string) {
			defer wg.Done()
			reqURL := fmt.Sprintf("https://%s:%s%s", ip, nodeAPIPort, drivesList)
			req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(reqBody))
			if err != nil {
				klog.Infof("error while constructing request: %v", err)
				return
			}
			resp, err := httpClient.Do(req)
			if err != nil {
				klog.Errorf("failed to get the result from node: %s, url: %s, error: %v", node, req.URL, err)
				return
			}
			defer drainBody(resp.Body)
			if resp.StatusCode != http.StatusOK {
				klog.Errorf("failed to get the result from node: %s, url: %s, statusCode: %d", node, req.URL, resp.StatusCode)
				return
			}
			nodeResponseInBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				klog.Errorf("failed to read response from node: %s, url: %s: %v", node, req.URL, err)
				return
			}
			var nodeResponse GetDevicesResponse
			if err := json.Unmarshal(nodeResponseInBytes, &nodeResponse); err != nil {
				klog.Errorf("couldn't parse the response from node: %s, url: %s: %v", node, req.URL, err)
				return
			}
			for k, v := range nodeResponse.DeviceInfo {
				mutex.Lock()
				devices[k] = v
				mutex.Unlock()
			}
		}(node, ip)
	}
	wg.Wait()
	return devices, nil
}

func formatDrivesHandler(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		klog.Errorf("couldn't read the request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var req FormatDevicesRequest
	if err = json.Unmarshal(data, &req); err != nil {
		klog.Errorf("couldn't parse the request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	formatStatus, err := formatDrives(context.Background(), req)
	if err != nil {
		klog.Errorf("couldn't format the drives: %v", err)
		return
	}
	// Marshal API response
	jsonBytes, err := json.Marshal(FormatDevicesResponse{
		DeviceInfo: formatStatus,
	})
	if err != nil {
		klog.Errorf("Couldn't marshal the format status: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(jsonBytes)))
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func formatDrives(ctx context.Context, req FormatDevicesRequest) (map[NodeName][]FormatDeviceStatus, error) {
	endpointsMap, err := getNodeEndpoints(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't get the node endpoints: %v", err)
	}
	httpClient := &http.Client{
		Transport: getTransport()(),
	}
	var wg sync.WaitGroup
	var formatStatus = make(map[NodeName][]FormatDeviceStatus)
	var mutex = &sync.RWMutex{}
	for node, formatDevices := range req.FormatInfo {
		endpoint, ok := endpointsMap[string(node)]
		if !ok {
			klog.Errorf("couldn't find an endpoint for %s", node)
			continue
		}
		wg.Add(1)
		go func(node NodeName, ip string, formatDevices []FormatDevice) {
			defer wg.Done()
			reqBody, err := json.Marshal(FormatDevicesRequest{
				FormatInfo: map[NodeName][]FormatDevice{
					node: formatDevices,
				},
			})
			reqURL := fmt.Sprintf("https://%s:%s%s", ip, nodeAPIPort, drivesFormat)
			req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(reqBody))
			if err != nil {
				klog.Infof("error while constructing request: %v", err)
				return
			}
			resp, err := httpClient.Do(req)
			if err != nil {
				klog.Errorf("failed to get the result from node: %s, url: %s, error: %v", node, req.URL, err)
				return
			}
			defer drainBody(resp.Body)
			if resp.StatusCode != http.StatusOK {
				klog.Errorf("failed to get the result from node: %s, url: %s, statusCode: %d", node, req.URL, resp.StatusCode)
				return
			}
			nodeResponseInBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				klog.Errorf("failed to read response from node: %s, url: %s: %v", node, req.URL, err)
				return
			}
			var nodeResponse FormatDevicesResponse
			if err = json.Unmarshal(nodeResponseInBytes, &nodeResponse); err != nil {
				klog.Errorf("couldn't parse the response from node: %s, url: %s: %v", node, req.URL, err)
				return
			}
			for k, v := range nodeResponse.DeviceInfo {
				mutex.Lock()
				formatStatus[k] = v
				mutex.Unlock()
			}
		}(node, endpoint, formatDevices)
	}
	wg.Wait()
	return formatStatus, nil
}

func getNodeEndpoints(ctx context.Context) (map[string]string, error) {
	endpoints, err := client.GetKubeClient().CoreV1().Endpoints(directPVNamespace).Get(ctx, directPVNodeServiceName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if len(endpoints.Subsets) == 0 {
		return nil, errNoSubsetsFound
	}
	var endpointsMap map[string]string
	for _, address := range endpoints.Subsets[0].Addresses {
		endpointsMap[*address.NodeName] = address.IP
	}
	if len(endpointsMap) == 0 {
		return nil, errNoEndpointsFound
	}
	return endpointsMap, nil
}
