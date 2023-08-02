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

set -e -C -o pipefail

declare NODE

function delete_resource() {
    resource="$1"
    selector="directpv.min.io/node=${NODE}"

    # unset the finalizers
    kubectl get "${resource}" --selector="${selector}" -o custom-columns=NAME:.metadata.name --no-headers | while read -r name; do
        kubectl patch "${resource}" "${name}" -p '{"metadata":{"finalizers":null}}' --type=merge
    done
    
    # delete the objects
    kubectl delete "${resource}" --selector="${selector}" --ignore-not-found=true
}

function init() {
    if [[ $# -ne 1 ]]; then
        cat <<EOF
usage: remove-node.sh <NODE>

This script forcefully removes all the DirectPV resources from the node.
CAUTION: Remove operation is irreversible and may incur data loss if not used cautiously.
EOF
        exit 255
    fi

    if ! which kubectl >/dev/null 2>&1; then
        echo "kubectl not found; please install"
        exit 255
    fi

    NODE="$1"

    if kubectl get --ignore-not-found=true csinode "${NODE}" -o go-template='{{range .spec.drivers}}{{if eq .name "directpv-min-io"}}{{.name}}{{break}}{{end}}{{end}}' | grep -q .; then
        echo "node ${NODE} is still in use; remove node ${NODE} from DirectPV DaemonSet and try again"
        exit 255
    fi
}

function main() {
    delete_resource directpvvolumes
    delete_resource directpvdrives
    delete_resource directpvinitrequests
    kubectl delete directpvnode "${NODE}" --ignore-not-found=true
}

init "$@"
main "$@"
