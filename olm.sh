#!/bin/bash

operator-sdk generate bundle \
    --package minio-directpv \
    --version "$RELEASE" \
    --deploy-dir resources/base/"$RELEASE" \
    --manifests \
    --metadata \
    --output-dir bundles/redhat-marketplace/"$RELEASE" \
    --channels stable \
    --overwrite

# Annotations to specify OCP versions compatibility.
yq -i '.annotations."com.redhat.openshift.versions" |= "v4.8-v4.12"' bundles/redhat-marketplace/"$RELEASE"/metadata/annotations.yaml
