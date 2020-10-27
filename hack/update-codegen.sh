#!/bin/bash
# This file is part of MinIO Direct CSI
# Copyright (c) 2020 MinIO, Inc.
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

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/..

GO111MODULE=off go get -d k8s.io/code-generator/...

REPOSITORY=github.com/minio/direct-csi
$(go env GOPATH)/src/k8s.io/code-generator/generate-groups.sh all \
                $REPOSITORY/pkg/client $REPOSITORY/pkg/apis \
                "direct.csi.min.io:v1alpha1" \
                --go-header-file $SCRIPT_ROOT/hack/boilerplate.go.txt
