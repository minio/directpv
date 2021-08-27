#!/usr/bin/env bash
#
# This file is part of MinIO Direct CSI
# Copyright (c) 2021 MinIO, Inc.
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
# This script is indented to run in Github workflow.
# DO NOT USE in other systems.
#

ME=$(basename "$0")
export ME

SCRIPT_DIR=$(dirname "$0")
export SCRIPT_DIR

if [[ $# -ne 1 ]]; then
    echo "error: build version must be provided"
    echo "usage: $ME <BUILD-VERSION>"
    exit 255
fi

BUILD_VERSION="$1"
export BUILD_VERSION

function execute() {
    # Open /var/tmp/check.sh.xtrace for fd 3
    exec 3>/var/tmp/check.sh.xtrace

    # Set PS4 to [filename:lineno]: for tracing
    # Save all traces to fd 3
    # Run passed command with enabled errexit/xtrace
    PS4='+ [${BASH_SOURCE}:${FUNCNAME[0]:+${FUNCNAME[0]}():}${LINENO}]: ' BASH_XTRACEFD=3 bash -ex "$@"
    exit_status=$?

    # Close fd 3
    exec 3>&-

    if [ "$exit_status" -ne 0 ]; then
        echo
        echo "xtrace:"
        tail /var/tmp/check.sh.xtrace | tac
    fi

    return "$exit_status"
}

execute "${SCRIPT_DIR}/tests.sh"
