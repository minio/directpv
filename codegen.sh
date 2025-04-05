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

echo "Installing code generators ..."
go install -v \
   k8s.io/code-generator/cmd/deepcopy-gen@v0.29.15 \
   k8s.io/code-generator/cmd/client-gen@v0.29.15 \
   k8s.io/code-generator/cmd/conversion-gen@v0.29.15
go install -v k8s.io/kube-openapi/cmd/openapi-gen@v0.0.0-20250318190949-c8a335a9a2ff
go install -v sigs.k8s.io/controller-tools/cmd/controller-gen@v0.17.3

cd "$(dirname "$0")"

REPOSITORY=github.com/minio/directpv

# Remove old generated code
rm -rf pkg/clientset

find . -name "*.go.in" | while read -r infile; do
    outfile="${infile%.in}"
    sed -e "s:{VERSION}:${VERSIONS[-1]}:g" -e "s:{CAPSVERSION}:${VERSIONS[-1]^}:g" \
        "${infile}" > "${outfile}"
    case "${OSTYPE}" in
        darwin*) sed -i '' -e '/^package/i \
// AUTO GENERATED CODE. DO NOT EDIT.\
' "${outfile}" ;;
        *) sed -i -e '/^package/i // AUTO GENERATED CODE. DO NOT EDIT.\n' "${outfile}" ;;
    esac
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
        --output-file openapi_generated.go \
        --output-dir "pkg/apis/directpv.min.io/${version}" \
        --output-pkg "${repo}" \
        "${repo}"
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
controller-gen crd:crdVersions=v1 paths=./... output:dir=pkg/admin/installer
rm -f pkg/admin/installer/direct.csi.min.io_directcsidrives.yaml pkg/admin/installer/direct.csi.min.io_directcsivolumes.yaml

echo "Running conversion-gen ..."
conversion-gen \
    --go-header-file boilerplate.go.txt \
    --input-dirs "${VERSIONS[-1]/#/$REPOSITORY/pkg/apis/directpv.min.io/}" \
    --output-package "${REPOSITORY}/pkg/"
