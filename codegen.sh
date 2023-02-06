#!/usr/bin/env bash
#
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

export PATH="$PATH:$GOPATH/bin"

# Must keep versions sorted.
VERSIONS=(v1beta1)

function install_code_generator() {
    if [ ! -x "$GOPATH/bin/deepcopy-gen" ]; then
        go install -v k8s.io/code-generator/cmd/deepcopy-gen@v0.26.1
    fi

    if [ ! -x "$GOPATH/bin/openapi-gen" ]; then
        go install -v k8s.io/code-generator/cmd/openapi-gen@v0.26.1
    fi

    if [ ! -x "$GOPATH/bin/client-gen" ]; then
        go install -v k8s.io/code-generator/cmd/client-gen@v0.26.1
    fi

    if [ ! -x "$GOPATH/bin/conversion-gen" ]; then
        go install -v k8s.io/code-generator/cmd/conversion-gen@v0.26.1
    fi
}

function install_controller_tools() {
    if [ ! -x "$GOPATH/bin/controller-gen" ]; then
        go install -v sigs.k8s.io/controller-tools/cmd/controller-gen@v0.11.2
    fi
}

install_code_generator
install_controller_tools

cd "$(dirname "$0")"

REPOSITORY=github.com/minio/directpv

# Remove old generated code
rm -rf pkg/clientset

find . -name "*.go.in" | while read -r infile; do
    outfile="${infile%.in}"
    sed -e "s:{VERSION}:${VERSIONS[-1]}:g" -e "s:{CAPSVERSION}:${VERSIONS[-1]^}:g" \
        "${infile}" > "${outfile}"
    sed -i -e '/^package/i // AUTO GENERATED CODE. DO NOT EDIT.\n' "${outfile}"
done

# Prefix ${REPOSITORY}/pkg/apis/directpv.min.io to each versions.
arr=("${VERSIONS[@]/#/$REPOSITORY/pkg/apis/directpv.min.io/}")

# Join array elements with ",".
input_dirs=$(IFS=,; echo "${arr[*]}")

echo "Running deepcopy-gen ..."
deepcopy-gen \
    --go-header-file boilerplate.go.txt \
    --input-dirs "${input_dirs}" \
    --output-package "${REPOSITORY}/pkg/"

echo "Running openapi-gen ..."
for version in "${VERSIONS[@]}"; do
    repo="${REPOSITORY}/pkg/apis/directpv.min.io/${version}"
    openapi-gen \
        --go-header-file boilerplate.go.txt \
        --input-dirs "${repo}" \
        --output-package "${repo}"
done

echo "Running client-gen ..." 
# Prefix directpv.min.io/ to each versions.
arr=("${VERSIONS[@]/#/directpv.min.io/}")
# Join array elements with ",".
input_versions=$(IFS=,; echo "${arr[*]}")
client-gen \
    --go-header-file boilerplate.go.txt \
    --input-dirs "${input_dirs}" \
    --output-package "${REPOSITORY}/pkg/" \
    --fake-clientset \
    --clientset-name clientset \
    --input "${input_versions}" \
    --input-base "${REPOSITORY}/pkg/apis"

echo "Running controller-gen ..." 
controller-gen crd:crdVersions=v1 paths=./... output:dir=pkg/installer
rm -f pkg/installer/direct.csi.min.io_directcsidrives.yaml pkg/installer/direct.csi.min.io_directcsivolumes.yaml

echo "Running conversion-gen ..."
conversion-gen \
    --go-header-file boilerplate.go.txt \
    --input-dirs "${VERSIONS[-1]/#/$REPOSITORY/pkg/apis/directpv.min.io/}" \
    --output-package "${REPOSITORY}/pkg/"
