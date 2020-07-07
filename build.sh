#!/bin/bash

set -e

CSI_VERSION=$(git describe --tags --always --dirty)
export CSI_VERSION

echo "running license checks"
find . | grep .go$ | xargs addlicense -check

echo "building jbod-csi-driver $CSI_VERSION"
CGO_ENABLED=0 go build -tags 'osusergo netgo static_build' -ldflags="-X github.com/minio/jbod-csi-driver/cmd.Version=$CSI_VERSION -extldflags=-static"

