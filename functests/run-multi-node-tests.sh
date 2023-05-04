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

# Environment variables:
#   - GITHUB_ACTIONS=true
#     If set this tests will be skipped.

if [ "${GITHUB_ACTIONS}" == "true" ]; then
    exit 0
fi

ME=$(basename "$0"); export ME

if [ -n "$1" ]; then
    arg1="$(realpath --canonicalize-missing --no-symlinks "$1")"
fi

cd "$(dirname "$0")" || exit 255
DIRECTPV_DIR="$(cd .. && echo "$PWD")"
export DIRECTPV_DIR

if ! which virt-customize >/dev/null 2>&1; then
    echo "virt-customize not found; please install"
    exit 255
fi

if ! which virt-install >/dev/null 2>&1; then
    echo "virt-install not found; please install"
    exit 255
fi

if ! which virsh >/dev/null 2>&1; then
    echo "virsh not found; please install"
    exit 255
fi

if ! which qemu-img >/dev/null 2>&1; then
    echo "qemu-img not found; please install"
    exit 255
fi

if [ $# -ne 3 ]; then
    cat <<EOF
USAGE:
  ${ME} <RHEL-CLOUD-IMAGE> <CLOUD-IMAGE-OSINFO> <NODE-COUNT>

ARGUMENTS:
  RHEL-CLOUD-IMAGE      Path to RHEL/CentOS/AlmaLinux 7/8/9 cloud image.
  CLOUD-IMAGE-OSINFO    RHEL/CentOS/AlmaLinux 7/8/9 name for virt-install. Check using '$ virt-install --osinfo list'.
  NODE-COUNT            Number of test nodes to be created.

EXAMPLE:
  # Download CentOS 7 cloud image if not already downloaded.
  $ curl --location --insecure --output CentOS-7-x86_64-GenericCloud-2211.qcow2c https://cloud.centos.org/centos/7/images/CentOS-7-x86_64-GenericCloud-2211.qcow2c

  # Build DirectPV if needed
  $ ./build.sh

  # Run this script
  $ ${ME} CentOS-7-x86_64-GenericCloud-2211.qcow2c centos7 2
EOF
        exit 255
fi

export CLOUD_IMAGE="${arg1}"
if [ ! -f "${CLOUD_IMAGE}" ]; then
    echo "RHEL cloud image ${CLOUD_IMAGE} not found"
    exit 255
fi

export CLOUD_IMAGE_OSINFO="$2"

NODE_COUNT="$3"
if [ -z "${NODE_COUNT}" ] || [ "${NODE_COUNT}" -le 0 ]; then
    echo "invalid node count ${NODE_COUNT}"
    exit 255
fi
export ACTIVE_NODES=${NODE_COUNT}
(( NODE_COUNT-- ))
export NODE_COUNT

id="$(date +%Y%m%d%H%M%S)"
export TEST_ID="test_${id}"
export BASE_VM_NAME="${TEST_ID}_base"
iname="$(basename "${CLOUD_IMAGE}")"
export BASE_IMAGE="Test-Base-${iname}"
export MASTER_VM_NAME="${TEST_ID}_master"
export MASTER_IMAGE="${MASTER_VM_NAME}.qcow2c"
export MASTER_WORK_DISK_IMAGE="${MASTER_VM_NAME}.work-disk.qcow2c"
export MASTER_VM_IP=

export RSA_PRIVATE_KEY="${BASE_IMAGE}_rsa"
export RSA_PUBLIC_KEY="${BASE_IMAGE}_rsa.pub"

(cd "${DIRECTPV_DIR}" && ./build.sh)

sudo -E ./execute.sh multi-node-tests.sh
