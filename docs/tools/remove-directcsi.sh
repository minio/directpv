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
# This script removes direct-csi drives and volumes after taking backup YAMLs
# to directcsidrives.yaml and directcsivolumes.yaml
#

set -e -C -o pipefail

function init() {
    if [[ $# -ne 0 ]]; then
        echo "usage: remove-directcsi.sh"
        echo
        echo "This script removes direct-csi drives and volumes after taking backup YAMLs"
        echo "to directcsidrives.yaml and directcsivolumes.yaml"
        exit 255
    fi

    if ! which kubectl >/dev/null 2>&1; then
        echo "kubectl not found; please install"
        exit 255
    fi
}

# usage: unset_object_finalizers <resource>
function unset_object_finalizers() {
    kubectl get "${1}" -o custom-columns=NAME:.metadata.name --no-headers | while read -r resource_name; do
        kubectl patch "${1}" "${resource_name}" -p '{"metadata":{"finalizers":null}}' --type=merge
    done
}

function main() {
    kubectl get directcsivolumes -o yaml > directcsivolumes.yaml
    kubectl get directcsidrives -o yaml > directcsidrives.yaml

    # unset the finalizers
    unset_object_finalizers "directcsidrives"
    unset_object_finalizers "directcsivolumes"
    
    # delete the resources
    kubectl delete directcsivolumes --all
    kubectl delete directcsidrives --all
}

init "$@"
main "$@"
