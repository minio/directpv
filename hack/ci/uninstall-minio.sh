#!/usr/bin/env bash

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

set -ex

kubectl delete -f hack/ci/minio.yaml
sleep 1m
kubectl delete pvc --all
sleep 1m
./kubectl-direct_csi volumes ls
./kubectl-direct_csi drives ls --all

directcsivolumes=$(./kubectl-direct_csi volumes ls | wc -l)
if [[ $directcsivolumes -gt 1 ]]
then
  echo "Volumes were not cleared upon deletion"
  exit 1
fi

inusedrives=$(./kubectl-direct_csi drives ls | grep -q InUse | wc -l)
if [[ $inusedrives -gt 0 ]]
then
  echo "Drives were not released upon volume deletion"
  exit 1
fi
