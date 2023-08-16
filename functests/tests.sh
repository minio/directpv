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

# shellcheck source=/dev/null
source "common.sh"

function run_tests() {
    setup_lvm
    setup_luks
    pod_count=$(( 3 + ACTIVE_NODES ))
    install_directpv "${DIRECTPV_DIR}/kubectl-directpv" "${pod_count}"
    add_drives "${DIRECTPV_DIR}/kubectl-directpv"
    deploy_minio minio.yaml
    test_force_delete
    test_volume_supending "${DIRECTPV_DIR}/kubectl-directpv"
    uninstall_minio "${DIRECTPV_DIR}/kubectl-directpv" minio.yaml
    test_volume_expansion "${DIRECTPV_DIR}/kubectl-directpv" sleep.yaml
    remove_drives "${DIRECTPV_DIR}/kubectl-directpv"
    uninstall_directpv "${DIRECTPV_DIR}/kubectl-directpv" "${pod_count}"
    unmount_directpv
    remove_luks
    remove_lvm
}

run_tests
