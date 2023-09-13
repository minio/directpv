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

declare BUILD_VERSION

function init() {
    if [ "$#" -ne 1 ]; then
        cat <<EOF
USAGE:
  generate.sh <VERSION>

EXAMPLE:
  $ generate.sh 4.0.7
EOF
        exit 255
    fi

    # assign after trimming 'v'
    BUILD_VERSION="${1/v/}"
    IMAGE_TAG_BASE=quay.io/minio/directpv-operator
    IMG="$IMAGE_TAG_BASE:$BUILD_VERSION"
    PACKAGE=minio-directpv-operator-rhmp
    BUNDLE_GEN_FLAGS="-q --overwrite --version ${BUILD_VERSION}"
    BUNDLE_GEN_FLAGS="${BUNDLE_GEN_FLAGS} --package ${PACKAGE}"
    BUNDLE_IMG="${IMAGE_TAG_BASE}-bundle:v${BUILD_VERSION}"

}

function main() {

    # docker-build: Build docker image with the manager.
    docker buildx build --platform linux/amd64 -t "${IMG}" .

    # Ask admin to run this command for you as it requires privileges
    # docker-push: Push docker image with the manager.
    # docker push "${IMG}"

    # bundle: Generate bundle manifests and metadata, then validate generated files.
    operator-sdk generate kustomize manifests -q --package minio-directpv-operator-rhmp
    (cd config/manager && kustomize edit set image controller="$IMG")
    kustomize build config/manifests | operator-sdk generate bundle "$BUNDLE_GEN_FLAGS"
    operator-sdk bundle validate ./bundle

    # bundle-build: Build the bundle image.
    docker build -f bundle.Dockerfile -t "$BUNDLE_IMG" .

    # Ask admin to run this command for you as it requires privileges
    # bundle-push: Push the bundle image.
    # docker push "${BUNDLE_IMG}"

}

init "$@"
main "$@"
