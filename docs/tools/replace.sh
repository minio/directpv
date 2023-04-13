#!/usr/bin/env bash
#
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
# This script replaces source drive to destination drive in the specified node
#

set -e

# usage: get_drive_id <node> <drive-name>
function get_drive_id() {
    kubectl get directpvdrives \
            --selector="directpv.min.io/node==${1},directpv.min.io/drive-name==${2}" \
            -o go-template='{{range .items}}{{.metadata.name}}{{end}}'
}

# usage: get_volumes <drive-id>
function get_volumes() {
    kubectl get directpvvolumes \
            --selector="directpv.min.io/drive=${1}" \
            -o go-template='{{range .items}}{{.metadata.name}}{{ " " | print }}{{end}}'
}

# usage: get_pod_name <volume>
function get_pod_name() {
    # shellcheck disable=SC2016
    kubectl get directpvvolumes "${1}" \
            -o go-template='{{range $k,$v := .metadata.labels}}{{if eq $k "directpv.min.io/pod.name"}}{{$v}}{{end}}{{end}}'
}

# usage: get_pod_namespace <volume>
function get_pod_namespace() {
    # shellcheck disable=SC2016
    kubectl get directpvvolumes "${1}" \
            -o go-template='{{range $k,$v := .metadata.labels}}{{if eq $k "directpv.min.io/pod.namespace"}}{{$v}}{{end}}{{end}}'
}

function init() {
    if [[ $# -eq 4 ]]; then
        echo "usage: replace.sh <NODE> <SRC-DRIVE> <DEST-DRIVE>"
        echo
        echo "This script replaces source drive to destination drive in the specified node"
        exit 255
    fi

    if ! which kubectl >/dev/null 2>&1; then
        echo "kubectl not found; please install"
        exit 255
    fi

    if ! kubectl directpv --version >/dev/null 2>&1; then
        echo "kubectl directpv not found; please install"
        exit 255
    fi
}

function main() {
    node="$1"
    src_drive="${2#/dev/}"
    dest_drive="${3#/dev/}"

    # Get source drive ID
    src_drive_id=$(get_drive_id "${node}" "${src_drive}")
    if [ -z "${src_drive_id}" ]; then
        echo "source drive ${src_drive} on node ${node} not found"
        exit 1
    fi

    # Get destination drive ID
    dest_drive_id=$(get_drive_id "${node}" "${dest_drive}")
    if [ -z "${dest_drive_id}" ]; then
        echo "destination drive ${dest_drive} on node ${node} not found"
        exit 1
    fi

    # Cordon source and destination drives
    if ! kubectl directpv cordon "${src_drive_id}" "${dest_drive_id}"; then
        echo "unable to cordon drives"
        exit 1
    fi

    # Cordon kubernetes node
    if ! kubectl cordon "${node}"; then
        echo "unable to cordon node ${node}"
        exit 1
    fi

    mapfile -t volumes < <(get_volumes "${src_drive_id}")
    IFS=' ' read -r -a volumes_arr <<< "${volumes[@]}"
    for volume in "${volumes_arr[@]}"; do
        pod_name=$(get_pod_name "${volume}")
        pod_namespace=$(get_pod_namespace "${volume}")

        if ! kubectl delete pod "${pod_name}" --namespace "${pod_namespace}"; then
            echo "unable to delete pod ${pod_name} using volume ${volume}"
            exit 1
        fi
    done

    if [ "${#volumes_arr[@]}" -gt 0 ]; then
        # Wait for associated DirectPV volumes to be unbound
        while kubectl directpv list volumes --no-headers "${volumes_arr[@]}" | grep -q Bounded; do
            echo "...waiting for volumes to be unbound"
            sleep 10
        done
    else
        echo "no volumes found in source drive ${src_drive} on node ${node}"
    fi

    # Run move command
    kubectl directpv move "${src_drive_id}" "${dest_drive_id}"

    # Uncordon destination drive
    kubectl directpv uncordon "${dest_drive_id}"

    # Uncordon kubernetes node
    kubectl uncordon "${node}"
}

init "$@"
main "$@"
