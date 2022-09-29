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
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"reflect"
	"sync"

	"github.com/google/uuid"
	"github.com/hashicorp/errwrap"
	apiTypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/device"
	"github.com/minio/directpv/pkg/drive"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

var (
	nodeAPIServerPrivateKeyPath = path.Join(consts.NodeAPIServerCertsPath, consts.PrivateKeyFileName)
	nodeAPIServerCertPath       = path.Join(consts.NodeAPIServerCertsPath, consts.PublicCertFileName)
)

// suggestions
var (
	formatRetrySuggestion          = "retry the format request"
	formatRetryWithForceSuggestion = "retry the format request with force"
)

// reasons
var (
	udevDataMismatchReason = "probed udevdata isn't matching with the udev data in the request"
	metaDataPathSuffix     = path.Join(fmt.Sprintf(".%s.sys", consts.AppName), "metadata.json")
)

// ServeNodeAPIServer starts the DirectPV Node API server
func ServeNodeAPIServer(ctx context.Context, nodeAPIPort int, identity, nodeID, rack, zone, region string) error {
	certs, err := tls.LoadX509KeyPair(nodeAPIServerCertPath, nodeAPIServerPrivateKeyPath)
	if err != nil {
		klog.Errorf("Filed to load key pair: %v", err)
		return err
	}

	// Create a secure http server
	server := &http.Server{
		TLSConfig: &tls.Config{
			Certificates:       []tls.Certificate{certs},
			InsecureSkipVerify: true,
		},
	}

	nodeHandler, err := newNodeAPIHandler(ctx, identity, nodeID, rack, zone, region)
	if err != nil {
		return err
	}

	// define http server and server handler
	mux := http.NewServeMux()
	mux.HandleFunc(devicesListAPIPath, nodeHandler.listLocalDevicesHandler)
	mux.HandleFunc(devicesFormatAPIPath, nodeHandler.formatLocalDevicesHandler)
	mux.HandleFunc(consts.ReadinessPath, readinessHandler)
	server.Handler = mux

	lc := net.ListenConfig{}
	listener, lErr := lc.Listen(ctx, "tcp", fmt.Sprintf(":%v", nodeAPIPort))
	if lErr != nil {
		return lErr
	}

	errCh := make(chan error)
	go func() {
		klog.V(3).Infof("Starting %s Node API server in port: %d", consts.AppPrettyName, nodeAPIPort)
		if err := server.ServeTLS(listener, "", ""); err != nil {
			klog.Errorf("Failed to listen and serve DirectPV Node API server: %v", err)
			errCh <- err
		}
	}()

	return <-errCh
}

// listLocalDevicesHandler fetches the devices present in the node and sends back
func (n *nodeAPIHandler) listLocalDevicesHandler(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		klog.Errorf("couldn't read the request: %v", err)
		writeErrorResponse(w, http.StatusBadRequest, toAPIError(err, "couldn't read the request"))
		return
	}
	// Unmarshal API request
	var req GetDevicesRequest
	if err = json.Unmarshal(data, &req); err != nil {
		klog.Errorf("couldn't parse the request: %v", err)
		writeErrorResponse(w, http.StatusBadRequest, toAPIError(err, "couldn't parse the request"))
		return
	}
	deviceList, err := n.listLocalDevices(context.Background(), req)
	if err != nil {
		klog.Errorf("couldn't list local drives: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, toAPIError(err, "couldn't list local drives"))
		return
	}
	jsonBytes, err := json.Marshal(GetDevicesResponse{
		DeviceInfo: map[NodeName][]Device{
			NodeName(n.nodeID): deviceList,
		},
	})
	if err != nil {
		klog.Errorf("Couldn't marshal the response: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, toAPIError(err, "couldn't marshal the response"))
		return
	}
	writeSuccessResponse(w, jsonBytes)
}

func (n *nodeAPIHandler) listLocalDevices(ctx context.Context, req GetDevicesRequest) ([]Device, error) {
	// Probe the devices from the node
	devices, err := device.ProbeDevices()
	if err != nil {
		return nil, fmt.Errorf("couldn't probe the devices: %v", err)
	}
	// Fetch the local drives from the k8s
	drives, err := n.listDrives(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch the drives: %v", err)
	}
	var deviceList []Device
	for _, drive := range drives {
		matchedDevices, unmatchedDevices := getMatchedDevicesForDrive(&drive, devices)
		switch len(matchedDevices) {
		case 0:
			// Drive which was online before is lost/detached/corrupted now
			if len(req.Statuses) > 0 && !utils.ItemIn(req.Statuses, DeviceStatusUnavailable) {
				break
			}
			deviceName := path.Base(drive.Status.Path)
			if len(req.Drives) > 0 && !utils.ItemIn(req.Drives, Selector(deviceName)) {
				break
			}
			deviceList = append(deviceList, Device{
				Name:        deviceName,
				Size:        uint64(drive.Status.TotalCapacity),
				Model:       drive.Status.ModelNumber,
				Vendor:      drive.Status.Vendor,
				Filesystem:  "xfs",
				Status:      DeviceStatusUnavailable,
				Description: "corrupted/lost drive",
			})
		case 1:
			// Drive detected
			if len(req.Statuses) > 0 && !utils.ItemIn(req.Statuses, DeviceStatusUnavailable) {
				break
			}
			if len(req.Drives) > 0 && !utils.ItemIn(req.Drives, Selector(matchedDevices[0].Name)) {
				break
			}
			deviceList = append(deviceList, Device{
				Name:        matchedDevices[0].Name,
				MajorMinor:  matchedDevices[0].MajorMinor,
				Size:        matchedDevices[0].Size,
				Model:       matchedDevices[0].Model(),
				Vendor:      matchedDevices[0].Vendor(),
				Filesystem:  matchedDevices[0].FSType(),
				Status:      DeviceStatusUnavailable,
				Description: "formatted drive",
				UDevData:    matchedDevices[0].UDevData,
			})
		default:
			// Multiple matches found for the Online drive
			klog.ErrorS(errDuplicateDevice, "drive: ", drive.Name, " devices: ", getDeviceNames(matchedDevices))
		}
		devices = unmatchedDevices
	}
	for _, device := range devices {
		deviceStatus := DeviceStatusAvailable
		isUnavailable, description := device.IsUnavailable()
		if isUnavailable {
			deviceStatus = DeviceStatusUnavailable
		}
		if len(req.Statuses) > 0 && !utils.ItemIn(req.Statuses, deviceStatus) {
			break
		}
		if len(req.Drives) > 0 && !utils.ItemIn(req.Drives, Selector(device.Name)) {
			continue
		}
		deviceList = append(deviceList, Device{
			Name:        device.Name,
			MajorMinor:  device.MajorMinor,
			Size:        device.Size,
			Model:       device.Model(),
			Vendor:      device.Vendor(),
			Filesystem:  device.FSType(),
			Status:      deviceStatus,
			Description: description,
			UDevData:    device.UDevData,
		})
	}
	return deviceList, nil
}

func (n *nodeAPIHandler) listDrives(ctx context.Context) ([]types.Drive, error) {
	labelSelector := fmt.Sprintf("%s=%s", types.NodeLabelKey, types.NewLabelValue(n.nodeID))
	result, err := client.DriveClient().List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// formatLocalDevicesHandler formats the devices present in the node and returns back the status
func (n *nodeAPIHandler) formatLocalDevicesHandler(w http.ResponseWriter, r *http.Request) {
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
	formatDevices, ok := req.FormatInfo[NodeName(n.nodeID)]
	if !ok {
		klog.Errorf("nodename not found in the request. expected %s", n.nodeID)
		writeErrorResponse(w, http.StatusBadRequest, toAPIError(err, "nodename not found in the request"))
		return
	}
	var formatStatusList []FormatDeviceStatus
	var wg sync.WaitGroup
	for _, formatDevice := range formatDevices {
		wg.Add(1)
		go func(device FormatDevice) {
			defer wg.Done()
			formatStatus := n.format(context.Background(), device)
			if formatStatus.Error == "" {
				if err := n.addDrive(context.Background(), device, formatStatus); err != nil {
					klog.Errorf("failed to create a new drive %s for device %s; %w", formatStatus.FSUUID, device.Name, err)
					formatStatus.setErr(err, "failed to create a new drive", formatRetrySuggestion)
				}
			}
			// Incase of error, umount the target so that the request can be retried
			if formatStatus.Error != "" && formatStatus.mountedAt != "" {
				if err := n.safeUnmount(formatStatus.mountedAt, false, false, false); err != nil {
					formatStatus.setErr(
						errwrap.Wrap(err, errors.New(formatStatus.Error)),
						"failed to umount on failure",
						fmt.Sprintf("please umount %s and retry the format request", formatStatus.mountedAt),
					)
				}
			}
			formatStatusList = append(formatStatusList, formatStatus)
		}(formatDevice)
	}
	wg.Wait()
	// Marshal API response
	jsonBytes, err := json.Marshal(FormatDevicesResponse{
		DeviceInfo: map[NodeName][]FormatDeviceStatus{
			NodeName(n.nodeID): formatStatusList,
		},
	})
	if err != nil {
		klog.Errorf("Couldn't marshal the format status: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, toAPIError(err, "couldn't marshal the format status"))
		return
	}
	writeSuccessResponse(w, jsonBytes)
}

func (n *nodeAPIHandler) format(ctx context.Context, device FormatDevice) (formatStatus FormatDeviceStatus) {
	var totalCapacity, freeCapacity uint64
	// Get format lock
	n.getFormatLock(device.MajorMinor).Lock()
	defer n.getFormatLock(device.MajorMinor).Unlock()
	formatStatus.Name = device.Name
	// Check if the udev data is matching
	udevData, err := n.readRunUdevDataByMajorMinor(device.MajorMinor)
	if err != nil {
		klog.V(3).Infof("error while reading udevdata for device %s: %v", device.Name, err)
		formatStatus.setErr(err, "couldn't read the udev data", "")
		return
	}
	if !reflect.DeepEqual(udevData, device.UDevData) {
		klog.V(3).Infof("udev data isn't matching for device %s", device.Name)
		formatStatus.setErr(errUDevDataMismatch, udevDataMismatchReason, formatRetrySuggestion)
		return
	}
	// Check if force is required
	if v, ok := udevData["ID_FS_TYPE"]; ok {
		if v != "" && !device.Force {
			formatStatus.setErr(errForceRequired, fmt.Sprintf("device %s already has a %s fs", device.Name, v), formatRetryWithForceSuggestion)
			return
		}
	}
	// Format the device
	fsuuid := uuid.New().String()
	err = n.makeFS(ctx, device.Path(), fsuuid, device.Force, n.reflinkSupport)
	if err != nil {
		klog.Errorf("failed to format drive %s; %w", device.Name, err)
		formatStatus.setErr(err, "failed to format device", formatRetrySuggestion)
		return
	}
	formatStatus.FSUUID = fsuuid
	// Mount the device
	mountTarget := path.Join(consts.MountRootDir, fsuuid)
	err = n.mountDevice(device.Path(), mountTarget)
	if err != nil {
		klog.Errorf("failed to mount drive %s; %w", device.Name, err)
		formatStatus.setErr(err, "failed to mount device", formatRetrySuggestion)
		return
	}
	formatStatus.mountedAt = mountTarget
	// probe fsinfo to calculate the allocatedcapacity
	_, _, totalCapacity, freeCapacity, err = n.probeXFS(device.Path())
	if err != nil {
		klog.Errorf("failed to probe XFS for device: %s: %s", device.Name, err.Error())
		formatStatus.setErr(err, "failed to probe XFS after formatting", formatRetrySuggestion)
		return
	}
	formatStatus.totalCapacity = totalCapacity
	formatStatus.freeCapacity = freeCapacity
	// Write metadata
	if err := writeFormatMetadata(FormatMetadata{
		FSUUID:      fsuuid,
		FormattedBy: consts.LatestAPIVersion,
	}, path.Join(mountTarget, metaDataPathSuffix)); err != nil {
		klog.Errorf("failed to write metadata for device: %s: %s", device.Name, err.Error())
		formatStatus.setErr(err, "failed to marshal device metadata", formatRetrySuggestion)
		return
	}
	// Create symbolic link
	if err := os.Symlink(mountTarget, path.Join(mountTarget, fsuuid)); err != nil {
		klog.Errorf("failed to create symlink for target %s. device: %s err: %s", mountTarget, device.Name, err.Error())
		formatStatus.setErr(err, "failed to create symlink", formatRetrySuggestion)
	}
	return
}

func (n *nodeAPIHandler) addDrive(ctx context.Context, formatDevice FormatDevice, formatStatus FormatDeviceStatus) error {
	newDrive := drive.NewDrive(formatStatus.FSUUID, types.DriveStatus{
		Path:              formatDevice.Path(),
		TotalCapacity:     int64(formatStatus.totalCapacity),
		AllocatedCapacity: int64(formatStatus.totalCapacity - formatStatus.freeCapacity),
		FreeCapacity:      int64(formatStatus.freeCapacity),
		FSUUID:            formatStatus.FSUUID,
		NodeName:          n.nodeID,
		Status:            apiTypes.DriveStatusOK,
		ModelNumber:       formatDevice.Model(),
		Vendor:            formatDevice.Vendor(),
		AccessTier:        apiTypes.AccessTierUnknown,
		Topology:          n.topology,
	})
	_, err := client.DriveClient().Create(ctx, newDrive, metav1.CreateOptions{})
	return err
}
