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

declare NAME
declare -a DRIVE_LABELS

function init() {
    if [[ $# -lt 2 ]]; then
        cat <<EOF
USAGE:
  create-storage-class.sh <NAME> <DRIVE-LABELS> ...

ARGUMENTS:
  NAME           new storage class name.
  DRIVE-LABELS   drive labels to be attached.

EXAMPLE:
  # Create new storage class 'fast-tier-storage' with drive labels 'directpv.min.io/tier: fast'
  $ create-storage-class.sh fast-tier-storage 'directpv.min.io/tier: fast'

  # Create new storage class with more than one drive label
  $ create-storage-class.sh fast-tier-unique 'directpv.min.io/tier: fast' 'directpv.min.io/volume-claim-id: bcea279a-df70-4d23-be41-9490f9933004'
EOF
        exit 255
    fi

    NAME="$1"
    shift
    DRIVE_LABELS=( "$@" )

    for val in "${DRIVE_LABELS[@]}"; do
        if ! [[ "${val}" =~ ^directpv.min.io/.* ]]; then
            echo "invalid label ${val}; label must start with 'directpv.min.io/'"
            exit 255
        fi
        if [[ "${val}" =~ ^directpv.min.io/volume-claim-id:.* ]] && ! [[ "${val#directpv.min.io/volume-claim-id: }" =~ ^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$ ]]; then
            echo "invalid volume-claim-id value; the value must be UUID as textual representation mentioned in https://en.wikipedia.org/wiki/Universally_unique_identifier#Textual_representation"
            exit 255
        fi
    done

    if ! which kubectl >/dev/null 2>&1; then
        echo "kubectl not found; please install"
        exit 255
    fi
}

function main() {
    kubectl apply -f - <<EOF
allowVolumeExpansion: true
allowedTopologies:
- matchLabelExpressions:
  - key: directpv.min.io/identity
    values:
    - directpv-min-io
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  finalizers:
  - foregroundDeletion
  labels:
    application-name: directpv.min.io
    application-type: CSIDriver
    directpv.min.io/created-by: kubectl-directpv
    directpv.min.io/version: v1beta1
  name: ${NAME}
parameters:
  fstype: xfs
$(printf '  %s\n' "${DRIVE_LABELS[@]}")
provisioner: directpv-min-io
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
EOF
}

init "$@"
main "$@"
