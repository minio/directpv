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

    cd operator

    "${PODMAN}" buildx build --platform linux/amd64 --tag "${IMG}" .
    "${PODMAN}" push "${IMG}"
    git_commit "Update operator for v${BUILD_VERSION}"

    # Package is intended for certified operators only not for rhmp anymore.
    "${OPERATOR_SDK}" generate kustomize manifests --quiet --package minio-directpv-operator
}

### subsequent_steps() objective:
#
# To add additional things once images has been pushed in make_release()
#
### Reasoning:
#
# We can't add these additional things until images are pushed.
# This is why we need this function, similar to what we are doing at
# Operator repo: https://github.com/minio/operator/blob/master/olm-post-script.sh
#
function subsequent_steps() {
    export IMAGE_TAG_BASE=quay.io/minio/directpv-operator
    # Package is intended for certified operators only not for rhmp anymore.
    export BUNDLE_GEN_FLAGS="-q --overwrite --version ${BUILD_VERSION} --package minio-directpv-operator"
    export BUNDLE_IMG="${IMAGE_TAG_BASE}-bundle:v${BUILD_VERSION}"

    "${PODMAN}" pull "${IMAGE_TAG_BASE}":"${BUILD_VERSION}"
    OPERATOR_DIGEST=$("${PODMAN}" image list quay.io/minio/directpv-operator --digests | grep sha | awk -F ' ' '{print $3}')
    export OPERATOR_DIGEST
    export DIGEST="${IMAGE_TAG_BASE}@${OPERATOR_DIGEST}"

    # Controller image, should be in SHA Digest form for certification to pass test:
    # verify-pinned-digest where all your container images should use SHA digests instead of tags.
    # Example:
    # (cd config/manager && kustomize edit set image controller=quay.io/minio/directpv-operator@sha256:04fec2fbd0d17f449a17c0f509b359c18d6c662e0a22e84cd625b538ca2a1af2)
    (cd config/manager && "${KUSTOMIZE}" edit set image controller="${DIGEST}")
    # shellcheck disable=SC2086
    "${KUSTOMIZE}" build config/manifests | "${OPERATOR_SDK}" generate bundle ${BUNDLE_GEN_FLAGS}
    # Since above line overwrites our redhat annotation,
    # it will be added back:
    {
        echo "  # Annotations to specify OCP versions compatibility."
        echo "  com.redhat.openshift.versions: v4.8-v4.13"
    } >> bundle/metadata/annotations.yaml
    "${OPERATOR_SDK}" bundle validate ./bundle
    git_commit "Update operator bundle for v${BUILD_VERSION}"

    "${PODMAN}" build -f bundle.Dockerfile --tag "${BUNDLE_IMG}" .
    "${PODMAN}" push "${BUNDLE_IMG}"
    git_commit "Update operator bundle image for v${BUILD_VERSION}"

    cd -

    "${PODMAN}" pull gcr.io/kubebuilder/kube-rbac-proxy:v0.13.1
    PROXY_DIGEST=$("${PODMAN}" image list gcr.io/kubebuilder/kube-rbac-proxy --digests | grep sha | awk -F ' ' '{print $3}')

    ### relatedImages: Field needed by RedHat Certification.
    # kind: ClusterServiceVersion
    # spec:
    #   relatedImages:
    #     - image: gcr.io/kubebuilder/kube-rbac-proxy@sha256:<digest>
    #       name: kube-rbac-proxy
    #     - image: quay.io/minio/directpv-operator@sha256:<digest>
    #       name: manager
    #
    # Add relatedImages to CSV
    yq -i ".spec.relatedImages |= []" ./operator/bundle/manifests/minio-directpv-operator.clusterserviceversion.yaml
    # Add kube-rbac-proxy image
    yq -i ".spec.relatedImages[0] = {\"image\": \"gcr.io/kubebuilder/kube-rbac-proxy@${PROXY_DIGEST}\", \"name\": \"kube-rbac-proxy\"}" ./operator/bundle/manifests/minio-directpv-operator.clusterserviceversion.yaml
    # Add manager image
    yq -i ".spec.relatedImages[1] = {\"image\": \"quay.io/minio/directpv-operator@${OPERATOR_DIGEST}\", \"name\": \"manager\"}" ./operator/bundle/manifests/minio-directpv-operator.clusterserviceversion.yaml
}

function main() {
    update_charts
    make_release
    subsequent_steps
}

init "$@"
main "$@"
