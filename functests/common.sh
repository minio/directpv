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

export LV_LOOP_DEVICE=
export VG_NAME="testvg${RANDOM}"
export LV_DEVICE=
export LUKS_LOOP_DEVICE=
export LUKS_DEVICE=
export DIRECTPV_CLIENT=

# usage: create_loop <newfile> <size>
function create_loop() {
    truncate --size="$2" "$1"
    sudo losetup --find "$1"
    if [ -n "${RHEL7_TEST}" ]; then
        sudo losetup --output NAME --associated "$1" | tail -n +2
    else
        sudo losetup --noheadings --output NAME --associated "$1"
    fi
}

function setup_lvm() {
    LV_LOOP_DEVICE=$(create_loop testpv.img 1G)
    sudo pvcreate "${LV_LOOP_DEVICE}"
    sudo vgcreate "${VG_NAME}" "${LV_LOOP_DEVICE}"
    sudo lvcreate --name=testlv --extents=100%FREE "${VG_NAME}"
    LV_DEVICE=$(basename "$(readlink -f "/dev/${VG_NAME}/testlv")")
}

function remove_lvm() {
    sudo lvchange --quiet --activate n "${VG_NAME}/testlv"
    sudo lvremove --quiet --yes "${VG_NAME}/testlv"
    sudo vgremove --quiet "${VG_NAME}"
    sudo pvremove --quiet "${LV_LOOP_DEVICE}"
    sudo losetup --detach "${LV_LOOP_DEVICE}"
    rm -f testpv.img
}

function setup_luks() {
    LUKS_LOOP_DEVICE=$(create_loop testluks.img 1G)
    echo "mylukspassword" > lukspassfile
    yes YES | sudo cryptsetup luksFormat "${LUKS_LOOP_DEVICE}" lukspassfile
    sudo cryptsetup luksOpen "${LUKS_LOOP_DEVICE}" myluks --key-file=lukspassfile
    LUKS_DEVICE=$(basename "$(readlink -f /dev/mapper/myluks)")
}

function remove_luks() {
    sudo cryptsetup luksClose myluks
    sudo losetup --detach "${LUKS_LOOP_DEVICE}"
    rm -f testluks.img
}

function install_directpv() {
    "${DIRECTPV_CLIENT}" install --quiet

    required_count=5
    running_count=0
    while [[ $running_count -lt $required_count ]]; do
        echo "$ME: waiting for $(( required_count - running_count )) DirectPV pods to come up"
        sleep $(( required_count - running_count ))
        running_count=$(kubectl get pods --field-selector=status.phase=Running --no-headers --namespace=directpv-min-io | wc -l)
    done

    while ! "${DIRECTPV_CLIENT}" info --quiet; do
        echo "$ME: waiting for DirectPV to come up"
        sleep 5
    done
}

function uninstall_directpv() {
    "${DIRECTPV_CLIENT}" uninstall --quiet

    pending=5
    while [[ $pending -gt 0 ]]; do
        echo "$ME: waiting for ${pending} DirectPV pods to go down"
        sleep ${pending}
        pending=$(kubectl get pods --field-selector=status.phase=Running --no-headers --namespace=directpv-min-io | wc -l)
    done

    while kubectl get namespace directpv-min-io --no-headers | grep -q .; do
        echo "$ME: waiting for directpv-min-io namespace to be removed"
        sleep 5
    done

    return 0
}

# usage: check_drives_status <status>
function check_drives_status() {
    status="$1"

    if ! "${DIRECTPV_CLIENT}" get drives -d "${LV_DEVICE}" --no-headers | grep -q -e "${LV_DEVICE}.*${status}"; then
        echo "$ME: error: LVM device ${LV_DEVICE} not found in ${status} state"
        return 1
    fi

    if ! "${DIRECTPV_CLIENT}" get drives -d "${LUKS_DEVICE}" --no-headers | grep -q -e "${LUKS_DEVICE}.*${status}"; then
        echo "$ME: error: LUKS device ${LUKS_DEVICE} not found in ${status} state"
        return 1
    fi
}

function add_drives() {
    # Get the nodeport service url
    url=$(minikube service --namespace=directpv-min-io admin-service --url)
    admin_server=${url#"http://"}

    echo -e 'ALL\nALL\nYes\n' | ./kubectl-directpv format --admin-server ${admin_server} --allowed --force

    # Show output for manual debugging.
    "${DIRECTPV_CLIENT}" get drives --all

    check_drives_status Ready
}

function remove_drives() {
    ./kubectl-directpv release --all --quiet
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
    retry_count=0
    while [[ $pending -gt 0 ]]; do
        if [[ $retry_count -gt 50 ]]; then
            kubectl delete pods --all --force --grace-period 0
        fi
        retry_count=$((retry_count + 1))
        echo "$ME: waiting for ${pending} minio pods to go down"
        sleep ${pending}
        pending=$(kubectl get pods --field-selector=status.phase=Running --no-headers | grep -c '^minio-' || true)
    done

    kubectl delete pvc --all --force
    sleep 3

    # purge all the volumes
    "${DIRECTPV_CLIENT}" purge --all || true

    while true; do
        count=$("${DIRECTPV_CLIENT}" get volumes --all --no-headers | tee /dev/stderr | wc -l)
        if [[ $count -eq 0 ]]; then
            break
        fi
        echo "$ME: waiting for ${count} volumes to be removed"
        sleep 3
    done

    # Show output for manual debugging.
    "${DIRECTPV_CLIENT}" get drives --all
}
