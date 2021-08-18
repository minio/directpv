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

declare -a loops=("loop0" "loop1" "loop2" "loop3")
for loop in "${loops[@]}"; do
    truncate --size=1G /tmp/${loop}.img
    losetup -f /tmp/${loop}.img
done

sleep 2

losetup -a

for loop in "${loops[@]}"; do
    lp=$(losetup -n -O NAME -j /tmp/${loop}.img)
    pvcreate ${lp}
done

sleep 2

pvdisplay

vgcreate vg0 $(sudo pvs --noheadings  --rows --separator ' ' -o NAME)
vgdisplay vg0

sleep 2

declare -a lvs=("lv0" "lv1" "lv2" "lv3")
for lv in "${lvs[@]}"; do
  lvcreate -L 512MiB -n ${lv} vg0
done

sleep 3

lvdisplay

lsblk -a
