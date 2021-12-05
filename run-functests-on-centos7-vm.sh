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

centos7image=CentOS-7-x86_64-GenericCloud-2111.qcow2c
baseimage="minikube-${centos7image}"
rsa_private_key="${baseimage}_rsa"
rsa_public_key="${baseimage}_rsa.pub"

function scp_cmd() {
    scp -o GSSAPIAuthentication=no -o StrictHostKeyChecking=no -i "$rsa_private_key" -p "$@"
}

function ssh_cmd() {
    ssh -o GSSAPIAuthentication=no -o StrictHostKeyChecking=no -i "$rsa_private_key" -l root "$@"
}

function setup_base_image() {
    if [ -f "$baseimage" ]; then
        return
    fi

    if [ ! -f "$centos7image" ]; then
        wget "https://cloud.centos.org/centos/7/images/${centos7image}"
    fi
    cp -af "$centos7image" "$baseimage"

    if [ ! -f "$rsa_private_key" ] || [ ! -f "$rsa_public_key" ]; then
        ssh-keygen -q -f "$rsa_private_key" -N ''
    fi

    sudo virt-customize -a "$baseimage" --ssh-inject "root:file:${rsa_public_key}" --root-password password:password --uninstall cloud-init --selinux-relabel
    virt-install --name "$baseimage" --memory 4096 --vcpus 2 --disk "$baseimage" --import --os-variant centos7.0 --print-xml | cat > "${baseimage}.xml"
    sudo virsh define "${baseimage}.xml"
    sudo virsh start "${baseimage}"
    rm -f "${baseimage}.xml"
    sleep 1m
    ipaddr=$(sudo virsh domifaddr "$baseimage" | awk '$4 ~ /[0-9]$/ { split($4,a,"/"); print a[1] }')

    cat > setup-minikube.sh <<EOF
#!/bin/bash
set -ex
setenforce Permissive
sed -i -e s:SELINUX=enforcing:SELINUX=permissive:g /etc/selinux/config
yum remove -y docker docker-client docker-client-latest docker-common docker-latest docker-latest-logrotate docker-logrotate docker-engine
yum install -y yum-utils
yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
yum update -y
yum install -y docker-ce docker-ce-cli containerd.io conntrack-tools wget lvm2 cryptsetup
systemctl enable docker
systemctl start docker
docker run hello-world
wget --quiet --output-document /usr/bin/minikube https://github.com/kubernetes/minikube/releases/download/v1.24.0/minikube-linux-amd64
chmod a+x /usr/bin/minikube
wget --quiet --output-document /usr/bin/kubectl "https://dl.k8s.io/release/\$(wget -q -O - https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod a+x /usr/bin/kubectl
echo 1 > /proc/sys/net/bridge/bridge-nf-call-iptables
minikube start --driver=none
minikube status
minikube stop
EOF
    chmod a+x setup-minikube.sh
    scp_cmd setup-minikube.sh "root@${ipaddr}:"
    ssh_cmd "$ipaddr" "./setup-minikube.sh"
    rm -f setup-minikube.sh
    sudo virsh shutdown "$baseimage"
    sleep 10
    sudo virsh undefine "$baseimage"
}

function build_directcsi() {
    BUILD_TAG=$(git describe --tags --always --dirty)
    export BUILD_TAG

    export CGO_ENABLED=0 GO111MODULE=on
    go build -tags "osusergo netgo static_build" -ldflags="-X main.Version=${BUILD_TAG} -extldflags=-static" github.com/minio/direct-csi/cmd/direct-csi
    go build -tags "osusergo netgo static_build" -ldflags="-X main.Version=${BUILD_TAG} -extldflags=-static" github.com/minio/direct-csi/cmd/kubectl-direct_csi
}

function start_test_vm() {
    cp -af "$baseimage" "${VM_IMAGE}"
    virt-install --name "${VM_NAME}" --memory 4096 --vcpus 2 --disk "${VM_IMAGE}" --import --os-variant centos7.0 --print-xml | cat > "${VM_NAME}.xml"
    sudo virsh define "${VM_NAME}.xml"
    sudo virsh start "${VM_NAME}"
    rm -f "${VM_NAME}.xml"
    sleep 1m
    VM_IPADDR=$(sudo virsh domifaddr "${VM_NAME}" | awk '$4 ~ /[0-9]$/ { split($4,a,"/"); print a[1] }')
    export VM_IPADDR
}

function remove_test_vm() {
    sudo virsh shutdown "${VM_NAME}"
    sleep 10
    sudo virsh undefine "${VM_NAME}"
    rm -f "${VM_IMAGE}"
}

function run_functional_test() {
    export VM_NAME="centos-7-directcsi-test-${BUILD_TAG}-${RANDOM}"
    export VM_IMAGE="${VM_NAME}.qcow2c"
    start_test_vm

    scp_cmd CREDITS LICENSE centos.repo direct-csi kubectl-direct_csi Dockerfile "root@${VM_IPADDR}:"
    ssh_cmd "${VM_IPADDR}" "docker build -t quay.io/minio/direct-csi:${BUILD_TAG} -f Dockerfile ."
    ssh_cmd "${VM_IPADDR}" "minikube start --driver=none"
    scp_cmd -r functests "root@${VM_IPADDR}:"
    ssh_cmd "${VM_IPADDR}" "RHEL7_TEST=1 functests/run.sh ${BUILD_TAG}"

    ssh_cmd "${VM_IPADDR}" "functests/install-directcsi.sh ${BUILD_TAG}"

    qemu-img create -f qcow2 "${VM_NAME}-vdb.qcow2" 512M
    cat > vdb.xml <<EOF
<disk type='file' device='disk'>
  <source file='${PWD}/${VM_NAME}-vdb.qcow2'/>
  <target dev='vdb'/>
</disk>
EOF

    # attach/detach drive in Available state
    sudo virsh attach-device "${VM_NAME}" vdb.xml --live
    sleep 1
    ssh_cmd "${VM_IPADDR}" "functests/run-check-drive-state.sh /dev/vdb Available"
    sudo virsh detach-device "${VM_NAME}" vdb.xml --live
    sleep 1
    ssh_cmd "${VM_IPADDR}" "functests/run-check-drive-not-exist.sh /dev/vdb"

    # attach/detach drive in Available and Ready state
    sudo virsh attach-device "${VM_NAME}" vdb.xml --live
    sleep 1
    ssh_cmd "${VM_IPADDR}" "functests/run-check-drive-state.sh /dev/vdb Available"
    ssh_cmd "${VM_IPADDR}" "./kubectl-direct_csi drive format --drives /dev/vdb"
    sleep 1
    ssh_cmd "${VM_IPADDR}" "functests/run-check-drive-state.sh /dev/vdb Ready"
    sudo virsh detach-device "${VM_NAME}" vdb.xml --live
    sleep 1
    ssh_cmd "${VM_IPADDR}" "functests/run-check-drive-not-exist.sh /dev/vdb"

    # attach/detach drive in Available, Ready, InUse and Terminating state
    sudo virsh attach-device "${VM_NAME}" vdb.xml --live
    sleep 1
    ssh_cmd "${VM_IPADDR}" "functests/run-check-drive-state.sh /dev/vdb Available"
    ssh_cmd "${VM_IPADDR}" "./kubectl-direct_csi drive format --drives /dev/vdb"
    sleep 1
    ssh_cmd "${VM_IPADDR}" "functests/run-check-drive-state.sh /dev/vdb Ready"
    ssh_cmd "${VM_IPADDR}" "functests/run-deploy-minio.sh"
    ssh_cmd "${VM_IPADDR}" "functests/run-check-drive-state.sh /dev/vdb InUse"
    sudo virsh detach-device "${VM_NAME}" vdb.xml --live
    sleep 1
    ssh_cmd "${VM_IPADDR}" "functests/run-check-drive-state.sh /dev/vdb Terminating"

    rm -f vdb.xml "${VM_NAME}-vdb.qcow2"

    stop_test_vm
}

setup_base_image
build_directcsi
run_functional_test
