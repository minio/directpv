#!/bin/bash

package=minio-directpv-operator-rhmp

operator-sdk generate bundle \
    --package $package \
    --version "$RELEASE" \
    --deploy-dir resources/base/"$RELEASE" \
    --manifests \
    --metadata \
    --output-dir bundles/redhat-marketplace/"$RELEASE" \
    --channels stable \
    --overwrite

# Annotations to specify OCP versions compatibility.
yq -i '.annotations."com.redhat.openshift.versions" |= "v4.8-v4.12"' bundles/redhat-marketplace/"$RELEASE"/metadata/annotations.yaml

# Add needed annotations for redhat marketplace
yq -i '.metadata.annotations."marketplace.openshift.io/remote-workflow" |= "https://marketplace.redhat.com/en-us/operators/minio-directpv-operator-rhmp/pricing?utm_source=openshift_console"' bundles/redhat-marketplace/"$RELEASE"/manifests/$package.clusterserviceversion.yaml
yq -i '.metadata.annotations."marketplace.openshift.io/support-workflow" |= "https://marketplace.redhat.com/en-us/operators/minio-directpv-operator-rhmp/support?utm_source=openshift_console"' bundles/redhat-marketplace/"$RELEASE"/manifests/$package.clusterserviceversion.yaml
