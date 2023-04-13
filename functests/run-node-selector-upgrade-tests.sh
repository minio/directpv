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

ME=$(basename "$0"); export ME

if [ "$#" -ne 1 ]; then
    echo "usage: $ME <NODE-SELECTOR>"
    exit 255
fi

cd "$(dirname "$0")" || exit 255
DIRECTPV_DIR="$(cd .. && echo "$PWD")"
export DIRECTPV_DIR

./execute.sh node-selector-upgrade-tests.sh "$1"
