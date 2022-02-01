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

INSTALL_DIRECT_PV_GITHUB_URL="https://github.com/minio/directpv"
DIRECTDIR=".directpv"
darwin="Darwin"
linux="Linux"
windows="Windows"

# info logs the given argument at info log level.
info() {
    echo "[INFO] " "$@"
}

# warn logs the given argument at warn log level.
warn() {
    echo "[WARN] " "$@" >&2
}

# fatal logs the given argument at fatal log level.
fatal() {
    echo "[ERROR] " "$@" >&2
    if [ -n "${SUFFIX}" ]; then
        echo "[ALT] Please visit 'https://github.com/minio/directpv/releases' directly and download the latest directpv-${SUFFIX}" >&2
    fi
    exit 1
}


# verify_priv verifies if installation has neccessary privileges.
verify_priv() {
    # --- bail if we are not root ---
    if [ "$(id -u)" -ne 0 ]; then
        fatal "You need to be root to perform this install"
    fi
}

function directpv(){
    verify_priv
    info "installing directPV"
    init 
    verify_downloader curl || verify_downloader wget || fatal "can not find curl or wget for downloading files"
    get_latest_version
    
    setup_arch
    setup
    setup_kubectl
    
    info "installing directpv plugin v${INSTALL_DIRECT_PV_VERSION}"
    download_checksums
    download_binary
    verify_binary
    install_binary
    verify_installation
}

# Fetch the operating system
function init(){
    case "$OSTYPE" in
        darwin*)  
            OPERATING_SYSTEM=$darwin            
            KUBECTL_URL="https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/darwin/amd64/kubectl"
            ;;
        linux*)   
            OPERATING_SYSTEM=$linux
            KUBECTL_URL="https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
            ;;
        msys*)    
            OPERATING_SYSTEM=$windows 
            KUBECTL_URL="https://dl.k8s.io/release/v1.23.0/bin/windows/amd64/kubectl.exe";;
        *)        echo "unknown: $OSTYPE" ;;
    esac
}


# verify_downloader verifies existence of
# network downloader executable.
verify_downloader() {
    cmd="$(command -v "${1}")"
    if [ -z "${cmd}" ]; then
        return 1
    fi
    if [ ! -x "${cmd}" ]; then
        return 1
    fi

    # Set verified executable as our downloader program and return success
    DOWNLOADER=${cmd}
    return 0
}

# get the latest version 
function get_latest_version() {
    INSTALL_DIRECT_PV_VERSION=$(curl -s  https://api.github.com/repos/minio/directpv/releases/latest | jq -r ".tag_name" | tr -d v )
}

# install kubectl if not installed
function setup_kubectl() {
    if [ ! "$(kubectl version --client)" ]; then
        case "$OPERATING_SYSTEM" in        
            Linux)
                KUBECTL_HOME="$HOME/.local/bin"
				TMP_KUBECTL_BINARY=$HOME/$DIRECTDIR/kubectl
                download "${TMP_KUBECTL_BINARY}" "${KUBECTL_URL}"
                chmod +x "${TMP_KUBECTL_BINARY}"
				mv "${TMP_KUBECTL_BINARY}" "${KUBECTL_HOME}/kubectl"
        esac
    fi
}


setup_arch() {
    case ${ARCH:=$(uname -m)} in
    amd64)
        ARCH=amd64
        SUFFIX=$(uname -s | tr '[:upper:]' '[:lower:]')_${ARCH}
        ;;
    x86_64)
        ARCH=amd64
        SUFFIX=$(uname -s | tr '[:upper:]' '[:lower:]')_${ARCH}
        ;;
    *)
        fatal "unsupported architecture ${ARCH}"
        ;;
    esac
}

# setup creates a directory
setup() {
    case "$OPERATING_SYSTEM" in 
        Linux)
            DIRECTPV_DIR=$HOME/$DIRECTDIR
            mkdir -p "$DIRECTPV_DIR"
            TMP_CHECKSUMS=${DIRECTPV_DIR}/directpv_${INSTALL_DIRECT_PV_VERSION}_checksums.txt
            TMP_BINARY=${DIRECTPV_DIR}/kubectl-directpv_${INSTALL_DIRECT_PV_VERSION}_${SUFFIX} ;;
    esac
}

# download_checksums downloads hash from github url.
download_checksums() {
    CHECKSUMS_URL=${INSTALL_DIRECT_PV_GITHUB_URL}/releases/download/v${INSTALL_DIRECT_PV_VERSION}/directpv_${INSTALL_DIRECT_PV_VERSION}_checksums.txt
    info "downloading checksums at ${CHECKSUMS_URL}"
    download "${TMP_CHECKSUMS}" "${CHECKSUMS_URL}"
    CHECKSUM_EXPECTED=$(grep "kubectl-directpv_${INSTALL_DIRECT_PV_VERSION}_${SUFFIX}" "${TMP_CHECKSUMS}" | awk '{print $1}')
}

# download_tarball downloads binary from github url.
download_binary() {
    BINARY_URL=${INSTALL_DIRECT_PV_GITHUB_URL}/releases/download/v${INSTALL_DIRECT_PV_VERSION}/kubectl-directpv_${INSTALL_DIRECT_PV_VERSION}_${SUFFIX}
    info "downloading binary at ${BINARY_URL}"
    download "${TMP_BINARY}" "${BINARY_URL}"
}

# verify_binary verifies the downloaded installer checksum.
verify_binary() {
    info "verifying binary"
    CHECKSUM_ACTUAL=$(sha256sum "${TMP_BINARY}" | awk '{print $1}')
    if [ "${CHECKSUM_EXPECTED}" != "${CHECKSUM_ACTUAL}" ]; then
        fatal "download sha256 does not match ${CHECKSUM_EXPECTED}, got ${CHECKSUM_ACTUAL}"
    fi
}

# install the binary
install_binary() {
    INSTALL_LOCATION=/usr/local/bin
    info "install binary to ${INSTALL_LOCATION}"
    chmod +x "${TMP_BINARY}"
    mv "${TMP_BINARY}" ${INSTALL_LOCATION}/kubectl-directpv
}

# download downloads from github url.
download() {
    if [ $# -ne 2 ]; then
        fatal "download needs exactly 2 arguments"
    fi

    case ${DOWNLOADER} in
    *curl)
        if ! curl -o "$1" -fsSL "$2"; then
            fatal "download failed"
        fi
        ;;
    *wget)
        if ! wget -qO "$1" "$2"; then
            fatal "download failed"
        fi
        ;;
    *)
        fatal "downloader executable not supported: '${DOWNLOADER}'"
        ;;
    esac
}

# verify the installation directory and version
function verify_installation(){
    INSTALLED_DIRECTPV=$(which kubectl-directpv)
    EXPECTED_DIRECTPV=${INSTALL_LOCATION}/kubectl-directpv

    if [ "${INSTALLED_DIRECTPV}" != "${EXPECTED_DIRECTPV}" ]; then
        fatal "intalled directpv does not match; expected ${EXPECTED_DIRECTPV}, got ${INSTALLED_DIRECTPV}"
    fi
    INSTALLED_VERSION=$(kubectl-directpv --version | awk '{print $3}')
    if [ "${INSTALLED_VERSION}" != "v${INSTALL_DIRECT_PV_VERSION}" ]; then
        fatal "intalled version does not match; expected v${INSTALL_DIRECT_PV_VERSION}, got ${INSTALLED_VERSION}"
    fi
    info "installation verified"
}

directpv
