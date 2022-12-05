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

# install_directpv <pod_count>
function install_directpv() {
    "${DIRECTPV_CLIENT}" install --quiet

    required_count="$1"
    running_count=0
    while [[ $running_count -lt $required_count ]]; do
        echo "$ME: waiting for $(( required_count - running_count )) DirectPV pods to come up"
        sleep $(( required_count - running_count ))
        running_count=$(kubectl get pods --field-selector=status.phase=Running --no-headers --namespace=directpv | wc -l)
    done

    while ! "${DIRECTPV_CLIENT}" info --quiet; do
        echo "$ME: waiting for DirectPV to come up"
        sleep 5
    done

    sleep 1m
}

# uninstall_directpv <pod_count>
function uninstall_directpv() {
    "${DIRECTPV_CLIENT}" uninstall --quiet
    
    pending="$1"
    while [[ $pending -gt 0 ]]; do
        echo "$ME: waiting for ${pending} DirectPV pods to go down"
        sleep "${pending}"
        pending=$(kubectl get pods --field-selector=status.phase=Running --no-headers --namespace=directpv | wc -l)
    done

    while kubectl get namespace directpv --no-headers | grep -q .; do
        echo "$ME: waiting for directpv namespace to be removed"
        sleep 5
    done

    return 0
}

# usage: check_drives_status <status>
function check_drives_status() {
    status="$1"

    if ! "${DIRECTPV_CLIENT}" list drives -d "${LV_DEVICE}" --no-headers | grep -q -e "${LV_DEVICE}.*${status}"; then
        echo "$ME: error: LVM device ${LV_DEVICE} not found in ${status} state"
        return 1
    fi

    if ! "${DIRECTPV_CLIENT}" list drives -d "${LUKS_DEVICE}" --no-headers | grep -q -e "${LUKS_DEVICE}.*${status}"; then
        echo "$ME: error: LUKS device ${LUKS_DEVICE} not found in ${status} state"
        return 1
    fi
}

function add_drives() {
    config_file="$(mktemp)"    

    if ! "${DIRECTPV_CLIENT}" discover --output-file "${config_file}"; then
        echo "$ME: error: failed to discover the devices"
        rm "${config_file}"
        return 1
    fi
    if ! echo Yes | "${DIRECTPV_CLIENT}" init "${config_file}"; then
        echo "$ME: error: failed to initialize the drives"
        rm "${config_file}"
        return 1
    fi
    
    rm "${config_file}"
   
    # Show output for manual debugging.
    "${DIRECTPV_CLIENT}" list drives --all

    check_drives_status Ready
}

function remove_drives() {
    "${DIRECTPV_CLIENT}" remove --all --quiet
}

# usage: deploy_minio <minio-yaml>
function deploy_minio() {
    minio_yaml="$1"

    kubectl apply -f "${minio_yaml}"

    required_count=4
    running_count=0
    while [[ $running_count -lt $required_count ]]; do
        echo "$ME: waiting for $(( required_count - running_count )) minio pods to come up"
        sleep $(( required_count - running_count ))
        running_count=$(kubectl get pods --field-selector=status.phase=Running --no-headers | grep -c '^minio-' || true)
    done
}

# usage: delete_minio <minio-yaml>
function delete_minio() {
    minio_yaml="$1"

    kubectl delete -f "${minio_yaml}"
    pending=4
    retry_count=0
    while [[ $pending -gt 0 ]]; do
        if [[ $retry_count -gt 50 ]]; then
            kubectl delete pods --all --force --grace-period 0
        fi
        retry_count=$((retry_count + 1))
        echo "$ME: waiting for ${pending} minio pods to go down"
        sleep "${pending}"
        pending=$(kubectl get pods --field-selector=status.phase=Running --no-headers | grep -c '^minio-' || true)
    done
}

# usage: uninstall_minio <minio-yaml>
function uninstall_minio() {
    minio_yaml="$1"

    delete_minio "${minio_yaml}"

    kubectl delete pvc --all --force
    sleep 3

    # release all the volumes
    "${DIRECTPV_CLIENT}" release --all || true

    while true; do
        count=$("${DIRECTPV_CLIENT}" list volumes --all --no-headers | tee /dev/stderr | wc -l)
        if [[ $count -eq 0 ]]; then
            break
        fi
        echo "$ME: waiting for ${count} volumes to be removed"
        sleep 3
    done

    # Show output for manual debugging.
    "${DIRECTPV_CLIENT}" list drives --all
}

# usage: install_directcsi <plugin> <pod-count>
function install_directcsi() {
    directcsi_client="$1"
    required_count="$2"

    "${directcsi_client}" install

    running_count=0
    while [[ $running_count -lt $required_count ]]; do
        echo "$ME: waiting for $(( required_count - running_count )) DirectCSI pods to come up"
        sleep $(( required_count - running_count ))
        running_count=$(kubectl get pods --field-selector=status.phase=Running --no-headers --namespace=direct-csi-min-io | wc -l)
    done

    while ! "${directcsi_client}" info; do
        echo "$ME: waiting for DirectCSI to come up"
        sleep 5
    done

    sleep 1m
}

# usage: install_directcsi <plugin> <pod-count>
function uninstall_directcsi() {
    directcsi_client="$1"
    pending="$2"

    "${directcsi_client}" uninstall

    while [[ $pending -gt 0 ]]; do
        echo "$ME: waiting for ${pending} direct-csi pods to go down"
        sleep "${pending}"
        pending=$(kubectl get pods --field-selector=status.phase=Running --no-headers --namespace=direct-csi-min-io | wc -l)
    done

    while kubectl get namespace direct-csi-min-io --no-headers | grep -q .; do
        echo "$ME: waiting for direct-csi-min-io namespace to be removed"
        sleep 5
    done
}

# usage: force_install_directcsi <plugin>
function force_uninstall_directcsi() {
    directcsi_client="$1"
    "${directcsi_client}" uninstall --force --crd
    sleep 5
}
