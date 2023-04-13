#!/usr/bin/env bash
# This file is part of MinIO DirectPV
# Copyright (c) 2023 MinIO, Inc.
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

declare -a NODE_VM_NAMES NODE_IMAGES NODE_WORK_DISK_IMAGES NODE_VM_IPS
for (( i = 1; i <= NODE_COUNT; ++i )); do
    # shellcheck disable=SC2206
    NODE_VM_NAMES=( ${NODE_VM_NAMES[@]} "${TEST_ID}_node_${i}" )

    # shellcheck disable=SC2206
    NODE_IMAGES=( ${NODE_IMAGES[@]} "${TEST_ID}_node_${i}.qcow2c" )

    # shellcheck disable=SC2206
    NODE_WORK_DISK_IMAGES=( ${NODE_WORK_DISK_IMAGES[@]} "${TEST_ID}_node_${i}.work-disk.qcow2c" )
done

function scp_cmd() {
    scp -q -o GSSAPIAuthentication=no -o StrictHostKeyChecking=no -i "${RSA_PRIVATE_KEY}" -p "$@"
}

function ssh_cmd() {
    ssh -q -o GSSAPIAuthentication=no -o StrictHostKeyChecking=no -i "${RSA_PRIVATE_KEY}" -l root "$@"
}

# get_ipaddr <vm-name>
function get_ipaddr() {
    vm_name="$1"
    for (( i = 0; i < 5; ++i )); do
        ipaddr=$(sudo virsh domifaddr "${vm_name}" | awk '$4 ~ /[0-9]$/ { split($4,a,"/"); print a[1] }')
        if [ -n "${ipaddr}" ]; then
            echo "${ipaddr}"
            return 0
        fi

        sleep 24
    done

    return 1
}

function _create_cloud_image() {
    if [ -f "${BASE_IMAGE}" ]; then
        return
    fi

    echo "* Create base VM image. This may take longer time."

    cp --force --dereference -p "${CLOUD_IMAGE}" "${BASE_IMAGE}"
    ssh-keygen -q -f "${RSA_PRIVATE_KEY}" -N ''
    sudo virt-customize --quiet -a "${BASE_IMAGE}" --ssh-inject "root:file:${RSA_PUBLIC_KEY}" --root-password password:password --uninstall cloud-init --selinux-relabel
    sudo virt-install --name "${BASE_VM_NAME}" --memory 3072 --vcpus 2 --disk "${BASE_IMAGE}" --import --osinfo "${CLOUD_IMAGE_OSINFO}" --print-xml | cat > "${BASE_VM_NAME}.xml"
    sudo virsh define "${BASE_VM_NAME}.xml" >/dev/null 2>&1
    sudo virsh start "${BASE_VM_NAME}" >/dev/null 2>&1
    rm -f "${BASE_VM_NAME}.xml"
    sleep 3
    ipaddr=$(get_ipaddr "${BASE_VM_NAME}")
    setup_sh="${TEST_ID}_setup.sh"
    cat > "${setup_sh}" <<EOF
#!/bin/bash

function main() {
    set -ex
    sed -i -e s:SELINUX=enforcing:SELINUX=disabled:g /etc/selinux/config
    grubby --update-kernel ALL --args selinux=0
    if which firewalld >/dev/null 2>&1; then
        systemctl disable firewalld
        systemctl remove firewalld
    fi
    dnf_cmd=dnf
    if ! which dnf >/dev/null 2>&1; then dnf_cmd=yum; fi
    \${dnf_cmd} update -y
    \${dnf_cmd} install podman -y
    if ! which python >/dev/null 2>&1; then
        if ! which python3 >/dev/null 2>&1; then
            \${dnf_cmd} install python3 -y
        fi
        (cd /usr/bin && ln -s python3 python)
    fi
}

if ! main >/tmp/.output 2>&1; then
    echo "execution failed"
    echo "trace:"
    cat /tmp/.output
    exit 1
fi
EOF
    chmod a+x "${setup_sh}"
    scp_cmd "${setup_sh}" "root@${ipaddr}:"
    ssh_cmd "$ipaddr" "./${setup_sh}"
    rm -f "${setup_sh}"
    sudo virsh shutdown "${BASE_VM_NAME}" >/dev/null 2>&1
    sleep 10
    sudo virsh undefine "${BASE_VM_NAME}" >/dev/null 2>&1
}

function create_cloud_image() {
    if _create_cloud_image; then
        return 0
    fi

    rm -f "${BASE_VM_NAME}.xml"

    if sudo virsh --quiet dominfo "${BASE_VM_NAME}" >/dev/null 2>&1; then
        sudo virsh destroy "${BASE_VM_NAME}" >/dev/null 2>&1 || true
        sudo virsh undefine "${BASE_VM_NAME}" >/dev/null 2>&1 || true
    fi

    rm -f "${BASE_IMAGE}"

    return 1
}

function create_vm_images() {
    echo "* Create tests VM images. This may take longer time."

    cp --force --dereference -p "${BASE_IMAGE}" "${MASTER_IMAGE}"
    qemu-img create -q -f qcow2 "${MASTER_WORK_DISK_IMAGE}" 512M

    for node_image in "${NODE_IMAGES[@]}"; do
        cp --force --dereference -p "${BASE_IMAGE}" "${node_image}"
    done
    for node_work_disk_image in "${NODE_WORK_DISK_IMAGES[@]}"; do
        qemu-img create -q -f qcow2 "${node_work_disk_image}" 512M
    done
}

function start_master_vm() {
    echo "* Start master VM"

    sudo virt-install --name "${MASTER_VM_NAME}" --memory 3072 --vcpus 2 --disk "${MASTER_IMAGE}" --disk "${MASTER_WORK_DISK_IMAGE}" --import --osinfo "${CLOUD_IMAGE_OSINFO}" --print-xml | cat > "${MASTER_VM_NAME}.xml"
    sudo virsh define "${MASTER_VM_NAME}.xml" >/dev/null 2>&1
    sudo virsh start "${MASTER_VM_NAME}" >/dev/null 2>&1
    rm -f "${MASTER_VM_NAME}.xml"
    sleep 3
    MASTER_VM_IP=$(get_ipaddr "${MASTER_VM_NAME}")
    ssh_cmd "${MASTER_VM_IP}" "hostnamectl set-hostname master"
}

function start_node_vms() {
    echo "* Start node VMs. This may take longer time."

    i=0
    n=1
    for node_vm_name in "${NODE_VM_NAMES[@]}"; do
        sudo virt-install --name "${node_vm_name}" --memory 3072 --vcpus 2 --disk "${NODE_IMAGES[$i]}" --disk "${NODE_WORK_DISK_IMAGES[$i]}" --import --osinfo "${CLOUD_IMAGE_OSINFO}" --print-xml | cat > "${node_vm_name}.xml"
        sudo virsh define "${node_vm_name}.xml" >/dev/null 2>&1
        sudo virsh start "${node_vm_name}" >/dev/null 2>&1
        rm -f "${node_vm_name}.xml"
        sleep 3
        node_vm_ip=$(get_ipaddr "${node_vm_name}")
        # shellcheck disable=SC2206
        NODE_VM_IPS=( ${NODE_VM_IPS[@]} "${node_vm_ip}" )
        ssh_cmd "${node_vm_ip}" "hostnamectl set-hostname node${n}"
        (( ++i ))
        (( ++n ))
    done
}

function update_hosts_file() {
    echo "* Update /etc/hosts file in test VMs"

    setup_sh="${TEST_ID}_setup.sh"
    rm -f "${setup_sh}"
    echo "echo ${MASTER_VM_IP} master >> /etc/hosts" >> "${setup_sh}"
    i=1
    for node_vm_ip in "${NODE_VM_IPS[@]}"; do
        echo "echo ${node_vm_ip} node${i} >> /etc/hosts" >> "${setup_sh}"
        (( ++i ))
    done
    chmod a+x "${setup_sh}"

    scp_cmd "${setup_sh}" "root@${MASTER_VM_IP}:"
    ssh_cmd "${MASTER_VM_IP}" "./${setup_sh}"
    for node_vm_ip in "${NODE_VM_IPS[@]}"; do
        scp_cmd "${setup_sh}" "root@${node_vm_ip}:"
        ssh_cmd "${node_vm_ip}" "./${setup_sh}"
    done
    rm -f "${setup_sh}"
}

function install_k3s_master() {
    echo "* Install k3s master. This may take longer time."

    setup_sh="${TEST_ID}_setup.sh"
    cat > "${setup_sh}" <<EOF
#!/bin/bash

function main() {
    set -ex
    curl --silent --location --insecure --fail --output k3s.sh https://get.k3s.io
    bash -ex k3s.sh
}

if ! main >/tmp/.output 2>&1; then
    echo "execution failed"
    echo "trace:"
    cat /tmp/.output
    exit 1
fi
EOF
    chmod a+x "${setup_sh}"
    scp_cmd "${setup_sh}" "root@${MASTER_VM_IP}:"
    ssh_cmd "${MASTER_VM_IP}" "./${setup_sh}"
}

function install_k3s_nodes() {
    echo "* Install k3s nodes. This may take longer time."

    scp_cmd "root@${MASTER_VM_IP}:/var/lib/rancher/k3s/server/node-token" "${TEST_ID}_node-token"
    setup_sh="${TEST_ID}_setup.sh"
    cat > "${setup_sh}" <<EOF
#!/bin/bash

function main() {
    set -ex
    curl --silent --location --insecure --fail --output k3s.sh https://get.k3s.io
    K3S_URL=https://master:6443 K3S_TOKEN=\$(cat ${TEST_ID}_node-token) bash -ex k3s.sh
}

if ! main >/tmp/.output 2>&1; then
    echo "execution failed"
    echo "trace:"
    cat /tmp/.output
    exit 1
fi
EOF
    chmod a+x "${setup_sh}"

    for node_vm_ip in "${NODE_VM_IPS[@]}"; do
        scp_cmd "${TEST_ID}_node-token" "${setup_sh}" "root@${node_vm_ip}:"
    done

    rm -f "${TEST_ID}_node-token" "${setup_sh}"

    for node_vm_ip in "${NODE_VM_IPS[@]}"; do
        ssh_cmd "${node_vm_ip}" "./${setup_sh}"
    done
}

function build_docker_images() {
    echo "* Build DirectPV image. This may take longer time."

    (cd "${DIRECTPV_DIR}" && ./build.sh)

    tag="$("${DIRECTPV_DIR}/kubectl-directpv" --version | awk '{ print $NF }')"
    directpv_image_tar="${TEST_ID}_directpv_${tag}.tar"
    directpv_image_tar_xz="${directpv_image_tar}.xz"

    sleep_dockerfile="${TEST_ID}_Dockerfile.sleep"
    cat <<EOF > "${sleep_dockerfile}"
FROM alpine:latest
RUN echo 'while true; do echo I am in a sleep loop; sleep 3; done' > sleep.sh
ENTRYPOINT sh sleep.sh
EOF
    sleep_image_tar="${TEST_ID}_sleep_v0.0.1.tar"
    sleep_image_tar_xz="${sleep_image_tar}.xz"

    setup_sh="${TEST_ID}_setup.sh"
    cat > "${setup_sh}" <<EOF
#!/bin/bash

function main() {
    set -ex

    podman build -t quay.io/minio/directpv:${tag} .
    podman save --output ${directpv_image_tar} quay.io/minio/directpv:${tag}
    /usr/local/bin/ctr images import --digests --base-name quay.io/minio/directpv ${directpv_image_tar}
    xz -z ${directpv_image_tar}

    podman build -t example.org/test/sleep:v0.0.1 -f ${sleep_dockerfile}
    podman save --quiet --output ${sleep_image_tar} example.org/test/sleep:v0.0.1
    /usr/local/bin/ctr images import --digests --base-name example.org/test/sleep ${sleep_image_tar}
    xz -z ${sleep_image_tar}
}

if ! main >/tmp/.output 2>&1; then
    echo "execution failed"
    echo "trace:"
    cat /tmp/.output
    exit 1
fi
EOF
    chmod a+x "${setup_sh}"
    scp_cmd "${DIRECTPV_DIR}/CREDITS" "${DIRECTPV_DIR}/LICENSE" "${DIRECTPV_DIR}/centos.repo" "${DIRECTPV_DIR}/directpv" "${DIRECTPV_DIR}/kubectl-directpv" "${DIRECTPV_DIR}/Dockerfile" "${sleep_dockerfile}" "${setup_sh}" "root@${MASTER_VM_IP}:"
    ssh_cmd "${MASTER_VM_IP}" "./${setup_sh}"

    if [ "${NODE_COUNT}" -eq 0 ]; then
        return
    fi

    scp_cmd "root@${MASTER_VM_IP}:{${directpv_image_tar_xz},${sleep_image_tar_xz}}" "."
    setup_sh="${TEST_ID}_setup.sh"
    cat > "${setup_sh}" <<EOF
#!/bin/bash

function main() {
    set -ex

    xz -d ${directpv_image_tar_xz}
    /usr/local/bin/ctr images import --digests --base-name quay.io/minio/directpv ${directpv_image_tar}

    xz -d ${sleep_image_tar_xz}
    /usr/local/bin/ctr images import --digests --base-name example.org/test/sleep ${sleep_image_tar}
}

if ! main >/tmp/.output 2>&1; then
    echo "execution failed"
    echo "trace:"
    cat /tmp/.output
    exit 1
fi
EOF
    chmod a+x "${setup_sh}"

    for node_vm_ip in "${NODE_VM_IPS[@]}"; do
        scp_cmd "${directpv_image_tar_xz}" "${sleep_image_tar_xz}" "${setup_sh}" "root@${node_vm_ip}:"
        ssh_cmd "${node_vm_ip}" "./${setup_sh}"
    done
    rm -f "${directpv_image_tar_xz}" "${sleep_image_tar_xz}"
}

function cleanup_directpv() {
    ssh_cmd "${MASTER_VM_IP}" "mount | awk '/direct|pvc-/ {print \$3}' > /tmp/.output; if grep -q . /tmp/.output; then sudo umount -fl \$(cat /tmp/.output); fi"
    for node_vm_ip in "${NODE_VM_IPS[@]}"; do
        ssh_cmd "${node_vm_ip}" "mount | awk '/direct|pvc-/ {print \$3}' > /tmp/.output; if grep -q . /tmp/.output; then sudo umount -fl \$(cat /tmp/.output); fi"
    done

    ssh_cmd "${MASTER_VM_IP}" "rm -fr /var/lib/directpv /var/lib/direct-csi"
    for node_vm_ip in "${NODE_VM_IPS[@]}"; do
        ssh_cmd "${node_vm_ip}" "rm -fr /var/lib/directpv /var/lib/direct-csi"
    done
}

function run_func_tests() {
    echo "********* Run functional tests *************"

    ssh_cmd "${MASTER_VM_IP}" "mkdir -p .kube && cp /etc/rancher/k3s/k3s.yaml .kube/config"
    ssh_cmd "${MASTER_VM_IP}" "mkdir -p functests"
    scp_cmd -r ./*.sh ./*.yaml "root@${MASTER_VM_IP}:functests"

    echo "### Run basic tests"
    cleanup_directpv
    ssh_cmd "${MASTER_VM_IP}" "ACTIVE_NODES=${ACTIVE_NODES} ./functests/run-tests.sh"
    echo

    echo "### Run migration tests"
    cleanup_directpv
    ssh_cmd "${MASTER_VM_IP}" "ACTIVE_NODES=${ACTIVE_NODES} ./functests/run-migration-tests.sh v3.2.2"
    echo
}

function remove_vms() {
    for node_vm_name in "${NODE_VM_NAMES[@]}"; do
        sudo virsh destroy "${node_vm_name}" >/dev/null 2>&1 || true
    done
    sudo virsh destroy "${MASTER_VM_NAME}" >/dev/null 2>&1 || true

    for node_vm_name in "${NODE_VM_NAMES[@]}"; do
        sudo virsh undefine "${node_vm_name}" >/dev/null 2>&1 || true
    done
    sudo virsh undefine "${MASTER_VM_NAME}" >/dev/null 2>&1 || true

    # shellcheck disable=SC2086
    rm -fr ${TEST_ID}_*
}

function main() {
    create_cloud_image
    create_vm_images
    start_master_vm
    start_node_vms
    update_hosts_file
    install_k3s_master
    install_k3s_nodes
    build_docker_images
    run_func_tests
}

rc=0
if ! main; then
    rc=$?
fi

remove_vms
exit ${rc}
