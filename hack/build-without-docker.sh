#!/usr/bin/env bash

# This file is part of MinIO Direct CSI
# Copyright (c) 2021 MinIO, Inc.
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.

set -ex

SCRIPT_ROOT="$( cd "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
REPOSITORY=github.com/minio/direct-csi
CSI_VERSION=$(git describe --tags --always --dirty)

export CGO_ENABLED=0

"${SCRIPT_ROOT}/update-codegen.sh"

go get -u github.com/jteeuwen/go-bindata/...
go-bindata -o ${SCRIPT_ROOT}/../cmd/kubectl-direct_csi/crd_bindata.go ${SCRIPT_ROOT}/../config/crd/...
gofmt -s -w cmd/kubectl-direct_csi/crd_bindata.go

go build -tags "osusergo netgo static_build" -ldflags="-X main.Version=${CSI_VERSION} -extldflags=-static" ${REPOSITORY}/cmd/direct-csi
go build -tags "osusergo netgo static_build" -ldflags="-X main.Version=${CSI_VERSION} -extldflags=-static" ${REPOSITORY}/cmd/kubectl-direct_csi
