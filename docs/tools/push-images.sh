#!/usr/bin/env bash
#
# This file is part of MinIO DirectPV
# Copyright (c) 2023 MinIO, Inc.
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

#
# This script pushes DirectPV and its sidecar images to private registry.
#

set -o errexit
set -o nounset
set -o pipefail

declare registry podman

function init() {
    if [ "$#" -ne 1 ]; then
        cat <<EOF
USAGE:
  push-images.sh <REGISTRY>

ARGUMENT:
<REGISTRY>    Image registry without 'http' prefix.

EXAMPLE:
$ push-images.sh example.org/myuser
EOF
        exit 255
    fi
    registry="$1"

    if which podman >/dev/null 2>&1; then
        podman=podman
    elif which docker >/dev/null 2>&1; then
        podman=docker
    else
        echo "no podman or docker found; please install"
        exit 255
    fi
}

# usage: push_image <image>
function push_image() {
    image="$1"
    private_image="${image/quay.io\/minio/$registry}"
    echo "Pushing image ${image}"
    "${podman}" pull --quiet "${image}"
    "${podman}" tag "${image}" "${private_image}"
    "${podman}" push --quiet "${private_image}"
}

function main() {
    push_image "quay.io/minio/csi-node-driver-registrar:v2.12.0-0"
    push_image "quay.io/minio/csi-provisioner:v5.0.2-0"
    push_image "quay.io/minio/csi-provisioner:v2.2.0-go1.18"
    push_image "quay.io/minio/livenessprobe:v2.14.0-0"
    push_image "quay.io/minio/csi-resizer:v1.12.0-0"
    release=$(curl -sfL "https://api.github.com/repos/minio/directpv/releases" | awk '/tag_name/ { print substr($2, 3, length($2)-4) }' | awk 'BEGIN{m = 0; ver=""} /^4\.0\./ { p = substr($1, 5, length($1)); if (0+p > 0+m) {m = p; ver = $1} } END{print ver}')
    push_image "quay.io/minio/directpv:v${release}"
}

init "$@"
main "$@"
