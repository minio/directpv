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
# This script sets up libvirt in Travis CI and runs ./run-functests-on-centos7-vm.sh
#

set -ex

if ! cat /sys/module/kvm_*/parameters/nested | grep -q Y; then
    echo "Nested virtualization NOT supported"
    exit 1
fi

sudo systemctl enable --now libvirtd

state=$(sudo virsh net-list --all | awk '$1 ~ /default/ { print $2 }')
if [ "$state" != "active" ]; then
    ## Assume default bridge interface is virbr0
    virbr=virbr0

    ## Show output for debugging
    sudo cat /etc/libvirt/qemu/networks/default.xml
    sudo ifconfig -a

    if ! sudo virsh net-start default; then
        sudo ifconfig "$virbr" down
        sudo brctl delbr "$virbr"
        sudo virsh net-start default
    fi

    state=$(sudo virsh net-list --all | awk '$1 ~ /default/ { print $2 }')
    if [ "$state" != "active" ]; then
        echo "Unable to start bridge network"
        exit 1
    fi
fi

./run-functests-on-centos7-vm.sh
