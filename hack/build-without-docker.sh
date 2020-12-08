#!/usr/bin/env bash

# This file is part of MinIO Direct CSI
# Copyright (c) 2020 MinIO, Inc.
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

set -e

SCRIPT_ROOT="$( cd "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
REPOSITORY=github.com/minio/direct-csi
CSI_VERSION=$(git describe --tags --always --dirty)

export CGO_ENABLED=0

"${SCRIPT_ROOT}/update-codegen.sh"
go build -tags "osusergo netgo static_build" -ldflags="-X ${REPOSITORY}/cmd.Version=${CSI_VERSION} -extldflags=-static"
go build -tags "plugin" -ldflags="-X ${REPOSITORY}/cmd.Version=${CSI_VERSION} -extldflags=-static" -o kubectl-directcsi
