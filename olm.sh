#!/bin/bash

operator-sdk generate bundle \
    --package minio-directpv-operator-rhmp \
    --version "$RELEASE" \
    --deploy-dir resources/base/"$RELEASE" \
    --manifests \
    --metadata \
    --output-dir bundles/redhat-marketplace/"$RELEASE" \
    --channels stable \
    --overwrite
