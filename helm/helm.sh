#!/bin/bash
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

set -o errexit
set -o nounset
set -o pipefail

declare BUILD_VERSION KUBECTL_DIRECTPV

function init() {
    if [ "$#" -ne 1 ]; then
        cat <<EOF
USAGE:
  helm.sh <VERSION>

EXAMPLE:
  $ helm.sh 4.0.7
EOF
        exit 255
    fi

    # assign after trimming 'v'
    BUILD_VERSION="${1/v/}"

    KUBECTL_DIRECTPV="./kubectl-directpv_${BUILD_VERSION}_$(go env GOOS)_$(go env GOARCH)"
    if which "kubectl-directpv_${BUILD_VERSION}_$(go env GOOS)_$(go env GOARCH)" >/dev/null 2>&1; then
        KUBECTL_DIRECTPV="kubectl-directpv_${BUILD_VERSION}_$(go env GOOS)_$(go env GOARCH)"
    elif [ ! -f "${KUBECTL_DIRECTPV}" ]; then
        echo "Downloading required kubectl-directpv"
        curl --silent --location --insecure --fail --output "${KUBECTL_DIRECTPV}" "https://github.com/minio/directpv/releases/download/v${BUILD_VERSION}/${KUBECTL_DIRECTPV:2}"
        chmod a+x "${KUBECTL_DIRECTPV}"
    fi
}

function main() {
    mkdir -p "./helm-charts/directpv-chart/templates"
    "${KUBECTL_DIRECTPV}" install -o yaml | sed -e 's/^---/~~~/g' | awk '{f="file" NR; print $0 > f}' RS='~~~'
    for file in file*; do
        name=$(awk '/^kind:/ { print $NF }' "${file}")
        if [ "${name}" == "CustomResourceDefinition" ]; then
            name=$(awk '/^  name:/ { print $NF }' "${file}")
        fi
        if [ -n "${name}" ]; then
            mv "${file}" "./helm/templates/${name}.yaml"
        fi
    done
}

init "$@"
main "$@"
