#!/usr/bin/env bash

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

#
# This script is indented to run in Github workflow.
# DO NOT USE in other systems.
#

if [[ $# -ne 1 ]]; then
    echo "error: image tag must be provided"
    echo "usage: $(basename "$0") <IMAGE_TAG>"
    exit 255
fi

set -ex

IMAGE_TAG="$1"
LV_DEVICE=""
LUKS_DEVICE=""
DRIVES_COUNT=0
VOLUMES_COUNT=0
FROM_VERSION="1.3.6"
OLD_CLIENT="kubectl-direct_csi_${FROM_VERSION}_linux_amd64"

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
    yes YES | sudo cryptsetup -y -v luksFormat "$loopdev" lukspassfile
    sudo cryptsetup -v luksOpen "$loopdev" myluks --key-file=lukspassfile
    LUKS_DEVICE=$(readlink -f /dev/mapper/myluks)
}

function install_directcsi() {
    cmd="$1"
    tag="$2"

    "./$cmd" install --image "direct-csi:$tag"
    sleep 1m
    kubectl describe pods -n direct-csi-min-io
    "./$cmd" info
}

function check_drives_state() {
    if ! ./kubectl-direct_csi drives list --drives="$LV_DEVICE" | grep -q -e "$LV_DEVICE.*$1"; then
        echo "LVM device $LV_DEVICE not found in $1 state"
        return 1
    fi

    if ! ./kubectl-direct_csi drives list --drives="$LUKS_DEVICE" | grep -q -e "$LUKS_DEVICE.*$1"; then
        echo "LUKS device $LUKS_DEVICE not found in $1 state"
        return 1
    fi
}

function check_drives() {
    ./kubectl-direct_csi drives list --all
    check_drives_state Available
    ./kubectl-direct_csi drives format --all
    sleep 5
    ./kubectl-direct_csi drives list --all -o wide
    check_drives_state Ready
}

function deploy_minio() {
    kubectl apply -f hack/ci/minio.yaml
    sleep 1m
    kubectl get pods -o wide

    runningpods=$(kubectl get pods --field-selector=status.phase=Running --no-headers | wc -l)
    if [[ $runningpods -ne 4 ]]; then
        echo "MinIO deployment failed"
        return 1
    fi
}

function uninstall_minio() {
    kubectl delete -f hack/ci/minio.yaml
    kubectl delete pvc --all
    sleep 10s
    ./kubectl-direct_csi volumes ls
    ./kubectl-direct_csi drives ls --all

    directcsivolumes=$(./kubectl-direct_csi volumes ls | wc -l)
    if [[ $directcsivolumes -gt 1 ]]; then
        echo "Volumes still exist"
        return 1
    fi

    if ./kubectl-direct_csi drives ls | grep -q InUse; then
        echo "Drives are still in use"
        return 1
    fi
}

function uninstall_directcsi() {
    ./kubectl-direct_csi uninstall --crd --force
    sleep 1m
    kubectl get pods -n direct-csi-min-io
    if kubectl get ns | grep -q direct-csi-min-io; then
        echo "direct-csi-min-io namespace still exists"
        return 1
    fi
    # Check uninstall succeeds even if direct-csi is completely gone.
    ./kubectl-direct_csi uninstall --crd --force
}


function install_directcsi_older() {
    wget https://github.com/minio/direct-csi/releases/download/v${FROM_VERSION}/${OLD_CLIENT}
    chmod +x ${OLD_CLIENT}
    install_directcsi ${OLD_CLIENT} v${FROM_VERSION}
    "./$OLD_CLIENT" drives format --all --force
    sleep 5
    "./$OLD_CLIENT" drives list --all
}

function uninstall_directcsi_older() {
    "./$OLD_CLIENT" drives list --all
    DRIVES_COUNT=$("./$OLD_CLIENT" drives list --all | wc -l)
    "./$OLD_CLIENT" volumes list
    VOLUMES_COUNT=$("./$OLD_CLIENT" volumes list | wc -l)
    "./$OLD_CLIENT" uninstall
    sleep 1m 
    kubectl get pods -n direct-csi-min-io
}

function verify_upgraded_objects() {
    ./kubectl-direct_csi drives list --all -o wide
    if [[ $(./kubectl-direct_csi drives list --all | wc -l) -ne ${DRIVES_COUNT} ]]; then
        echo "Incorrect drive list after upgrade"
        return 1
    fi
    ./kubectl-direct_csi volumes list -o wide
    if [[ $(./kubectl-direct_csi volumes list | wc -l) -ne ${VOLUMES_COUNT} ]]; then
        echo "Incorrect volume list after upgrade"
        return 1
    fi
}

function check_fresh_installation() {
    install_directcsi kubectl-direct_csi "${IMAGE_TAG}"
    check_drives
    deploy_minio
    uninstall_minio
    uninstall_directcsi
}

function check_upgrades() {
    install_directcsi_older
    deploy_minio
    uninstall_directcsi_older
    install_directcsi kubectl-direct_csi "${IMAGE_TAG}"
    verify_upgraded_objects
    uninstall_minio
    uninstall_directcsi
}
 
function main() {
    setup_lvm
    setup_luks
    
    check_fresh_installation
    mount | awk '/direct-csi/ {print $3}' | xargs sudo umount -fl
    check_upgrades
}

main
