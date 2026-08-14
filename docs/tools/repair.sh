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
force_flag=""
output_format=""

# usage: is_uuid <value>
function is_uuid() {
    [[ "$1" =~ ^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$ ]]
}

# usage: get_suspend_value <drive-id>
function get_suspend_value() {
    # shellcheck disable=SC2016
    kubectl get directpvdrives "${1}" \
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
  ${ME} [FLAGS] <DRIVE-ID> ...

ARGUMENTS:
  DRIVE-ID      Faulty drive ID.

FLAGS:
  --force               Force log zeroing.
  -o, --output FORMAT   Generate repair job manifest of drives in yaml|json
                        format. In this mode no drive is suspended, no pod is
                        deleted and no repair job is created; commands to run
                        the prerequisites are printed to standard error.

EXAMPLE:
  # Repair drive af3b8b4c-73b4-4a74-84b7-1ec30492a6f0.
  $ ${ME} af3b8b4c-73b4-4a74-84b7-1ec30492a6f0

  # Generate repair job manifest of drive af3b8b4c-73b4-4a74-84b7-1ec30492a6f0.
  $ ${ME} --output yaml af3b8b4c-73b4-4a74-84b7-1ec30492a6f0 > repair.yaml
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

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --force)
                force_flag="--force"
                shift
                ;;
            -o|--output)
                if [[ $# -lt 2 ]]; then
                    echo "no value provided to $1 flag"
                    exit 255
                fi
                output_format="$2"
                shift 2
                ;;
            -o=*|--output=*)
                output_format="${1#*=}"
                shift
                ;;
            *)
                if ! is_uuid "$1"; then
                    echo "invalid drive ID $1"
                    exit 255
                fi
                if [[ ! ${drive_ids[*]} =~ $1 ]]; then
                    drive_ids+=( "$1" )
                fi
                shift
                ;;
        esac
    done

    # Check if we have at least one drive ID
    if [[ ${#drive_ids[@]} -eq 0 ]]; then
        echo "no drive IDs provided"
        exit 255
    fi

    case "${output_format}" in
        ""|yaml|json)
            ;;
        *)
            echo "invalid output format ${output_format}; must be one of yaml|json"
            exit 255
            ;;
    esac
}

# usage: repair <drive-id>
function repair() {
    drive_id="$1"

    pods_deleted=true
    if ! is_suspended "${drive_id}"; then
        kubectl directpv suspend drives "${drive_id}" --dangerous

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
        if [[ -n "${force_flag}" ]]; then
            kubectl directpv repair "${drive_id}" "${force_flag}"
        else
            kubectl directpv repair "${drive_id}"
        fi
    else
        echo "delete pods manually and retry again for drive ${drive_id}"
    fi
}

# usage: print_prerequisites <drive-id>
#
# Prints the commands to be run before applying the generated manifests. They go
# to standard error to keep the manifests in standard output pipeable.
function print_prerequisites() {
    drive_id="$1"

    if is_suspended "${drive_id}"; then
        echo "# drive ${drive_id} is already suspended" >&2
        return 0
    fi

    {
        echo "# drive ${drive_id} is not suspended; run below commands before applying the manifests"
        echo "kubectl directpv suspend drives ${drive_id} --dangerous"
    } >&2

    if ! volume_list=$(get_volumes "${drive_id}" 2>/dev/null) || [[ -z "${volume_list}" ]]; then
        echo "# delete pods using volumes of this drive, if any" >&2
        return 0
    fi

    for volume in ${volume_list}; do
        pod_name=$(get_pod_name "${volume}" 2>/dev/null) || pod_name=""
        pod_namespace=$(get_pod_namespace "${volume}" 2>/dev/null) || pod_namespace=""

        if [[ -z "${pod_name}" ]] || [[ -z "${pod_namespace}" ]]; then
            echo "# delete the pod using volume ${volume} manually" >&2
            continue
        fi

        echo "kubectl delete pod ${pod_name} --namespace ${pod_namespace}" >&2
    done
}

function generate_manifests() {
    for drive_id in "${drive_ids[@]}"; do
        print_prerequisites "${drive_id}"
    done

    if [[ -n "${force_flag}" ]]; then
        kubectl directpv repair "${drive_ids[@]}" "${force_flag}" --output "${output_format}"
    else
        kubectl directpv repair "${drive_ids[@]}" --output "${output_format}"
    fi
}

function main() {
    if [[ -n "${output_format}" ]]; then
        generate_manifests
        return
    fi

    for drive in "${drive_ids[@]}"; do
        repair "${drive}"
    done
}

init "$@"
main "$@"
