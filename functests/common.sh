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

function is_github_workflow() {
    [ "${GITHUB_ACTIONS}" == "true" ]
}

function unmount_directpv() {
    mount | awk '/direct|pvc-/ {print $3}' > /tmp/.output
    if grep -q . /tmp/.output; then
        # shellcheck disable=SC2046
        sudo umount -fl $(cat /tmp/.output)
    fi
    rm /tmp/.output
}

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
    if ! is_github_workflow; then
        return
    fi

    LV_LOOP_DEVICE=$(create_loop testpv.img 1G)
    sudo pvcreate --quiet "${LV_LOOP_DEVICE}" >/dev/null 2>&1
    sudo vgcreate --quiet "${VG_NAME}" "${LV_LOOP_DEVICE}" >/dev/null 2>&1
    sudo lvcreate --quiet --name=testlv --extents=100%FREE "${VG_NAME}" >/dev/null 2>&1
    LV_DEVICE=$(basename "$(readlink -f "/dev/${VG_NAME}/testlv")")
}

function remove_lvm() {
    if ! is_github_workflow; then
        return
    fi

    sudo lvchange --quiet --activate n "${VG_NAME}/testlv"
    sudo lvremove --quiet --yes "${VG_NAME}/testlv" >/dev/null 2>&1
    sudo vgremove --quiet "${VG_NAME}" >/dev/null 2>&1
    sudo pvremove --quiet "${LV_LOOP_DEVICE}" >/dev/null 2>&1
    sudo losetup --detach "${LV_LOOP_DEVICE}"
    rm -f testpv.img
}

function setup_luks() {
    if ! is_github_workflow; then
        return
    fi

    LUKS_LOOP_DEVICE=$(create_loop testluks.img 1G)
    echo "mylukspassword" > lukspassfile
    echo -ne "YES\nYES\nYES\nYES\n" | sudo cryptsetup luksFormat "${LUKS_LOOP_DEVICE}" lukspassfile
    sudo cryptsetup luksOpen "${LUKS_LOOP_DEVICE}" myluks --key-file=lukspassfile
    LUKS_DEVICE=$(basename "$(readlink -f /dev/mapper/myluks)")
}

function remove_luks() {
    if ! is_github_workflow; then
        return
    fi

    sudo cryptsetup luksClose myluks
    sudo losetup --detach "${LUKS_LOOP_DEVICE}"
    rm -f testluks.img
}

# install_directpv <plugin> <pod_count> [node-selector] [tolerations] [kubelet-dir]
function install_directpv() {
    directpv_client="$1"
    node_selector="$3"
    tolerations="$4"
    kubelet_dir="$5"

    echo "* Installing DirectPV $(${directpv_client} --version | awk '{ print $NF }')"

    cmd=( "${directpv_client}" install --quiet )
    if [ -n "${node_selector}" ]; then
        # shellcheck disable=SC2206
        cmd=( ${cmd[@]} "--node-selector=${node_selector}" )
    fi
    if [ -n "${tolerations}" ]; then
        # shellcheck disable=SC2206
        cmd=( ${cmd[@]} "--tolerations=${tolerations}" )
    fi
    if [ -n "${kubelet_dir}" ]; then
        export KUBELET_DIR_PATH="${kubelet_dir}"
    fi

    "${cmd[@]}"

    unset KUBELET_DIR_PATH

    required_count="$2"
    running_count=0
    while [[ $running_count -lt $required_count ]]; do
        echo "  ...waiting for $(( required_count - running_count )) DirectPV pods to come up"
        sleep 1m
        running_count=$(kubectl get pods --field-selector=status.phase=Running --no-headers --namespace=directpv | wc -l)
    done

    while ! "${directpv_client}" info --quiet; do
        echo "  ...waiting for DirectPV to come up"
        sleep 1m
    done

    sleep 10
}

# uninstall_directpv <plugin> <pod_count>
function uninstall_directpv() {
    directpv_client="$1"

    echo "* Uninstalling DirectPV $(${directpv_client} --version | awk '{ print $NF }')"

    "${directpv_client}" uninstall --quiet

    pending="$2"
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

# usage: check_drives_status <plugin>
function check_drives_status() {
    if ! is_github_workflow; then
        return 0
    fi

    directpv_client="$1"

    if ! "${directpv_client}" list drives -d "${LV_DEVICE}" --no-headers | grep -q -e "${LV_DEVICE}.*Ready"; then
        echo "$ME: error: LVM device ${LV_DEVICE} not found in Ready state"
        return 1
    fi

    if ! "${directpv_client}" list drives -d "${LUKS_DEVICE}" --no-headers | grep -q -e "${LUKS_DEVICE}.*Ready"; then
        echo "$ME: error: LUKS device ${LUKS_DEVICE} not found in Ready state"
        return 1
    fi
}

# usage: add_drives <plugin>
function add_drives() {
    echo "* Adding drives"

    directpv_client="$1"
    config_file="$(mktemp)"

    if ! "${directpv_client}" discover --quiet --output-file "${config_file}" > /tmp/.output 2>&1; then
        cat /tmp/.output
        echo "$ME: error: failed to discover the devices"
        rm "${config_file}"
        return 1
    fi
    if ! "${directpv_client}" init --dangerous --quiet "${config_file}" > /tmp/.output 2>&1; then
        cat /tmp/.output
        echo "$ME: error: failed to initialize the drives"
        rm "${config_file}"
        return 1
    fi

    rm "${config_file}"

    check_drives_status "${directpv_client}"
}

# usage: remove_drives <plugin>
function remove_drives() {
    echo "* Deleting drives"

    directpv_client="$1"

    "${directpv_client}" remove --all --quiet
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

# usage: uninstall_minio <plugin> <minio-yaml>
function uninstall_minio() {
    echo "* Uninstalling minio"

    directpv_client="$1"
    minio_yaml="$2"

    delete_minio "${minio_yaml}"

    if ! kubectl delete pvc --all --force >/tmp/.output 2>&1; then
        cat /tmp/.output
        return 1
    fi
    sleep 5

    # clean all the volumes
    "${directpv_client}" clean --all >/dev/null 2>&1 || true

    while true; do
        count=$("${directpv_client}" list volumes --all --no-headers 2>/dev/null | wc -l)
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

# usage: test_volume_expansion <plugin> <sleep-yaml>
function test_volume_expansion() {
    echo "* Testing volume expansion"

    directpv_client="$1"

    sleep_yaml="$2"
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

    while "${directpv_client}" list volumes --all --no-headers 2>/dev/null | grep -q .; do
        echo "  ...waiting for sleep-pvc volume to be removed"
        sleep 1m
    done
}
