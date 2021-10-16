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

export LV_DEVICE=
export LUKS_DEVICE=
export DIRECT_CSI_CLIENT=
export DIRECT_CSI_VERSION=

# usage: create_loop <newfile> <size>
function create_loop() {
    truncate --size="$2" "$1"
    sudo losetup --find "$1"
    sudo losetup --noheadings --output NAME --associated "$1"
}

function setup_lvm() {
    loopdev=$(create_loop testpv.img 1G)
    sudo pvcreate "$loopdev"
    vgname="testvg$RANDOM"
    sudo vgcreate "$vgname" "$loopdev"
    sudo lvcreate --name=testlv --extents=100%FREE "$vgname"
    LV_DEVICE=$(readlink -f "/dev/$vgname/testlv")
}

function setup_luks() {
    loopdev=$(create_loop testluks.img 1G)
    echo "mylukspassword" > lukspassfile
    yes YES | sudo cryptsetup luksFormat "$loopdev" lukspassfile
    sudo cryptsetup -v luksOpen "$loopdev" myluks --key-file=lukspassfile
    LUKS_DEVICE=$(readlink -f /dev/mapper/myluks)
}

function install_directcsi() {
    "${DIRECT_CSI_CLIENT}" install --image "direct-csi:${DIRECT_CSI_VERSION}"

    required_count=4
    if [[ "$DIRECT_CSI_VERSION" == "v1.3.6" ]]; then
        required_count=7 # plus 3 for conversion deployment pods
    fi
    running_count=0
    while [[ $running_count -lt $required_count ]]; do
        echo "$ME: waiting for $(( required_count - running_count )) direct-csi pods to come up"
        sleep $(( required_count - running_count ))
        running_count=$(kubectl get pods --field-selector=status.phase=Running --no-headers --namespace=direct-csi-min-io | wc -l)
    done

    while true; do
        echo "$ME: waiting for direct-csi to come up"
        sleep 5
        if "${DIRECT_CSI_CLIENT}" info; then
            return 0
        fi
    done
}

function uninstall_directcsi() {
    "${DIRECT_CSI_CLIENT}" uninstall  --crd --force

    pending=4
    if [[ "$DIRECT_CSI_VERSION" == "v1.3.6" ]]; then
        pending=7 # plus 3 for conversion deployment pods
    fi
    while [[ $pending -gt 0 ]]; do
        echo "$ME: waiting for ${pending} direct-csi pods to go down"
        sleep ${pending}
        pending=$(kubectl get pods --field-selector=status.phase=Running --no-headers --namespace=direct-csi-min-io | wc -l)
    done

    while true; do
        echo "$ME: waiting for direct-csi-min-io namespace to be removed"
        sleep 5
        if ! kubectl get namespace direct-csi-min-io --no-headers | grep -q .; then
            return 0
        fi
    done
}

# usage: check_drives_state <state>
function check_drives_state() {
    state="$1"
    if ! "${DIRECT_CSI_CLIENT}" drives list --drives="${LV_DEVICE}" | grep -q -e "${LV_DEVICE}.*${state}"; then
        echo "$ME: error: LVM device ${LV_DEVICE} not found in ${state} state"
        return 1
    fi

    if ! "${DIRECT_CSI_CLIENT}" drives list --drives="${LUKS_DEVICE}" | grep -q -e "${LUKS_DEVICE}.*${state}"; then
        echo "$ME: error: LUKS device ${LUKS_DEVICE} not found in ${state} state"
        return 1
    fi
}

function check_drives() {
    # Show output for manual debugging.
    "${DIRECT_CSI_CLIENT}" drives list --all

    check_drives_state Available
    "${DIRECT_CSI_CLIENT}" drives format --all --force
    sleep 5

    # Show output for manual debugging.
    "${DIRECT_CSI_CLIENT}" drives list --all

    check_drives_state Ready
}

function deploy_minio() {
    kubectl apply -f functests/minio.yaml

    required_count=4
    running_count=0
    while [[ $running_count -lt $required_count ]]; do
        echo "$ME: waiting for $(( required_count - running_count )) minio pods to come up"
        sleep $(( required_count - running_count ))
        running_count=$(kubectl get pods --field-selector=status.phase=Running --no-headers | grep -c '^minio-' || true)
    done
}

function uninstall_minio() {
    kubectl delete -f functests/minio.yaml
    pending=4
    while [[ $pending -gt 0 ]]; do
        echo "$ME: waiting for ${pending} minio pods to go down"
        sleep ${pending}
        pending=$(kubectl get pods --field-selector=status.phase=Running --no-headers | grep -c '^minio-' || true)
    done

    kubectl delete pvc --all
    # Show output for manual debugging.
    "${DIRECT_CSI_CLIENT}" volumes ls

    while true; do
        count=$("${DIRECT_CSI_CLIENT}" volumes ls | wc -l)
        if [[ $count -eq 1 ]]; then
            break
        fi
        echo "$ME: error: ${count} provisioned volumes still exist"
        sleep 3
    done

    # Show output for manual debugging.
    "${DIRECT_CSI_CLIENT}" drives ls --all

    while true; do
        count=$("${DIRECT_CSI_CLIENT}" drives ls | grep -c InUse || true)
        if [[ $count -eq 0 ]]; then
            break
        fi
        echo "$ME: waiting for ${count} drives to be released"
        sleep 5
    done
}
