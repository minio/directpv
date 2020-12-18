#!/bin/bash

binary=$(basename "$(dirname "${1}")")
minisign -s "/media/${USER}/minio/minisign.key" \
         -x "${binary}.minisig" \
         -Sm "${1}" < "/media/${USER}/minio/minisign-passphrase"

cp -f "${binary}.minisig" "${1}.minisig"
cp -f LICENSE "$(dirname "${1}")"
zip -r -j "${binary}.zip" "$(dirname "${1}")"

if [ -f direct-csi_linux_amd64.minisig ]; then
    mv direct-csi_linux_amd64.minisig direct-csi.minisig
fi

if [ -f kubectl-direct_csi_linux_amd64.minisig ]; then
    mv kubectl-direct_csi_linux_amd64.minisig kubectl-direct_csi.minisig
fi
