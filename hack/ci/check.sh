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
    ./kubectl-direct_csi install --image "direct-csi:${IMAGE_TAG}"
    sleep 1m
    kubectl describe pods -n direct-csi-min-io
    ./kubectl-direct_csi info
    ./kubectl-direct_csi drives list -o wide --all
}

function check_drives() {
    if ! ./kubectl-direct_csi drives list --drives="$LV_DEVICE" | grep -q -e "$LV_DEVICE.*Available"; then
        echo "LVM device $LV_DEVICE not found in Available state"
        return 1
    fi

    if ! ./kubectl-direct_csi drives list --drives="$LUKS_DEVICE" | grep -q -e "$LUKS_DEVICE.*Available"; then
        echo "LUKS device $LUKS_DEVICE not found in Available state"
        return 1
    fi

    ./kubectl-direct_csi drives format --all
    sleep 5
    ./kubectl-direct_csi drives list --all -o wide
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
}

function main() {
    setup_lvm
    setup_luks
    install_directcsi
    check_drives
    deploy_minio
    uninstall_minio
    uninstall_directcsi
}

main
