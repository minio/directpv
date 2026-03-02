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
go install \
   k8s.io/code-generator/cmd/deepcopy-gen@v0.35.1 \
   k8s.io/code-generator/cmd/client-gen@v0.35.1 \
   k8s.io/code-generator/cmd/conversion-gen@v0.35.1
go install k8s.io/kube-openapi/cmd/openapi-gen@v0.0.0-20260127142750-a19766b6e2d4
go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.20.1

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
    --output-file deepcopy_generated.go \
    "${input_dirs}"

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
    --fake-clientset \
    --output-dir pkg \
    --output-pkg "${REPOSITORY}/pkg" \
    --clientset-name clientset \
    --input "${input_versions}" \
    --input-base "${REPOSITORY}/pkg/apis" \
    "${input_dirs}"

echo "Running controller-gen ..." 
controller-gen crd:crdVersions=v1 paths=./... output:dir=pkg/admin/installer
rm -f pkg/admin/installer/direct.csi.min.io_directcsidrives.yaml pkg/admin/installer/direct.csi.min.io_directcsivolumes.yaml

echo "Running conversion-gen ..."
conversion-gen \
    --go-header-file boilerplate.go.txt \
    --output-file zz_generated.conversion.go \
    "${VERSIONS[-1]/#/$REPOSITORY/pkg/apis/directpv.min.io/}"
