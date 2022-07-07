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

set -ex

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
    wget --quiet --output-document="kubectl-direct_csi_$1" "https://github.com/minio/directpv/releases/download/v$1/kubectl-direct_csi_$1_linux_amd64"
    chmod a+x "kubectl-direct_csi_$1"

    # unmount all direct-csi mounts of previous installation if any.
    mount | awk '/direct-csi/ {print $3}' | xargs sudo umount -fl

    DIRECT_CSI_CLIENT="./kubectl-direct_csi_$1"
    DIRECT_CSI_VERSION="v$1"
    image="direct-csi:${DIRECT_CSI_VERSION}"

    if [ -n "${RHEL7_TEST}" ]; then
        image="direct-csi:${DIRECT_CSI_VERSION}-rhel7"
    fi

    if [[ $1 > "2.0.0" ]]; then
        image="directpv:${DIRECT_CSI_VERSION}"
    fi

    install_directcsi "$image"
    check_drives
    deploy_minio

    declare -A volumes
    for volume in $( "${DIRECT_CSI_CLIENT}" volumes list --status published | awk '{print $1}' ); do
        volumes["${volume}"]=
    done

    # Show output for manual debugging.
    "${DIRECT_CSI_CLIENT}" volumes list

    # Check version compatibility client.
    ./kubectl-direct_csi uninstall

    pending=4
    if [[ $1 == "1.4.6" ]]; then
        pending=7 # conversion webhook pods
    fi
    while [[ $pending -gt 0 ]]; do
        echo "$ME: waiting for ${pending} direct-csi pods to go down"
        sleep ${pending}
        pending=$(kubectl get pods --field-selector=status.phase=Running --no-headers --namespace=direct-csi-min-io | wc -l)
    done

    # Show output for manual debugging.
    kubectl get pods -n direct-csi-min-io

    
    wait_namespace_removal
    

    export DIRECT_CSI_CLIENT=./kubectl-direct_csi
    export DIRECT_CSI_VERSION="${BUILD_VERSION}"

    # Check version compatibility client.

    # Show output for manual debugging.
    "${DIRECT_CSI_CLIENT}" drives list --all -o wide

    # Show output for manual debugging.
    "${DIRECT_CSI_CLIENT}" volumes list --all -o wide

    mapfile -t upgraded_volumes < <("${DIRECT_CSI_CLIENT}" volumes list --status published | awk '!/WARNING/ {print $1}')
    if [[ ${#upgraded_volumes[@]} -ne ${#volumes[@]} ]]; then
        echo "$ME: volume count is not matching in version compatibility client tests"
        return 1
    fi

    for volume in "${upgraded_volumes[@]}"; do
        if [[ ! ${volumes[${volume}]+_} ]]; then
            echo "$ME: ${volume} not found in version compatibility client tests"
            return 1
        fi
    done

    install_directcsi

    # wait for initial sync
    sleep 35

    # Show output for manual debugging.
    "${DIRECT_CSI_CLIENT}" drives list --all -o wide

    check_drives_state InUse

    # Show output for manual debugging.
    "${DIRECT_CSI_CLIENT}" volumes list --all -o wide

    mapfile -t upgraded_volumes < <("${DIRECT_CSI_CLIENT}" volumes list --status published | awk '!/WARNING/ {print $1}')
    if [[ ${#upgraded_volumes[@]} -ne ${#volumes[@]} ]]; then
        echo "$ME: volume count is not matching after upgrade"
        return 1
    fi

    for volume in "${upgraded_volumes[@]}"; do
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

echo "$ME: ================================= Run upgrade test from v1.4.6 ================================="
do_upgrade_test "1.4.6"

# kubernetes version 1.22+ is not supported in directpv:v2.0.9
# skipping v2.0.9 upgrade test for v1.22+ versions
minor=$(kubectl version -o json | jq .serverVersion.minor)
minor=$(echo "$minor" | tr -d '"')
if [ "$minor" -lt 23 ]; then
    echo "$ME: ================================= Run upgrade test from v2.0.9 ================================="
    do_upgrade_test "2.0.9"
fi

# unmount all direct-csi mounts of previous installation if any.
mount | awk '/direct-csi/ {print $3}' | xargs sudo umount -fl

remove_luks
remove_lvm
