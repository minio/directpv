#!/usr/bin/env bash
#
# This file is part of MinIO DirectPV
# Copyright (c) 2022 MinIO, Inc.
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

if [ "$#" -ne 1 ]; then
    echo "usage: migration-tests.sh <DIRECTCSI-VERSION>"
    exit 255
fi

LEGACY_VERSION="$1"
LEGACY_FILE="kubectl-direct_csi_${LEGACY_VERSION:1}_linux_amd64"

set -ex

# shellcheck source=/dev/null
source "common.sh"

sed -e s:directpv-min-io:direct-csi-min-io:g -e s:directpv.min.io:direct.csi.min.io:g minio.yaml > directcsi-minio.yaml

# usage: migrate_test <plugin> <legacy-pod-count>
function migrate_test() {
    directcsi_client="$1"
    legacy_pod_count=$(( $2 + ACTIVE_NODES - 1 ))
    pod_count=$(( 6 + ACTIVE_NODES * 2 ))

    setup_lvm
    setup_luks

    install_directcsi "${directcsi_client}" "${legacy_pod_count}"

    "${directcsi_client}" drives format --all --force

    deploy_minio directcsi-minio.yaml

    uninstall_directcsi "${directcsi_client}" "${legacy_pod_count}"

    install_directpv "${DIRECTPV_DIR}/kubectl-directpv" "${pod_count}"

    delete_minio directcsi-minio.yaml

    deploy_minio directcsi-minio.yaml

    uninstall_minio "${DIRECTPV_DIR}/kubectl-directpv" directcsi-minio.yaml

    force_uninstall_directcsi "${directcsi_client}"

    remove_drives "${DIRECTPV_DIR}/kubectl-directpv"
    uninstall_directpv "${DIRECTPV_DIR}/kubectl-directpv" "${pod_count}"

    unmount_directpv

    remove_luks
    remove_lvm
}

curl --silent --location --insecure --fail --output "${LEGACY_FILE}" "https://github.com/minio/directpv/releases/download/${LEGACY_VERSION}/${LEGACY_FILE}"
chmod a+x "${LEGACY_FILE}"
migrate_test "./${LEGACY_FILE}" 4
