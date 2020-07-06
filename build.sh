#!/bin/bash

set -e

export GIT_COMMIT=$(git describe --always --dirty)
export VERSION=v0.1.1-$GIT_COMMIT

echo "building jbod-csi-driver" $VERSION

CGO_ENABLED=0 go build -tags 'osusergo netgo static_build' -ldflags="-X github.com/minio/jbod-csi-driver/cmd.Version=$VERSION -extldflags=-static"
