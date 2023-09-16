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

ME=$(basename "$0"); export ME
cd "$(dirname "$0")" || exit 255

set -o errexit
set -o nounset
set -o pipefail

declare BUILD_VERSION KUBECTL_DIRECTPV PODMAN OPERATOR_SDK KUSTOMIZE

function init() {
    if [ "$#" -ne 1 ]; then
        cat <<EOF
USAGE:
  ${ME} <VERSION>

EXAMPLE:
  $ ${ME} 4.0.8
EOF
        exit 255
    fi

    # assign after trimming 'v'
    BUILD_VERSION="${1/v/}"

    kubectl_directpv="kubectl-directpv_${BUILD_VERSION}_$(go env GOOS)_$(go env GOARCH)"
    KUBECTL_DIRECTPV="${PWD}/${kubectl_directpv}"
    if [ ! -f "${KUBECTL_DIRECTPV}" ]; then
        echo "Downloading ${kubectl_directpv}"
        curl --silent --location --insecure --fail --output "${kubectl_directpv}" "https://github.com/minio/directpv/releases/download/v${BUILD_VERSION}/${kubectl_directpv}"
        chmod a+x "${kubectl_directpv}"
    fi

    if which podman >/dev/null 2>&1; then
        PODMAN=podman
    elif which docker >/dev/null 2>&1; then
        PODMAN=docker
    else
        echo "no podman or docker found; please install"
        exit 255
    fi

    if which operator-sdk >/dev/null 2>&1; then
        OPERATOR_SDK=operator-sdk
    fi
    if [ -f "operator-sdk" ]; then
        OPERATOR_SDK="${PWD}/operator-sdk"
    fi
    if [ -z "${OPERATOR_SDK=}" ]; then
        OPERATOR_SDK="${PWD}/operator-sdk"
        echo "Downloading operator-sdk"
        release=$(curl -sfL "https://api.github.com/repos/operator-framework/operator-sdk/releases/latest" | awk '/tag_name/ { print substr($2, 2, length($2)-3) }')
        curl -sfLo operator-sdk "https://github.com/operator-framework/operator-sdk/releases/download/${release}/operator-sdk_$(go env GOOS)_$(go env GOARCH)"
        chmod a+x operator-sdk
    fi

    if which kustomize >/dev/null 2>&1; then
        KUSTOMIZE=kustomize
    fi
    if [ -f ./kustomize ]; then
        KUSTOMIZE="${PWD}/kustomize"
    fi
    if [ -z "${KUSTOMIZE=}" ]; then
        KUSTOMIZE="${PWD}/kustomize"
        echo "Downloading kustomize"
        release=$(curl -sfL "https://api.github.com/repos/kubernetes-sigs/kustomize/releases/latest" | awk '/tag_name/ { print substr($2, 12, length($2)-13) }')
        curl --silent --location --insecure --fail "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2F${release}/kustomize_${release}_$(go env GOOS)_$(go env GOARCH).tar.gz" | tar -zxf -
    fi
}

function git_commit() {
    case "$(git describe --always --dirty)" in
        *dirty)
            git commit --quiet --all -m "$*"
            git push --quiet
            ;;
    esac
}

function update_charts() {
    sed -i -e "/^appVersion: /c appVersion: \"${BUILD_VERSION}\"" "${PWD}/operator/helm-charts/directpv-chart/Chart.yaml"
    "${KUBECTL_DIRECTPV}" install -o yaml | sed -e 's/^---/~~~/g' | awk '{f="file" NR; print $0 > f}' RS='~~~'
    for file in file*; do
        name=$(awk '/^kind:/ { print $NF }' "${file}")
        if [ "${name}" == "CustomResourceDefinition" ]; then
            name=$(awk '/^  name:/ { print $NF }' "${file}")
        fi
        if [ -n "${name}" ]; then
            mv "${file}" "${PWD}/operator/helm-charts/directpv-chart/templates/${name}.yaml"
        else
            rm "${file}"
        fi
    done

    git_commit "Update Helm charts for v${BUILD_VERSION}"
}

function make_release() {
    export IMAGE_TAG_BASE=quay.io/minio/directpv-operator
    export IMG="${IMAGE_TAG_BASE}:${BUILD_VERSION}"
    # We need RedHat annotations, if you set overwrite to true, they will be
    # removed, please set it to true only when needed for something new but
    # right after, put RedHat annotations back in place. For example:
    # com.redhat.openshift.versions: v4.8-v4.13
    export BUNDLE_GEN_FLAGS="-q --overwrite=false --version ${BUILD_VERSION} --package minio-directpv-operator-rhmp"
    export BUNDLE_IMG="${IMAGE_TAG_BASE}-bundle:v${BUILD_VERSION}"

    cd operator

    "${PODMAN}" buildx build --platform linux/amd64 --tag "${IMG}" .
    "${PODMAN}" push "${IMG}"
    git_commit "Update operator for v${BUILD_VERSION}"

    "${OPERATOR_SDK}" generate kustomize manifests --quiet --package minio-directpv-operator-rhmp
    (cd config/manager && "${KUSTOMIZE}" edit set image controller="${IMG}")
    # shellcheck disable=SC2086
    "${KUSTOMIZE}" build config/manifests | "${OPERATOR_SDK}" generate bundle ${BUNDLE_GEN_FLAGS}
    "${OPERATOR_SDK}" bundle validate ./bundle
    git_commit "Update operator bundle for v${BUILD_VERSION}"

    "${PODMAN}" build -f bundle.Dockerfile --tag "${BUNDLE_IMG}" .
    "${PODMAN}" push "${BUNDLE_IMG}"
    git_commit "Update operator bundle image for v${BUILD_VERSION}"

    cd -
}

function main() {
    update_charts
    make_release
}

init "$@"
main "$@"
