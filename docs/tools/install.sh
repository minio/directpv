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
# This script installs or upgrades DirectPV declaratively.
#

set -e

declare -a plugin_cmd directpv_args
apply_flag=0

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

function is_migrated_volume_found() {
    if ! kubectl get --ignore-not-found=true crd directpvvolumes.directpv.min.io --no-headers -o NAME | grep -q .; then
        return 1
    fi

    exists=$(kubectl get directpvvolumes.directpv.min.io --selector=directpv.min.io/migrated=true --ignore-not-found=true -o go-template='{{range .items}}{{1}}{{break}}{{end}}')
    [ -n "${exists}" ]
}

declare -A fsuuidDriveNameMap driveNameFSUUIDMap

function check_directcsi_consistency() {
    ecount=0
    # shellcheck disable=SC2207
    names=( $(kubectl get directcsidrives -o go-template='{{range .items}}{{if or (eq .status.driveStatus "Ready") (eq .status.driveStatus "InUse") }}{{.metadata.name}} {{end}}{{end}}') )
    for name in "${names[@]}"; do
        fsuuid=$(kubectl get directcsidrives "${name}" -o go-template='{{.status.filesystemUUID}}')

        if [ -z "${fsuuid}" ]; then
            echo "[ERROR] drive ${name}: empty fsuuid"
            (( ++ecount ))
            continue
        fi

        if ! echo "${fsuuid}" | grep -q -E "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"; then
            echo "[ERROR] drive ${name}: invalid fsuuid ${fsuuid}"
            (( ++ecount ))
            continue
        fi

        if [ -n "${fsuuidDriveNameMap[$fsuuid]}" ]; then
            echo "[ERROR] drive ${name}: duplicate FSUUID ${fsuuid} found"
            (( ++ecount ))
            continue
        fi

        fsuuidDriveNameMap[$fsuuid]=$name
        driveNameFSUUIDMap[$name]=$fsuuid
    done

    if [ "${ecount}" -gt 0 ]; then
        exit 1
    fi

    for nameDrive in $(kubectl get directcsivolumes -o go-template='{{range .items}}{{.metadata.name}}={{.status.drive}} {{end}}'); do
        name=$(echo "${nameDrive}" | cut -d= -f1)
        drive=$(echo "${nameDrive}" | cut -d= -f2)

        if [ -z "${drive}" ]; then
            echo "[ERROR] volume ${name}: empty drive name"
            (( ++ecount ))
            continue
        fi

        if [ -z "${driveNameFSUUIDMap[$drive]}" ]; then
            echo "[ERROR] volume ${name}: FSUUID not found for drive ${drive}"
            (( ++ecount ))
            continue
        fi
    done

    if [ "${ecount}" -gt 0 ]; then
        exit 1
    fi
}

function init() {
    help_flag=0
    for arg in "$@"; do
        if [ "${arg}" != "apply" ]; then
            directpv_args+=("${arg}")
            if [ "${arg}" == "--help" ]; then
                help_flag=1
            fi
        else
            apply_flag=1
        fi
    done

    if [ "${help_flag}" -eq 1 ]; then
        cat << EOF
USAGE:
  install.sh [INSTALL-FLAGS] [apply]

ARGUMENTS:
  INSTALL-FLAGS    Optional DirectPV installation flags.
  apply            If present, apply manifest file after generation.

EXAMPLES:
  # Generate DirectPV manifests.
  $ install.sh

  # Generate DirectPV manifests with private registry.
  $ install.sh --registry private-registry.io --org org-name

  # Generate DirectPV manifests with node-selector.
  $ install.sh --node-selector node-label-key=node-label-value

  # Generate and apply DirectPV manifests.
  $ install.sh apply
EOF
        exit 255
    fi

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

function wait_for_crd() {
    while ! kubectl get --ignore-not-found=true crd directpvdrives.directpv.min.io --no-headers -o NAME | grep -q .; do
        echo "  ...waiting for drive CRD to be created"
        sleep 1
    done

    while ! kubectl get --ignore-not-found=true crd directpvvolumes.directpv.min.io --no-headers -o NAME | grep -q .; do
        echo "  ...waiting for volume CRD to be created"
        sleep 1
    done
}

function main() {
    echo "* Probe legacy DirectCSI"
    legacy=0
    migrate=0
    if is_legacy_drive_found || is_legacy_volume_found; then
        legacy=1
        migrate=1
        check_directcsi_consistency
    elif is_migrated_volume_found; then
        legacy=1
    fi

    echo "* Generate DirectPV manifests to be saved at directpv.yaml"
    if [[ "${legacy}" -eq 0 ]]; then
        run_plugin_cmd install --declarative "${directpv_args[@]}" > directpv.yaml
    else
        run_plugin_cmd install --legacy --declarative "${directpv_args[@]}" > directpv.yaml
    fi

    if [ "${apply_flag}" -eq 0 ]; then
        echo "* To install/upgrade DirectPV, run 'kubectl apply -f directpv.yaml'"
        if [[ "${migrate}" -eq 1 ]]; then
            echo "* After install/upgrade, run 'migrate.sh' to complete migration"
        fi

        return 0
    fi

    echo "* Apply manifests from directpv.yaml"
    if ! kubectl apply -f directpv.yaml; then
        return 1
    fi

    if [[ "${migrate}" -eq 0 ]]; then
        return 0
    fi

    echo "* Migrate legacy drives and volumes"
    wait_for_crd
    if ! run_plugin_cmd migrate; then
        return 1
    fi
    kubectl -n directpv delete pods --all
}

init "$@"
main "$@"
