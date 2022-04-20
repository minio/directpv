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

set -e

SCRIPT_ROOT="$( cd "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
GID=$(id -g)

docker_run=( docker run -u "${UID}:${GID}" -e HOME=/go/home \
                    -v "${SCRIPT_ROOT}:/go/src/github.com/minio/directpv" \
                    -w /go/src/github.com/minio/directpv \
                    --entrypoint hack/build-without-docker.sh golang:1.17.8 )
[ -t 1 ] && docker_run+=( --interactive --tty )

"${docker_run[@]}"
