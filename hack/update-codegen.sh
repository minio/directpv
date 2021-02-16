#!/usr/bin/env bash

# This file is part of MinIO Direct CSI
# Copyright (c) 2020 MinIO, Inc.
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

PATH="$PATH:$GOPATH/bin"
SCRIPT_ROOT="$( cd "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
PROJECT_ROOT="${SCRIPT_ROOT}/.."

GO111MODULE=off
go get k8s.io/code-generator/...
go get sigs.k8s.io/controller-tools/cmd/controller-gen

REPOSITORY=github.com/minio/direct-csi

# Remove old generated code
rm -rf "${PROJECT_ROOT}/config/crd"
rm -rf "${PROJECT_ROOT}/pkg/clientset"

versions="v1alpha1 v1beta1"
for version in $versions; do
    repo="${REPOSITORY}/pkg/apis/direct.csi.min.io/${version}"

    # deepcopy
    deepcopy-gen \
        --go-header-file "${SCRIPT_ROOT}/boilerplate.go.txt" \
        --output-package "${REPOSITORY}/pkg/" \
        --input-dirs "${repo}"

    # openapi
    openapi-gen \
        --go-header-file "${SCRIPT_ROOT}/boilerplate.go.txt" \
        --output-package "${repo}" \
        --input-dirs "${repo}"

    # client
    inputDir="direct.csi.min.io/${version}"
    client-gen \
        --fake-clientset \
        --go-header-file  "${SCRIPT_ROOT}/boilerplate.go.txt" \
        --clientset-name clientset \
        --output-package "${REPOSITORY}/pkg/" \
        --input-dirs "${repo}" \
        --input "${inputDir}" \
        --input-base "${REPOSITORY}/pkg/apis"

done

# crd
controller-gen \
    crd:crdVersions=v1 \
    paths=./...

conversion-gen \
        --input-dirs "${REPOSITORY}/pkg/apis/direct.csi.min.io/v1beta1,${REPOSITORY}/pkg/apis/direct.csi.min.io/v1alpha1 " \
        --go-header-file "${SCRIPT_ROOT}/boilerplate.go.txt" \
        --output-package "${REPOSITORY}/pkg/" 