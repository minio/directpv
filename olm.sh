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

declare BUILD_VERSION IMAGE_HASH KUBECTL_DIRECTPV OPERATOR_SDK YQ
PACKAGE=minio-directpv-operator-rhmp

function get_operator_sdk() {
    curl --silent --location --insecure --fail https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/operator-sdk/4.13.1/operator-sdk-v1.28.0-ocp-linux-x86_64.tar.gz | tar --strip-component 2 -zxf - ./x86_64/operator-sdk
}

function get_yq() {
    release=$(curl --silent --location --insecure --fail "https://api.github.com/repos/mikefarah/yq/releases/latest" | awk '/tag_name/ { print substr($2, 3, length($2)-4) }')
    curl --silent --location --insecure --fail --output yq "https://github.com/mikefarah/yq/releases/download/v${release}/yq_linux_amd64"
    chmod a+x yq
}

function get_kubectl_directpv() {
    [ -f "${KUBECTL_DIRECTPV}" ] && return 0
    curl --silent --location --insecure --fail --output "${KUBECTL_DIRECTPV}" "https://github.com/minio/directpv/releases/download/v${BUILD_VERSION}/${KUBECTL_DIRECTPV:2}"
    chmod a+x "${KUBECTL_DIRECTPV}"
}

function init() {
    if [ "$#" -ne 2 ]; then
        cat <<EOF
USAGE:
  olm.sh <VERSION> <IMAGE-HASH>

EXAMPLE:
  $ olm.sh 4.0.6 sha256:53610c35d42971ad57ce8491dbcab3b95395a68820d9024a1298b9f653802688
EOF
        exit 255
    fi

    # assign after trimming 'v'
    BUILD_VERSION="${1/v/}"
    IMAGE_HASH="$2"
    KUBECTL_DIRECTPV="./kubectl-directpv_${BUILD_VERSION}_$(go env GOOS)_$(go env GOARCH)"

    echo "Downloading required tools"

    OPERATOR_SDK=./operator-sdk
    if which operator-sdk >/dev/null 2>&1; then
        OPERATOR_SDK=operator-sdk
    elif [ ! -f ./operator-sdk ]; then
        if ! get_operator_sdk; then
            echo "unable to get operator-sdk; exiting..."
            exit 1
        fi
    fi

    YQ=./yq
    if which yq >/dev/null 2>&1; then
        YQ=yq
    elif [ ! -f ./yq ]; then
        if ! get_yq; then
            echo "unable to get yq; exiting..."
            exit 1
        fi
    fi

    get_kubectl_directpv
}

function main() {
    mkdir -p "resources/base/${BUILD_VERSION}"
    "${KUBECTL_DIRECTPV}" install --openshift -o yaml > "resources/base/${BUILD_VERSION}/directpv.yaml"

    "${OPERATOR_SDK}" generate bundle \
                      --package "${PACKAGE}" \
                      --version "${BUILD_VERSION}" \
                      --deploy-dir "resources/base/${BUILD_VERSION}" \
                      --manifests \
                      --metadata \
                      --output-dir "bundles/redhat-marketplace/${BUILD_VERSION}" \
                      --channels stable \
                      --overwrite

    # Annotations to specify OCP versions compatibility.
    "${YQ}" -i '.annotations."com.redhat.openshift.versions" |= "v4.10-v4.13"' "bundles/redhat-marketplace/${BUILD_VERSION}/metadata/annotations.yaml"

    cluster_srv_ver_yaml="bundles/redhat-marketplace/${BUILD_VERSION}/manifests/${PACKAGE}.clusterserviceversion.yaml"

    # Add needed annotations for redhat marketplace
    "${YQ}" -i '.metadata.annotations."marketplace.openshift.io/remote-workflow" |= "https://marketplace.redhat.com/en-us/operators/minio-directpv-operator-rhmp/pricing?utm_source=openshift_console"' "${cluster_srv_ver_yaml}"
    "${YQ}" -i '.metadata.annotations."marketplace.openshift.io/support-workflow" |= "https://marketplace.redhat.com/en-us/operators/minio-directpv-operator-rhmp/support?utm_source=openshift_console"' "${cluster_srv_ver_yaml}"

    # Use DirectPV image and hash from Red Hat registry
    sed -i -e "s/quay.io\/minio\/directpv:v.*/registry.connect.redhat.com\/minio\/directpv@${IMAGE_HASH}/g" "${cluster_srv_ver_yaml}"

    # To add icon right after bundle creation
    icon_base64data=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
    "${YQ}" -i ".spec.icon[0].base64data |= (\"${icon_base64data}\")" "${cluster_srv_ver_yaml}"
    "${YQ}" -i '.spec.icon[0].mediatype |= ("image/png")' "${cluster_srv_ver_yaml}"

    count=$("${YQ}" '.spec.install.spec.deployments[0].spec.template.spec.containers | length' "${cluster_srv_ver_yaml}")
    for (( i = 0; i < count; ++i )); do
        image=$("${YQ}" ".spec.install.spec.deployments[0].spec.template.spec.containers[$i].image" "${cluster_srv_ver_yaml}")
        "${YQ}" -i ".spec.relatedImages[$i].image |= (\"${image}\")" "${cluster_srv_ver_yaml}"

        name=$("${YQ}" ".spec.install.spec.deployments[0].spec.template.spec.containers[$i].name" "${cluster_srv_ver_yaml}")
        "${YQ}" -i ".spec.relatedImages[$i].name |= \"${name}\"" "${cluster_srv_ver_yaml}"
    done
}

init "$@"
main "$@"
