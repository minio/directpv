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

if [[ $kv = 0 ]]; then
    echo "kubectl found"
else
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    chmod +x kubectl
    mkdir -p ~/.local/bin/kubectl
    mv ./kubectl ~/.local/bin/kubectl
    echo "kubectl installed"
fi
# Install krew if not found
if [[ $krew-version = 0 ]]; then
    echo "krew found"
else
    set -x; cd "$(mktemp -d)" &&
    curl -fsSLO "https://github.com/kubernetes-sigs/krew/releases/latest/download/krew.tar.gz" &&
    tar zxvf krew.tar.gz &&
    KREW=./krew-"$(uname | tr '[:upper:]' '[:lower:]')_$(uname -m | sed -e 's/x86_64/amd64/' -e 's/arm.*$/arm/')" &&
    "$KREW" install krew
    export PATH="${KREW_ROOT:-$HOME/.krew}/bin:$PATH"
    echo "krew installed"
fi
# Install kubectl directpv plugin
if [[ $krew = 0 ]]; then
    echo "Installed kubectl directpv plugin"
else
    echo "Problem in installing krew DirectPV plugin"
fi

# Use the plugin to install directpv in your kubernetes cluster
if [[ $install = 0 ]]; then
    echo "DirectPV installed successfully!"
else
    echo "Please install krew"
fi

kv="kubectl version"
krew-version="kubectl krew version"
krew="kubectl krew install directpv"
install="kubectl directpv install"
