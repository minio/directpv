#!/bin/bash

set -e

CSI_VERSION=$(git describe --tags --always --dirty)
export CSI_VERSION

echo "running license checks"
find . | grep .go$ | xargs addlicense -check

echo "building direct-csi $CSI_VERSION"
CGO_ENABLED=0 go build -tags 'kqueue osusergo netgo static_build' -ldflags="-s -w -X github.com/minio/direct-csi/cli.VERSION=$CSI_VERSION"

echo "generating storagetopology CRD"
controller-gen paths=./... crd:trivialVersions=true output:crd:artifacts:config=resources
