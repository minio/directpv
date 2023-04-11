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
    sudo pvcreate --quiet "${LV_LOOP_DEVICE}" >/dev/null 2>&1
    sudo vgcreate --quiet "${VG_NAME}" "${LV_LOOP_DEVICE}" >/dev/null 2>&1
    sudo lvcreate --quiet --name=testlv --extents=100%FREE "${VG_NAME}" >/dev/null 2>&1
    LV_DEVICE=$(basename "$(readlink -f "/dev/${VG_NAME}/testlv")")
}

function remove_lvm() {
    sudo lvchange --quiet --activate n "${VG_NAME}/testlv"
    sudo lvremove --quiet --yes "${VG_NAME}/testlv" >/dev/null 2>&1
    sudo vgremove --quiet "${VG_NAME}" >/dev/null 2>&1
    sudo pvremove --quiet "${LV_LOOP_DEVICE}" >/dev/null 2>&1
    sudo losetup --detach "${LV_LOOP_DEVICE}"
    rm -f testpv.img
}

function setup_luks() {
    LUKS_LOOP_DEVICE=$(create_loop testluks.img 1G)
    echo "mylukspassword" > lukspassfile
    yes YES 2>/dev/null | sudo cryptsetup luksFormat "${LUKS_LOOP_DEVICE}" lukspassfile
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
    echo "* Installing DirectPV"

    "${DIRECTPV_CLIENT}" install --quiet

    required_count="$1"
    running_count=0
    while [[ $running_count -lt $required_count ]]; do
        echo "  ...waiting for $(( required_count - running_count )) DirectPV pods to come up"
        sleep 1m
        running_count=$(kubectl get pods --field-selector=status.phase=Running --no-headers --namespace=directpv | wc -l)
    done

    while ! "${DIRECTPV_CLIENT}" info --quiet; do
        echo "  ...waiting for DirectPV to come up"
        sleep 1m
    done

    sleep 10
}

# uninstall_directpv <pod_count>
function uninstall_directpv() {
    echo "* Uninstalling DirectPV"

    "${DIRECTPV_CLIENT}" uninstall --quiet

    pending="$1"
    while [[ $pending -gt 0 ]]; do
        echo "  ...waiting for ${pending} DirectPV pods to go down"
        sleep 5
        pending=$(kubectl get pods --field-selector=status.phase=Running --no-headers --namespace=directpv-min-io 2>/dev/null | wc -l)
    done

    while kubectl get namespace directpv-min-io --no-headers 2>/dev/null | grep -q .; do
        echo "  ...waiting for directpv-min-io namespace to be removed"
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
    echo "* Adding drives"

    config_file="$(mktemp)"

    if ! "${DIRECTPV_CLIENT}" discover --quiet --output-file "${config_file}" > /tmp/.output 2>&1; then
        cat /tmp/.output
        echo "$ME: error: failed to discover the devices"
        rm "${config_file}"
        return 1
    fi
    if ! "${DIRECTPV_CLIENT}" init --dangerous --quiet "${config_file}" > /tmp/.output 2>&1; then
        cat /tmp/.output
        echo "$ME: error: failed to initialize the drives"
        rm "${config_file}"
        return 1
    fi

    rm "${config_file}"

    check_drives_status Ready
}

function remove_drives() {
    echo "* Deleting drives"

    "${DIRECTPV_CLIENT}" remove --all --quiet
}

# usage: deploy_minio <minio-yaml>
function deploy_minio() {
    echo "* Deploying minio"

    minio_yaml="$1"

    kubectl apply -f "${minio_yaml}" 1>/dev/null

    required_count=4
    running_count=0
    while [[ $running_count -lt $required_count ]]; do
        echo "  ...waiting for $(( required_count - running_count )) minio pods to come up"
        sleep 1m
        running_count=$(kubectl get pods --field-selector=status.phase=Running --no-headers 2>/dev/null | grep -c '^minio-' || true)
    done
}

# usage: test_force_delete
function test_force_delete() {
    echo "* Testing force deletion"

    running_count=0
    required_count=$(kubectl get pods --field-selector=status.phase=Running --no-headers 2>/dev/null | grep -c '^minio-' || true)

    kubectl delete pods --all --force --grace-period 0

    while [[ $running_count -lt $required_count ]]; do
        echo "  ...waiting for $(( required_count - running_count )) minio pods to come up after force deletion"
        sleep 30
        running_count=$(kubectl get pods --field-selector=status.phase=Running --no-headers 2>/dev/null | grep -c '^minio-' || true)
    done
}

# usage: delete_minio <minio-yaml>
function delete_minio() {
    echo "* Deleting minio"

    minio_yaml="$1"

    kubectl delete -f "${minio_yaml}" 1>/dev/null
    pending=4
    for (( i = 0; i < 5; i++ )); do
        if [[ $pending -eq 0 ]]; then
            break
        fi

        if [[ $i -eq 4 ]]; then
            echo "* Deleting minio forcefully"
            kubectl delete pods --all --force --grace-period 0 >/dev/null 2>&1
            sleep 5
            break
        fi

        echo "  ...waiting for ${pending} minio pods to go down"
        sleep 30
        pending=$(kubectl get pods --field-selector=status.phase=Running --no-headers 2>/dev/null | grep -c '^minio-' || true)
    done
}

# usage: uninstall_minio <minio-yaml>
function uninstall_minio() {
    echo "* Uninstalling minio"

    minio_yaml="$1"

    delete_minio "${minio_yaml}"

    if ! kubectl delete pvc --all --force >/tmp/.output 2>&1; then
        cat /tmp/.output
        return 1
    fi
    sleep 5

    # clean all the volumes
    "${DIRECTPV_CLIENT}" clean --all >/dev/null 2>&1 || true

    while true; do
        count=$("${DIRECTPV_CLIENT}" list volumes --all --no-headers 2>/dev/null | wc -l)
        if [[ $count -eq 0 ]]; then
            break
        fi
        echo "  ...waiting for ${count} volumes to be removed"
        sleep 30
    done
}

# usage: install_directcsi <plugin> <pod-count>
function install_directcsi() {
    echo "* Installing DirectCSI"

    directcsi_client="$1"
    required_count="$2"

    if ! "${directcsi_client}" install >/tmp/.output 2>&1; then
        cat /tmp/.output
        return 1
    fi

    running_count=0
    while [[ $running_count -lt $required_count ]]; do
        echo "  ...waiting for $(( required_count - running_count )) DirectCSI pods to come up"
        sleep 1m
        running_count=$(kubectl get pods --field-selector=status.phase=Running --no-headers --namespace=direct-csi-min-io 2>/dev/null | wc -l)
    done

    while ! "${directcsi_client}" info >/dev/null 2>&1; do
        echo "  ...waiting for DirectCSI to come up"
        sleep 1m
    done

    sleep 10
}

# usage: install_directcsi <plugin> <pod-count>
function uninstall_directcsi() {
    echo "* Uninstalling DirectCSI"

    directcsi_client="$1"
    pending="$2"

    if ! "${directcsi_client}" uninstall >/tmp/.output 2>&1; then
        cat /tmp/.output
        return 1
    fi

    while [[ $pending -gt 0 ]]; do
        echo "  ...waiting for ${pending} direct-csi pods to go down"
        sleep 5
        pending=$(kubectl get pods --field-selector=status.phase=Running --no-headers --namespace=direct-csi-min-io 2>/dev/null | wc -l)
    done

    while kubectl get namespace direct-csi-min-io --no-headers 2>/dev/null | grep -q .; do
        echo "  ...waiting for direct-csi-min-io namespace to be removed"
        sleep 5
    done
}

# usage: force_install_directcsi <plugin>
function force_uninstall_directcsi() {
    echo "* Uninstalling DirectCSI forcefully"

    directcsi_client="$1"
    if ! "${directcsi_client}" uninstall --force --crd >/tmp/.output 2>&1; then
        cat /tmp/.output
        return 1
    fi
    sleep 5
}

# usage: test_volume_expansion <sleep-yaml>
function test_volume_expansion() {
    echo "* Testing volume expansion"

    sleep_yaml="$1"
    kubectl apply -f "${sleep_yaml}" 1>/dev/null
    while ! kubectl get pods --field-selector=status.phase=Running --no-headers 2>/dev/null | grep -q sleep-pod; do
        echo "  ...waiting for sleep-pod to come up"
        sleep 1m
    done

    kubectl get pvc sleep-pvc -o json > /tmp/8.json
    python <<EOF
import json
with open("/tmp/8.json") as f:
    d = json.load(f)
d["spec"]["resources"]["requests"]["storage"] = "16Mi"
with open("/tmp/16.json", "w") as f:
    json.dump(d, f)
EOF
    rm -f /tmp/8.json
    kubectl apply -f /tmp/16.json 1>/dev/null
    rm -f /tmp/16.json
    while [ "$(kubectl get pvc sleep-pvc --no-headers -o custom-columns=CAPACITY:.status.capacity.storage)" != "16Mi" ]; do
        echo "  ...waiting for sleep-pvc to be expanded"
        sleep 1m
    done

    kubectl delete -f "${sleep_yaml}" 1>/dev/null
    while kubectl get pods sleep-pod --no-headers 2>/dev/null | grep -q sleep-pod; do
        echo "  ...waiting for sleep-pod to go down"
        sleep 1m
    done

    while "${DIRECTPV_CLIENT}" list volumes --all --no-headers 2>/dev/null | grep -q .; do
        echo "  ...waiting for sleep-pvc volume to be removed"
        sleep 1m
    done
}
