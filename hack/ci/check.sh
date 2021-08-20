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

set -ex

IMAGE_TAG="$1"

function setup_lvm() {
    sudo truncate --size=1G disk-{1..4}.img
    for disk in disk-{1..4}.img; do sudo losetup --find $disk; done
    devices=( $(for disk in disk-{1..4}.img; do sudo losetup --noheadings --output NAME --associated $disk; done) )
    sudo pvcreate "${devices[@]}"
    vgname="test-vg-$RANDOM"
    sudo vgcreate "$vgname" "${devices[@]}"
    for i in {1..4}; do sudo lvcreate --size=800MiB "$vgname"; done
}

function install_directcsi() {
    ./kubectl-direct_csi install --image "direct-csi:${IMAGE_TAG}"
    sleep 1m
    kubectl describe pods -n direct-csi-min-io
    ./kubectl-direct_csi info

    if ! ./kubectl-direct_csi drives list | grep -q Available; then
        ./kubectl-direct_csi drives list -o wide --all
        echo "No available disks found in the list"
        exit 1
    fi
    
    ./kubectl-direct_csi drives format --all
    sleep 5
    ./kubectl-direct_csi drives list -o wide --all
}

function deploy_minio() {
    kubectl apply -f hack/ci/minio.yaml
    sleep 1m
    kubectl get pods -o wide

    runningpods=$(kubectl get pods --field-selector=status.phase=Running --no-headers | wc -l)
    if [[ $runningpods -ne 4 ]]; then
        echo "MinIO deployment failed"
        exit 1
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
        echo "Volumes were not cleared upon deletion"
        exit 1
    fi

    if ./kubectl-direct_csi drives ls | grep -q InUse; then
        echo "disks are still inuse after clearing up volumes"
        exit 1
    fi
}

function uninstall_directcsi() {   
    ./kubectl-direct_csi uninstall --crd --force
    sleep 1m
    kubectl get pods -n direct-csi-min-io
    if kubectl get ns | grep -q direct-csi-min-io; then
        echo "namespace not cleared upon uninstallation"
        exit 1
    fi
}

function main() {
    setup_lvm
    install_directcsi
    deploy_minio
    uninstall_minio
    uninstall_directcsi
}

main
