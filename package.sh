#!/bin/bash

## Goreleaser helper script.

if [ "$#" -ne 1 ]; then
    echo "usage: package.sh <kubectl-directpv-binary-path>"
    exit 255
fi

dir="$(dirname "$1")"
binary=$(basename "${dir}")
cp -f LICENSE CREDITS README.md "${dir}"
rm -f "${binary}.zip"
zip -r -j "${binary}.zip" "${dir}"
