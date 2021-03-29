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

set -e

SCRIPT_ROOT="$( cd "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
docker run \
    -u $(id -u ${USER}):$(id -g ${USER}) \
    -e HOME=/go/home \
    -v "${SCRIPT_ROOT}:/go/src/github.com/minio/direct-csi" \
    -w /go/src/github.com/minio/direct-csi \
    --entrypoint hack/build-without-docker.sh golang:1.14
