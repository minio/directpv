#!/usr/bin/env bash
#
# This file is part of MinIO Direct PV
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

ME=$(basename "$0")
export ME

SCRIPT_DIR=$(dirname "$0")
export SCRIPT_DIR

if [[ $# -ne 1 ]]; then
    echo "error: DRIVE must be provided"
    echo "usage: $ME <DRIVE>"
    exit 255
fi

"${SCRIPT_DIR}/execute.sh" "${SCRIPT_DIR}/check-drive-exist.sh" "$@"
