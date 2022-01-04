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

SCRIPT_DIR=$(dirname "$0")
BUILD_VERSION=$(git describe --tags --always --dirty)

"${SCRIPT_DIR}/update-codegen.sh"

export CGO_ENABLED=0

go get -u github.com/jteeuwen/go-bindata/...
go-bindata -pkg installer -o "${SCRIPT_DIR}/../pkg/installer/crd_bindata.go" "${SCRIPT_DIR}/../config/crd/..."
gofmt -s -w "${SCRIPT_DIR}/../pkg/installer/crd_bindata.go"

"${SCRIPT_DIR}/add-license-header.sh"

export GO111MODULE=on

go build -tags "osusergo netgo static_build" \
   -ldflags="-X main.Version=${BUILD_VERSION} -extldflags=-static" \
   github.com/minio/directpv/cmd/directpv

go build -tags "osusergo netgo static_build" \
   -ldflags="-X main.Version=${BUILD_VERSION} -extldflags=-static" \
   github.com/minio/directpv/cmd/kubectl-directpv

go build -tags "osusergo netgo static_build" \
   -ldflags="-X main.Version=${BUILD_VERSION} -extldflags=-static" \
   github.com/minio/directpv/cmd/kubectl-direct_csi
