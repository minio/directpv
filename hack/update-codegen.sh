#!/usr/bin/env bash
#
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

set -o errexit
set -o nounset
set -o pipefail

export PATH="$PATH:$GOPATH/bin"

function install_code_generator() {
    if [ ! -x "$GOPATH/bin/deepcopy-gen" ]; then
        go install -v k8s.io/code-generator/cmd/deepcopy-gen@latest
    fi

    if [ ! -x "$GOPATH/bin/openapi-gen" ]; then
        go install -v k8s.io/code-generator/cmd/openapi-gen@latest
    fi

    if [ ! -x "$GOPATH/bin/client-gen" ]; then
        go install -v k8s.io/code-generator/cmd/client-gen@latest
    fi

    if [ ! -x "$GOPATH/bin/conversion-gen" ]; then
        go install -v k8s.io/code-generator/cmd/conversion-gen@latest
    fi
}

function install_controller_tools() {
    if [ ! -x "$GOPATH/bin/controller-gen" ]; then
        go install -v sigs.k8s.io/controller-tools/cmd/controller-gen@latest
    fi
}

install_code_generator
install_controller_tools

REPOSITORY=github.com/minio/direct-csi
SCRIPT_ROOT=$(cd "$(dirname "$0")"; pwd -P)
PROJECT_ROOT=$(cd "${SCRIPT_ROOT}/.."; pwd -P)

# Remove old generated code
rm -rf "${PROJECT_ROOT}/config/crd" "${PROJECT_ROOT}/pkg/clientset"

versions=(v1alpha1 v1beta1 v1beta2)

# Prefix ${REPOSITORY}/pkg/apis/direct.csi.min.io/ to each versions.
arr=("${versions[@]/#/$REPOSITORY/pkg/apis/direct.csi.min.io/}")

# Join array elements with ",".
input_dirs=$(IFS=,; echo "${arr[*]}")

# deepcopy
deepcopy-gen \
    --go-header-file "${SCRIPT_ROOT}/boilerplate.go.txt" \
    --input-dirs "${input_dirs}" \
    --output-package "${REPOSITORY}/pkg/"

for version in "${versions[@]}"; do
    repo="${REPOSITORY}/pkg/apis/direct.csi.min.io/${version}"

    # openapi
    openapi-gen \
        --go-header-file "${SCRIPT_ROOT}/boilerplate.go.txt" \
        --input-dirs "${repo}" \
        --output-package "${repo}"

    # client
    client-gen \
        --go-header-file "${SCRIPT_ROOT}/boilerplate.go.txt" \
        --input-dirs "${repo}" \
        --output-package "${REPOSITORY}/pkg/" \
        --fake-clientset \
        --clientset-name clientset \
        --input "direct.csi.min.io/${version}" \
        --input-base "${REPOSITORY}/pkg/apis"

done

# crd
controller-gen crd:crdVersions=v1 paths=./...

# conversion
conversion-gen \
    --go-header-file "${SCRIPT_ROOT}/boilerplate.go.txt" \
    --input-dirs "${input_dirs}" \
    --output-package "${REPOSITORY}/pkg/" 
