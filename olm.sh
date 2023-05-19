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

# Add needed annotations for redhat marketplace
yq -i '.metadata.annotations."marketplace.openshift.io/remote-workflow" |= "https://marketplace.redhat.com/en-us/operators/minio-directpv-operator-rhmp/pricing?utm_source=openshift_console"' bundles/redhat-marketplace/"$RELEASE"/manifests/minio-directpv.clusterserviceversion.yaml
yq -i '.metadata.annotations."marketplace.openshift.io/support-workflow" |= "https://marketplace.redhat.com/en-us/operators/minio-directpv-operator-rhmp/support?utm_source=openshift_console"' bundles/redhat-marketplace/"$RELEASE"/manifests/minio-directpv.clusterserviceversion.yaml
