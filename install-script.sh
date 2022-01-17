#!/usr/bin/env bash

# This file is part of MinIO DirectPV
# Copyright (c) 2022 MinIO, Inc.
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

echo "Installing direct PV"

if [[ $kv -eq 0 ]]; then
    echo "kubectl found"
else
    echo "Please install kubectl"
fi
# Install kubectl directpv plugin
if [[ $krew -eq 0 ]]; then
    echo "Installed kubectl directpv plugin"
else
    echo "Please install krew"
fi

# Use the plugin to install directpv in your kubernetes cluster
if [[ $install -eq 0 ]]; then
    echo "DirectPv installed successfully!"
else
    echo "Please install krew"
fi

kv="kubectl version"
krew="kubectl krew install directpv"
install="kubectl directpv install"
