#!/usr/bin/env bash
#
# This file is part of MinIO DirectPV
# Copyright (c) 2024 MinIO, Inc.
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
# This script repairs faulty drives
#

set -e

ME=$(basename "$0"); export ME

declare -a drive_ids

# usage: is_uuid <value>
function is_uuid() {
    [[ "$1" =~ ^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$ ]]
}

# usage: get_suspend_value <drive-id>
function get_suspend_value() {
    # shellcheck disable=SC2016
    kubectl get directpvvolumes "${1}" \
            -o go-template='{{range $k,$v := .metadata.labels}}{{if eq $k "directpv.min.io/suspend"}}{{$v}}{{end}}{{end}}'
}

# usage: is_suspended <drive-id>
function is_suspended() {
    value=$(get_suspend_value "${1}")
    [[ "${value,,}" = "true" ]]
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

function init() {
    if [[ $# -eq 0 ]]; then
        cat <<EOF
NAME:
  ${ME} - This script repairs faulty drives.

USAGE:
  ${ME} <DRIVE-ID> ...

ARGUMENTS:
  DRIVE-ID      Faulty drive ID.

EXAMPLE:
  # Repair drive af3b8b4c-73b4-4a74-84b7-1ec30492a6f0.
  $ ${ME} af3b8b4c-73b4-4a74-84b7-1ec30492a6f0
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

    for drive in "$@"; do
        if ! is_uuid "${drive}"; then
            echo "invalid drive ID ${drive}"
            exit 255
        fi
        if [[ ! ${drive_ids[*]} =~ ${drive} ]]; then
            drive_ids+=( "${drive}" )
        fi
    done
}

# usage: repair <drive-id>
function repair() {
    drive_id="$1"

    pods_deleted=true
    if ! is_suspended "${drive_id}"; then
        kubectl directpv suspend "${drive_id}"

        # shellcheck disable=SC2207
        volumes=( $(get_volumes "${drive_id}") )
        for volume in "${volumes[@]}"; do
            pod_name=$(get_pod_name "${volume}")
            pod_namespace=$(get_pod_namespace "${volume}")

            if ! kubectl delete pod "${pod_name}" --namespace "${pod_namespace}"; then
                echo "unable to delete pod '${pod_name}' using volume '${volume}'; please delete the pod manually"
                pods_deleted=false
            fi
        done
    else
        echo "drive ${drive_id} already suspended"
    fi

    if [ "${pods_deleted}" == "true" ]; then
        kubectl directpv repair "${drive_id}"
    else
        echo "delete pods manually and retry again for drive ${drive_id}"
    fi
}

function main() {
    for drive in "${drive_ids[@]}"; do
        repair "${drive}"
    done
}

init "$@"
main "$@"
