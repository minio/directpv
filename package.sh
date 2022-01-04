#!/bin/bash

binary=$(basename "$(dirname "${1}")")
if [[ "${binary}" =~ kubectl-direct_csi* ]]; then
    cp -f LICENSE "$(dirname "${1}")"
    cp -f CREDITS "$(dirname "${1}")"
    cp -f README.md  "$(dirname "${1}")"
    rm -f "${binary}.zip"
    zip -q -r -j "${binary}.zip" "$(dirname "${1}")"
elif [[ "${binary}" =~ kubectl-directpv* ]]; then
    cp -f LICENSE "$(dirname "${1}")"
    cp -f CREDITS "$(dirname "${1}")"
    cp -f README.md  "$(dirname "${1}")"
    rm -f "${binary}.zip"
    zip -q -r -j "${binary}.zip" "$(dirname "${1}")"
fi
