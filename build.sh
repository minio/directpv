#!/usr/bin/env bash
# This file is part of MinIO DirectPV
# Copyright (c) 2021, 2022 MinIO, Inc.
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

set -o errexit
set -o nounset
set -o pipefail

cd "$(dirname "$0")"

./codegen.sh

BUILD_VERSION=$(git describe --tags --always --dirty)

go build -tags "osusergo netgo static_build" \
   -ldflags="-X main.Version=${BUILD_VERSION} -extldflags=-static" \
   github.com/minio/directpv/cmd/directpv

go build -tags "osusergo netgo static_build" \
   -ldflags="-X main.Version=${BUILD_VERSION} -extldflags=-static" \
   github.com/minio/directpv/cmd/kubectl-directpv
