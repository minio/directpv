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

#
# This script migrates legacy drives and volumes.
#
# Usage:
#   migrate.sh
#

set -e

declare -a plugin_cmd

function run_plugin_cmd() {
    "${plugin_cmd[@]}" "$@"
}

function is_legacy_drive_found() {
    if ! kubectl get --ignore-not-found=true crd directcsidrives.direct.csi.min.io --no-headers -o NAME | grep -q .; then
        return 1
    fi

    exists=$(kubectl get directcsidrives --ignore-not-found=true -o go-template='{{range .items}}{{1}}{{break}}{{end}}')
    [ -n "${exists}" ]
}

function is_legacy_volume_found() {
    if ! kubectl get --ignore-not-found=true crd directcsivolumes.direct.csi.min.io --no-headers -o NAME | grep -q .; then
        return 1
    fi

    exists=$(kubectl get directcsivolumes --ignore-not-found=true -o go-template='{{range .items}}{{1}}{{break}}{{end}}')
    [ -n "${exists}" ]
}

function init() {
    if ! which kubectl >/dev/null 2>&1; then
        echo "kubectl not found; please install"
        exit 255
    fi

    if kubectl directpv --version >/dev/null 2>&1; then
        plugin_cmd=( kubectl directpv )
    elif which kubectl-directpv >/dev/null 2>&1; then
        plugin_cmd=( kubectl-directpv )
    elif ./kubectl-directpv --version >/dev/null 2>&1; then
        plugin_cmd=( ./kubectl-directpv )
    else
        echo "kubectl directpv plugin not found; please install"
        exit 255
    fi
}

function main() {
    if ! is_legacy_drive_found && ! is_legacy_volume_found; then
        return 0
    fi

    if ! run_plugin_cmd migrate; then
        return 1
    fi

    kubectl -n directpv delete pods --all
}

init "$@"
main "$@"
