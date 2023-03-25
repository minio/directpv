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

package installer

import "github.com/minio/directpv/pkg/consts"

const (
	// UnixCSIEndpoint is csi drive control socket.
	UnixCSIEndpoint = "unix:///csi/csi.sock"

	namespace                = consts.AppName
	healthZContainerPortName = "healthz"
	healthZContainerPort     = 9898
	volumePathSysDir         = "/sys"
	volumeNameSocketDir      = "socket-dir"
	socketDir                = "/csi"
	selectorKey              = "selector." + consts.GroupName
	kubeNodeNameEnvVarName   = "KUBE_NODE_NAME"
	csiEndpointEnvVarName    = "CSI_ENDPOINT"
	kubeletDirPath           = "/var/lib/kubelet"
	pluginName               = "kubectl-" + consts.AppName
	selectorValueEnabled     = "enabled"
	serviceSelector          = "selector." + consts.GroupName + ".service"
	healthZContainerPortPath = "/healthz"
	logLevel                 = 3
	createdByLabel           = "created-by"
	localhost                = "localhost"
)
