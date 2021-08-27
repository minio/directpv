#!/usr/bin/env bash
#
# This file is part of MinIO Direct CSI
# Copyright (c) 2021 MinIO, Inc.
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

# Enable tracing if set.
[ -n "$BASH_XTRACEFD" ] && set -ex

source "${SCRIPT_DIR}/common.sh"

function test_build() {
    DIRECT_CSI_CLIENT=./kubectl-direct_csi
    DIRECT_CSI_VERSION="$BUILD_VERSION"
    install_directcsi
    check_drives
    deploy_minio
    uninstall_minio
    uninstall_directcsi
    # Check uninstall succeeds even if direct-csi is completely gone.
    "${DIRECT_CSI_CLIENT}" uninstall --crd --force
}

function do_upgrade_test() {
    wget --quiet --output-document=kubectl-direct_csi_1.3.6 https://github.com/minio/direct-csi/releases/download/v1.3.6/kubectl-direct_csi_1.3.6_linux_amd64
    chmod a+x kubectl-direct_csi_1.3.6

    # unmount all direct-csi mounts of previous installation if any.
    mount | awk '/direct-csi/ {print $3}' | xargs sudo umount -fl

    DIRECT_CSI_CLIENT=./kubectl-direct_csi_1.3.6
    DIRECT_CSI_VERSION="v1.3.6"
    install_directcsi
    check_drives
    deploy_minio

    declare -A volumes
    for volume in $("${DIRECT_CSI_CLIENT}" volumes list | awk '{print $1}' ); do
        volumes["${volume}"]=
    done

    "${DIRECT_CSI_CLIENT}" uninstall
    pending=7
    while [[ $pending -gt 3 ]]; do # webhook uninstallation is not supported in v1.3.6
        echo "$ME: waiting for ${pending} direct-csi pods to go down"
        sleep ${pending}
        pending=$(kubectl get pods --field-selector=status.phase=Running --no-headers --namespace=direct-csi-min-io | wc -l)
    done

    # Show output for manual debugging.
    kubectl get pods -n direct-csi-min-io

    DIRECT_CSI_CLIENT=./kubectl-direct_csi
    DIRECT_CSI_VERSION="${BUILD_VERSION}"
    install_directcsi

    # Show output for manual debugging.
    "${DIRECT_CSI_CLIENT}" drives list --all -o wide

    check_drives_state InUse

    # Show output for manual debugging.
    "${DIRECT_CSI_CLIENT}" volumes list -o wide

    for volume in $("${DIRECT_CSI_CLIENT}" volumes list | awk '{print $1}' ); do
        if [[ ! ${volumes[${volume}]+_} ]]; then
            echo "$ME: ${volume} not found after upgrade"
            return 1
        fi
    done

    uninstall_minio
    uninstall_directcsi
}

echo "$ME: Setup environment"
setup_lvm
setup_luks

echo "$ME: Run build test"
test_build

echo "$ME: Run upgrade test"
do_upgrade_test
