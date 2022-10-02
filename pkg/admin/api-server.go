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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"path"
	"sync"

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/utils"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

const (
	devicesListAPIPath   = "/devices/list"
	devicesFormatAPIPath = "/devices/format"
)

var (
	apiServerPrivateKeyPath = path.Join(consts.APIServerCertsPath, consts.PrivateKeyFileName)
	apiServerCertPath       = path.Join(consts.APIServerCertsPath, consts.PublicCertFileName)
)

// ServeAPIServer starts the API server
func ServeAPIServer(ctx context.Context, apiPort int) error {
	certs, err := tls.LoadX509KeyPair(apiServerCertPath, apiServerPrivateKeyPath)
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
		// TODO: Implement GetCertificate
	}

	// define http server and server handler
	mux := http.NewServeMux()
	mux.HandleFunc(devicesListAPIPath, authMiddleware(listDevicesHandler))
	mux.HandleFunc(devicesFormatAPIPath, authMiddleware(formatDevicesHandler))
	mux.HandleFunc(consts.ReadinessPath, readinessHandler)
	server.Handler = mux

	lc := net.ListenConfig{}
	listener, lErr := lc.Listen(ctx, "tcp", fmt.Sprintf(":%v", apiPort))
	if lErr != nil {
		return lErr
	}

	errCh := make(chan error)
	go func() {
		klog.V(3).Infof("Starting API server in port: %d", apiPort)
		if err := server.ServeTLS(listener, "", ""); err != nil {
			klog.Errorf("Failed to listen and serve API server: %v", err)
			errCh <- err
		}
	}()

	return <-errCh
}

// listDevicesHandler gathers the list of available and unavailable devices from the nodes
func listDevicesHandler(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		klog.Errorf("couldn't read the request: %v", err)
		writeErrorResponse(w, http.StatusBadRequest, toAPIError(err, "couldn't read the request"))
		return
	}
	var req GetDevicesRequest
	if err = json.Unmarshal(data, &req); err != nil {
		klog.Errorf("couldn't parse the request: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, toAPIError(err, "couldn't parse the request"))
		return
	}
	deviceInfo, err := listDevices(context.Background(), req)
	if err != nil {
		klog.Errorf("couldn't get the drive list: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, toAPIError(err, "couldn't get the drive list"))
		return
	}
	jsonBytes, err := json.Marshal(GetDevicesResponse{
		DeviceInfo: deviceInfo,
	})
	if err != nil {
		klog.Errorf("couldn't marshal the format status: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, toAPIError(err, "couldn't marshal the format status"))
		return
	}
	writeSuccessResponse(w, jsonBytes)
}

// listDevices queries the nodes parallelly to get the available and unavailable devices
func listDevices(ctx context.Context, req GetDevicesRequest) (map[NodeName][]Device, error) {
	var nodes []string
	for _, nodeSelector := range req.Nodes {
		nodes = append(nodes, string(nodeSelector))
	}
	endpointsMap, nodeAPIPort, err := getNodeEndpoints(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't get the node endpoints: %v", err)
	}
	httpClient := &http.Client{
		Transport: getDefaultTransport(true),
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
		if len(nodes) > 0 && !utils.ItemIn(nodes, node) {
			continue
		}
		wg.Add(1)
		go func(node, ip string) {
			defer wg.Done()
			reqURL := fmt.Sprintf("https://%s:%d%s", ip, nodeAPIPort, devicesListAPIPath)
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
			nodeResponseInBytes, err := io.ReadAll(resp.Body)
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

// formatDevicesHandler forwards the format requests to respective nodes
func formatDevicesHandler(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		klog.Errorf("couldn't read the request: %v", err)
		writeErrorResponse(w, http.StatusBadRequest, toAPIError(err, "couldn't read the request"))
		return
	}
	var req FormatDevicesRequest
	if err = json.Unmarshal(data, &req); err != nil {
		klog.Errorf("couldn't parse the request: %v", err)
		writeErrorResponse(w, http.StatusBadRequest, toAPIError(err, "couldn't parse the request"))
		return
	}
	formatStatus, err := formatDrives(context.Background(), req)
	if err != nil {
		klog.Errorf("couldn't format the drives: %v", err)
		writeErrorResponse(w, http.StatusBadRequest, toAPIError(err, "couldn't format the drives"))
		return
	}
	// Marshal API response
	jsonBytes, err := json.Marshal(FormatDevicesResponse{
		DeviceInfo: formatStatus,
	})
	if err != nil {
		klog.Errorf("Couldn't marshal the format status: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, toAPIError(err, "couldn't format the drives"))
		return
	}
	writeSuccessResponse(w, jsonBytes)
}

// formatDrives forwards the format requests to respective nodes
func formatDrives(ctx context.Context, req FormatDevicesRequest) (map[NodeName][]FormatDeviceStatus, error) {
	endpointsMap, nodeAPIPort, err := getNodeEndpoints(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't get the node endpoints: %v", err)
	}
	httpClient := &http.Client{
		Transport: getDefaultTransport(true),
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
			if err != nil {
				klog.Infof("error while parsing format devices request: %v", err)
				return
			}
			reqURL := fmt.Sprintf("https://%s:%d%s", ip, nodeAPIPort, devicesFormatAPIPath)
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
			nodeResponseInBytes, err := io.ReadAll(resp.Body)
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

// getNodeEndpoints reads the endpoint objects present in the node svc to get the endpoints of the nodes
func getNodeEndpoints(ctx context.Context) (endpointsMap map[string]string, apiPort int, err error) {
	var endpoints *corev1.Endpoints
	endpoints, err = k8s.KubeClient().CoreV1().Endpoints(consts.Namespace).Get(ctx, consts.NodeAPIServerHLSVC, metav1.GetOptions{})
	if err != nil {
		return
	}
	if len(endpoints.Subsets) == 0 {
		err = errNoSubsetsFound
		return
	}
	endpointsMap = make(map[string]string)
	for _, address := range endpoints.Subsets[0].Addresses {
		endpointsMap[*address.NodeName] = address.IP
	}
	if len(endpointsMap) == 0 {
		err = errNoEndpointsFound
		return
	}
	for _, port := range endpoints.Subsets[0].Ports {
		if port.Name == consts.NodeAPIPortName {
			apiPort = int(port.Port)
			break
		}
	}
	if apiPort == 0 {
		err = errNodeAPIPortNotFound
	}
	return
}
