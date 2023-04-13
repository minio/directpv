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
# This script drains the DirectPV resources from a selected node
# 
# **CAUTION**
#
# This operation is irreversible and may incur data loss if not used cautiously.
#

set -e -C -o pipefail


function drain() {
    selector="directpv.min.io/node=${2}"

    # unset the finalizers
    kubectl get "${1}" --selector="${selector}" -o custom-columns=NAME:.metadata.name --no-headers | while read -r resource_name; do
        kubectl patch "${1}" "${resource_name}" -p '{"metadata":{"finalizers":null}}' --type=merge
    done
    
    # delete the objects
    kubectl delete "${1}" --selector="${selector}" --ignore-not-found=true
}

function init() {

    if [[ $# -ne 1 ]]; then
        echo "usage: drain.sh <NODE>"
        echo
        echo "This script forcefully removes all the DirectPV resources from the node"
        echo "This operation is irreversible and may incur data loss if not used cautiously."
        exit 255
    fi

    if ! which kubectl >/dev/null 2>&1; then
        echo "kubectl not found; please install"
        exit 255
    fi

    if kubectl get csinode "${1}" -o go-template="{{range .spec.drivers}}{{if eq .name \"directpv-min-io\"}}{{.name}}{{end}}{{end}}" --ignore-not-found | grep -q .; then
        echo "the node is still under use by DirectPV CSI Driver; please remove DirectPV installation from the node to drain"
        exit 255
    fi
}

function main() {
    node="$1"

    drain "directpvvolumes" "${node}"
    drain "directpvdrives" "${node}"
    drain "directpvinitrequests" "${node}"
    kubectl delete directpvnode "${node}" --ignore-not-found=true
}

init "$@"
main "$@"
