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

# Use SHA Digest for CSI Provisioner Image
csiProvisionerImage="quay.io/minio/csi-provisioner:v3.4.0"
csiProvisionerImageDigest=$(podman pull "$csiProvisionerImage" | awk '/Digest/ { print $2}')
csiProvisionerImageDigest="quay.io/minio/csi-provisioner@${csiProvisionerImageDigest}"
yq -i ".spec.install.spec.deployments[0].spec.template.spec.containers[0].image |= (\"${csiProvisionerImageDigest}\")" bundles/redhat-marketplace/"$RELEASE"/manifests/"$package".clusterserviceversion.yaml

# Use SHA Digest for CSI Resizer Image
csiResizerImage="quay.io/minio/csi-resizer:v1.7.0"
csiResizerImageDigest=$(podman pull "$csiResizerImage" | awk '/Digest/ { print $2}')
csiResizerImageDigest="quay.io/minio/csi-resizer@${csiResizerImageDigest}"
yq -i ".spec.install.spec.deployments[0].spec.template.spec.containers[1].image |= (\"${csiResizerImageDigest}\")" bundles/redhat-marketplace/"$RELEASE"/manifests/"$package".clusterserviceversion.yaml

# Use SHA Digest for DirectPV Image
directPVImage="quay.io/minio/directpv:v$RELEASE"
directPVImageDigest=$(podman pull "$directPVImage" | awk '/Digest/ { print $2}')
directPVImageDigest="quay.io/minio/directpv@${directPVImageDigest}"
yq -i ".spec.install.spec.deployments[0].spec.template.spec.containers[2].image |= (\"${directPVImageDigest}\")" bundles/redhat-marketplace/"$RELEASE"/manifests/"$package".clusterserviceversion.yaml
