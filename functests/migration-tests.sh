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

source "${SCRIPT_DIR}/common.sh"

sed -e s:directpv-min-io:direct-csi-min-io:g -e s:directpv.min.io:direct.csi.min.io:g functests/minio.yaml > functests/directcsi-minio.yaml

# usage: migrate_test <plugin> <pod-count>
function migrate_test() {
    directcsi_client="$1"
    pod_count="$2"
    
    setup_lvm
    setup_luks

    install_directcsi "${directcsi_client}" "${pod_count}"

    "${directcsi_client}" drives format --all --force

    deploy_minio functests/directcsi-minio.yaml

    uninstall_directcsi "${directcsi_client}" "${pod_count}"

    export DIRECTPV_CLIENT=./kubectl-directpv
    install_directpv 9

    delete_minio functests/directcsi-minio.yaml

    deploy_minio functests/directcsi-minio.yaml

    uninstall_minio functests/directcsi-minio.yaml

    force_uninstall_directcsi "${directcsi_client}"

    remove_drives
    uninstall_directpv 9

    mount | awk '/direct|pvc-/ {print $3}' | xargs sudo umount -fl

    grep -v docker /proc/mounts

    remove_luks
    remove_lvm
}

echo "-------------- Migration test on DirectCSI ${LEGACY_VERSION} -----------------------"
wget --quiet "https://github.com/minio/directpv/releases/download/${LEGACY_VERSION}/${LEGACY_FILE}"
chmod a+x "${LEGACY_FILE}"
migrate_test "./${LEGACY_FILE}" 4
