#!/usr/bin/env bash
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

ME=$(basename "$0"); export ME
cd "$(dirname "$0")" || exit 255

export GITHUB_PROJECT_NAME=external-provisioner
export PROJECT_NAME=csi-provisioner
export PROJECT_DESCRIPTION="CSI External Provisioner"

if [ "$#" -ne 1 ]; then
    cat <<EOF
USAGE:
  ${ME} <VERSION>
EXAMPLES:
  # Release ${PROJECT_NAME} v5.0.2
  $ ${ME} v5.0.2
EOF
    exit 255
fi

# shellcheck source=/dev/null
source release.sh
release "$1"
