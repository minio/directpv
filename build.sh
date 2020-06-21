#!/bin/bash

set -e

export GIT_COMMIT=$(git describe --always --dirty)
export VERSION=v0.0.1-$GIT_COMMIT

echo "building jbod-csi-driver" $VERSION

go build -a -ldflags "-X github.com/minio/jbod-csi-driver/cmd.Version=$VERSION"
