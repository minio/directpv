#!/bin/sh

set -e

if [ "${DEBUG}" = 1 ]; then
    set -x
fi

INSTALL_DIRECT_CSI_VERSION=1.0.0
INSTALL_DIRECT_CSI_GITHUB_URL="https://github.com/minio/direct-csi"

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
        echo "[ALT] Please visit 'https://github.com/minio/direct-csi/releases' directly and download the latest direct-csi_${SUFFIX}" >&2
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

# setup_tmp creates a temporary directory
# and cleans up when done.
setup_tmp() {
    TMP_DIR=$(mktemp -d -t directcsi-install.XXXXXXXXXX)
    TMP_CHECKSUMS=${TMP_DIR}/direct-csi_${INSTALL_DIRECT_CSI_VERSION}_checksums.txt
    TMP_BINARY=${TMP_DIR}/kubectl-direct_csi_${INSTALL_DIRECT_CSI_VERSION}_${SUFFIX}
    cleanup() {
        code=$?
        set +e
        trap - EXIT
        rm -rf "${TMP_DIR}"
        exit $code
    }
    trap cleanup INT EXIT
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

# download_checksums downloads hash from github url.
download_checksums() {
    CHECKSUMS_URL=${INSTALL_DIRECT_CSI_GITHUB_URL}/releases/download/v${INSTALL_DIRECT_CSI_VERSION}/direct-csi_${INSTALL_DIRECT_CSI_VERSION}_checksums.txt
    info "downloading checksums at ${CHECKSUMS_URL}"
    download "${TMP_CHECKSUMS}" "${CHECKSUMS_URL}"
    CHECKSUM_EXPECTED=$(grep "kubectl-direct_csi" "${TMP_CHECKSUMS}" | awk '{print $1}')
}

# download_tarball downloads binary from github url.
download_binary() {
    BINARY_URL=${INSTALL_DIRECT_CSI_GITHUB_URL}/releases/download/v${INSTALL_DIRECT_CSI_VERSION}/kubectl-direct_csi_${INSTALL_DIRECT_CSI_VERSION}_${SUFFIX}
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
install_binary() {
    INSTALL_LOCATION=/usr/local/bin
    info "install binary to ${INSTALL_LOCATION}"
    chmod +x "${TMP_BINARY}"
    mv "${TMP_BINARY}" ${INSTALL_LOCATION}/kubectl-direct_csi
}

do_install() {
    verify_priv
    setup_arch
    verify_downloader curl || verify_downloader wget || fatal "can not find curl or wget for downloading files"

    setup_tmp
    info "installing direct-csi plugin v${INSTALL_DIRECT_CSI_VERSION}"

    download_checksums
    download_binary
    verify_binary
    install_binary
    info "installation successful. Run 'kubectl direct-csi --version' to verify"
}

do_install
exit 0
