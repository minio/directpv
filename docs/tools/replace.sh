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

ME=$(basename "$0"); export ME

export drive_id=""

# usage: get_drive_ids <node> <drive-name>
function get_drive_ids() {
    kubectl get directpvdrives \
            --selector="directpv.min.io/node==${1},directpv.min.io/drive-name==${2}" \
            -o go-template='{{range .items}}{{.metadata.name}} {{end}}'
}

# usage: get_volumes <drive-id>
function get_volumes() {
    kubectl get directpvvolumes \
            --selector="directpv.min.io/drive=${1}" \
            -o go-template='{{range .items}}{{.metadata.name}}{{ " " | print }}{{end}}'
}

# usage: get_pod_name <volume-id>
function get_pod_name() {
    # shellcheck disable=SC2016
    kubectl get directpvvolumes "${1}" \
            -o go-template='{{range $k,$v := .metadata.labels}}{{if eq $k "directpv.min.io/pod.name"}}{{$v}}{{end}}{{end}}'
}

# usage: get_pod_namespace <volume-id>
function get_pod_namespace() {
    # shellcheck disable=SC2016
    kubectl get directpvvolumes "${1}" \
            -o go-template='{{range $k,$v := .metadata.labels}}{{if eq $k "directpv.min.io/pod.namespace"}}{{$v}}{{end}}{{end}}'
}

# usage: get_node_name <drive-id>
function get_node_name() {
    # shellcheck disable=SC2016
    kubectl get directpvdrives "${1}" \
            -o go-template='{{range $k,$v := .metadata.labels}}{{if eq $k "directpv.min.io/node"}}{{$v}}{{end}}{{end}}'
}

# usage: is_uuid input
function is_uuid() {
    [[ "$1" =~ ^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$ ]]
}

# usage: must_get_drive_id <node> <drive-name>
function must_get_drive_id() {
    if [ -z "${1}" ]; then
        print "node argument must be provided for drive name"
        exit 255
    fi
    # shellcheck disable=SC2207
    drive_ids=( $(get_drive_ids "${1}" "${2}") )
    if [ "${#drive_ids[@]}" -eq 0 ]; then
        printf 'drive '%s' on node '%s' not found' "$2" "$1"
        exit 255
    fi
    if [ "${#drive_ids[@]}" -gt 1 ]; then
        printf 'duplicate drive ids found for '%s'' "$2"
        exit 255
    fi
    drive_id="${drive_ids[0]}"
}

function init() {
    if [[ $# -lt 2 || $# -gt 3 ]]; then
        cat <<EOF
NAME:
  ${ME} - This script replaces source drive to destination drive in the specified node.

USAGE:
  ${ME} <SRC-DRIVE> <DEST-DRIVE> [NODE]

ARGUMENTS:
  SRC-DRIVE      Source drive by name or drive ID.
  DEST-DRIVE     Destination drive by name or drive ID.
  NODE           Valid node name. Should be provided if drive name is used to refer source/destination drive.

EXAMPLE:
  # Replace /dev/sdb by /dev/sdc on node worker4.
  $ ${ME} /dev/sdb /dev/sdc worker4

  # Replace detached /dev/sdb with drive ID 1bff96ba-f32e-4493-b95b-897c07d68460 by newly added /dev/sdb with 
  # drive ID 52bf469b-e62e-40b8-a23e-941cd7fe03b3 on worker3.
  $ ${ME} 1bff96ba-f32e-4493-b95b-897c07d68460 52bf469b-e62e-40b8-a23e-941cd7fe03b3
EOF
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
    src_drive="${1#/dev/}"
    dest_drive="${2#/dev/}"
    node="${3}"

    if [ "${src_drive}" == "${dest_drive}" ]; then
        echo "the source and destination drives are same"
        exit 255
    fi

    if ! is_uuid "${src_drive}"; then
        must_get_drive_id "${node}" "${src_drive}"
        src_drive_id="${drive_id}"
    else
        src_drive_id="${src_drive}"
    fi

    if ! is_uuid "${dest_drive}"; then
        must_get_drive_id "${node}" "${dest_drive}"
        dest_drive_id="${drive_id}"
    else
        dest_drive_id="${dest_drive}"
    fi

    if [ "${src_drive_id}" == "${dest_drive_id}" ]; then
        echo "the source and destination drive IDs are same"
        exit 1
    fi

    src_node=$(get_node_name "${src_drive_id}")
    if [ -z "${src_node}" ]; then
        echo "unable to find the node name of the source drive '${src_drive}'"
        exit 1
    fi

    dest_node=$(get_node_name "${dest_drive_id}")
    if [ -z "${dest_node}" ]; then
        echo "unable to find the node name of the destination drive '${dest_drive}'"
        exit 1
    fi

    if [ "${src_node}" != "${dest_node}" ]; then
        echo "the drives are not from the same node"
        exit 1
    fi

    # Cordon source and destination drives
    if ! kubectl directpv cordon "${src_drive_id}" "${dest_drive_id}"; then
        echo "unable to cordon drives"
        exit 1
    fi

    # Cordon kubernetes node
    if ! kubectl cordon "${src_node}"; then
        echo "unable to cordon node '${src_node}'"
        exit 1
    fi

    mapfile -t volumes < <(get_volumes "${src_drive_id}")
    IFS=' ' read -r -a volumes_arr <<< "${volumes[@]}"
    for volume in "${volumes_arr[@]}"; do
        pod_name=$(get_pod_name "${volume}")
        pod_namespace=$(get_pod_namespace "${volume}")

        if ! kubectl delete pod "${pod_name}" --namespace "${pod_namespace}"; then
            echo "unable to delete pod '${pod_name}' using volume '${volume}'; please delete the pod manually"
        fi
    done

    if [ "${#volumes_arr[@]}" -gt 0 ]; then
        # Wait for associated DirectPV volumes to be unbound
        while kubectl directpv list volumes --no-headers "${volumes_arr[@]}" | grep -q Bounded; do
            echo "...waiting for volumes to be unbound"
            sleep 10
        done
    else
        echo "no volumes found in source drive '${src_drive}' on node '${src_node}'"
    fi

    # Run move command
    kubectl directpv move "${src_drive_id}" "${dest_drive_id}"

    # Uncordon destination drive
    kubectl directpv uncordon "${dest_drive_id}"

    # Uncordon kubernetes node
    kubectl uncordon "${src_node}"
}

init "$@"
main "$@"
