#!/usr/bin/env bash
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

ME=$(basename "$0"); export ME
cd "$(dirname "$0")" || exit 255

set -o errexit
set -o nounset
set -o pipefail

declare BUILD_VERSION PODMAN IMG BUILD_DIR

function init() {

    if [ "$#" -eq 1 ]; then
        if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
            cat <<EOF
USAGE:
  ${ME} [VERSION]
EXAMPLES:
  # Release with the latest release tag of the upstream
  $ ${ME}

  # Release the image with tag 0.10.0
  $ ${ME} 0.10.0
EOF
            exit 255
        fi
    fi

    if  [ $# -eq 0 ]; then
        BUILD_VERSION=$(curl -sfL https://api.github.com/repos/kubernetes-csi/external-health-monitor/releases/latest | awk '/tag_name/ { print substr($2, 3, length($2)-4) }')
    else
        BUILD_VERSION="${1/v/}"
    fi

    IMG="quay.io/minio/csi-external-health-monitor-controller:v${BUILD_VERSION}"

    if which podman >/dev/null 2>&1; then
        PODMAN=podman
    elif which docker >/dev/null 2>&1; then
        PODMAN=docker
    else
        echo "no podman or docker found; please install"
        exit 255
    fi

    if "${PODMAN}" pull --quiet "${IMG}" >/dev/null 2>&1; then
        echo "image ${IMG} is already pushed"
        exit 0
    fi

    BUILD_DIR="$(mktemp -d -p "${PWD}" health-monitor-build.XXXXXXXX)"
}

function main() {
    curl --silent --location --insecure --fail "https://github.com/kubernetes-csi/external-health-monitor/archive/refs/tags/v${BUILD_VERSION}.tar.gz" | tar -zxf -
    cd "external-health-monitor-${BUILD_VERSION}" || return
    make build
    "${PODMAN}" buildx build --platform linux/amd64 --tag "${IMG}" -f - . <<EOF
FROM registry.access.redhat.com/ubi8/ubi-minimal:8.8
LABEL maintainers="dev@min.io"
LABEL description="Rebuild of CSI External Health Monitor Controller for Red Hat Marketplace"
COPY ./bin/csi-external-health-monitor-controller csi-external-health-monitor-controller
COPY ./LICENSE /licenses/LICENSE
ENTRYPOINT ["/csi-external-health-monitor-controller"]
EOF
    "${PODMAN}" push "${IMG}"
    cd - || return
}

init "$@"
cd "${BUILD_DIR}" || exit 255
main
cd - || exit 255
rm -fr "${BUILD_DIR}"
