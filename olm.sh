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
csiProvisionerImage="registry.redhat.io/openshift4/ose-csi-external-provisioner:latest"
csiProvisionerImageDigest=$(podman pull "$csiProvisionerImage" | awk '/Digest/ { print $2}')
csiProvisionerImageDigest="registry.redhat.io/openshift4/ose-csi-external-provisioner@${csiProvisionerImageDigest}"
yq -i ".spec.install.spec.deployments[0].spec.template.spec.containers[0].image |= (\"${csiProvisionerImageDigest}\")" bundles/redhat-marketplace/"$RELEASE"/manifests/"$package".clusterserviceversion.yaml

# Use SHA Digest for CSI Resizer Image
csiResizerImage="registry.redhat.io/openshift4/ose-csi-external-resizer-rhel8:latest"
csiResizerImageDigest=$(podman pull "$csiResizerImage" | awk '/Digest/ { print $2}')
csiResizerImageDigest="registry.redhat.io/openshift4/ose-csi-external-resizer-rhel8@${csiResizerImageDigest}"
yq -i ".spec.install.spec.deployments[0].spec.template.spec.containers[1].image |= (\"${csiResizerImageDigest}\")" bundles/redhat-marketplace/"$RELEASE"/manifests/"$package".clusterserviceversion.yaml

# Use SHA Digest for DirectPV Image
directPVImage="quay.io/minio/directpv:v$RELEASE"
directPVImageDigest=$(podman pull "$directPVImage" | awk '/Digest/ { print $2}')
directPVImageDigest="quay.io/minio/directpv@${directPVImageDigest}"
yq -i ".spec.install.spec.deployments[0].spec.template.spec.containers[2].image |= (\"${directPVImageDigest}\")" bundles/redhat-marketplace/"$RELEASE"/manifests/"$package".clusterserviceversion.yaml

# To add icon right after bundle creation
icon_base64data=iVBORw0KGgoAAAANSUhEUgAAAKcAAACnCAYAAAB0FkzsAAAACXBIWXMAABcRAAAXEQHKJvM/AAAIj0lEQVR4nO2dT6hVVRSHjykI/gMDU0swfKAi2KgGOkv6M1RpqI9qZBYo9EAHSaIopGCQA8tJDXzNgnRcGm+SgwLDIFR4omBmCQrqE4Tkxu/6Tlyv7569zzn73Lvu3t83VO+5HN/31t5r7bX3ntVqtVoZgD0mnuOHAlZBTjALcoJZkBPMgpxgFuQEsyAnmAU5wSzICWZBTjALcoJZkBPMgpxgFuQEsyAnmAU5wSzICWZBTjDLHH40Yfn3/lR299zP2Z2z57PH9x889exFr72SLd60MZu/dtXwv2gfYA9RICTl9SNfZbfP/Oh84Lw1q7KX9+5oywo9mUDOANw5dz6b/ORY9vjBVKmHLX59QzZyeCybs3C+0TcbKMhZl9tnfsgm931e+SmKouu+OYqgz8Luyzrc++ViLTHFw8tXsz/e39OeFsDTIGcNJvcdC/IcCXpl14EBvYVdkLMiGs4f3fwn2PPu/fp79tep031+C9sgZ0V8RJr74gvZks1vZIteXe/1JTdOjGePbv49kPexCHXOCkggDcVFrNi5LVvx4fb//4U+c3nXwcLPKdtX1q8ECYiclXj0Z3F0U4moU8ysHUWXtqVTdl6EhneVpgA5KzF1qThqLh/dMuOfq1zkI6iiJ9k7claie1myDLmgmo/2QsO75p+pg5wVcC07upIaCbr6i/3Z7AW9C++3xk+366gpg5wVmL1wQeGHrn120jn0q/lDEbRI0GtHTvbpjWyCnBWQWK5hWas+rgjqElSZfcq1T+SsyJLNbxZ+UIKqdORKbFyCau6ZanKEnBVZNrq1cEjOSqyb54LORF77TBHkrIiSGrW7uSgj6Mihj2f8u7s/nU8yOULOGjy/aUO2bPvMNc1OfAXVVKGXoKGaTIYJ5KxJu6PdY+28rqBqMkmt9omcAVh9fL9z1Scr0RrXS1Bl7ik1hiBnAHyXJbPptXOfIVqCdk8ZUkuOkDMQZQTVJjgfQTVlUMtdJyk1hiBnQJoQdOTQ2DOCapdnCrVP5AxMPwRVcnTr1PeG3roZkLMBfDqPcqoKeuPLb6NPjpCzIXw6j3IkqE+ThwTtjMixJ0fI2SA+nUc5apHTpjkXnVOG2JMj5GyYMoJqD7xL0O45bczJEXL2gSYFjXnlCDn7RJOCakrgam4eRpCzj5QV1DWfzAXV8zS8xwZy9pmi3s1ulI27ImIuaIzzTk6ZGxC+p9OpVrr+uxMpnkLHKXODoqh3sxMlPKke8oWcA8RXUNUzfWqgsYGcA8ZX0BQ3uiFnn9A6uNbQZ6pJStDuzqNuNLzfPp1W9ETOhlG0k5AX3n6v8DIDrZu7tnvcGo+/E6kT5GwQzRMvvPVuu4PIB9duTkXPlE6gQ84G0BCuzWwqFZW5YUPHJOpczyJ0x1EqIGdgtAnt4jsftTPsKizZUnySSEr715EzEHm0vH70ZOn7iDpR9NThs73Q0J7KDkzkDIDmgXWiZTfOIxYdJyvHAnLWRB3sV3YfrBUtu3HJmcrQzoUFFVGJSMO46+KCKnBx6xOQswLqFJKYIaMlPAtylkS1S51cjJjNg5wlqHsJK5QDOT3REqTvSk9duOblCcjpgRo2fC75F9oyUXfIf3hpsvDv5760tNbzhwVKSQ7KiKnGDZ/Tjl241s9VqE8B5CygjJg6rjDUpf6u9XNXHTQWGNZ7oDVyXzHVLOy6XcMXFdiLrsr2vYE4BoicM6CsXGvkPoQUM5tOvIpYvGljsO+yDpGzC833fMpFSnw0jIdczdEvhWt93tW1FBNEzg608uNzclsTYqrTSMX9IrSVI6Utwsg5jWqLV3YfcJaBmhBT363b3lzf3X2He+wg5zTaG16UiOSsOf5pcDF9GkgUNVMpIeUg53QS4tOLqeQnZBlHmbn2GLnEVLReufeDYN87LCSfEEkQn2XJlXt2BMvKNb/UL4R3qerwWIrH0aQtZz7Xc6Ehdfmo+xpBH5SRl1mj13frGsMUSXpYV2buSkJ0/qX2lIfCZ16bo71EIb972EhWTtUzdRtvEXlmPghCrdMPM0kO6xrOfeqZyswHMdfTUJ5yxMxJUk4lI86a4s5tpTNzSe9zZUsvFKlVyww1vx12kpNT2bnOUC9C88wyBW9JqRvV1CxStZczH8ZTq2UWkZycrsYKRS8N5z6EkFInF7cP8UqkDa4MScnp01ihIdUneklIn+lBLySlonPIjqbYSEpOV9T0Gc7bdcoT46VKQp0gpT/JyCmpXELpfvOiz9eRMufJQbGI6UMycvq0o80071MCpQy8iZM9oJgk5FTUK5ob5iWcTtpr7p4NIdAMScjpmmt2JkFIaYfo5XTNNRU1l41urS2lniPJ560daZ86B/WJXk6VfIpQ47AajetKKcG11JnSycNNE7Wc2hPkSmTqDN9KotQEnGKvZT+IWs6mrkaRlEqgWGpslmjl1NLinbNhr0VByv4SrZw60iXUGZpIORiilTNE1ETKwRKlnBrSXV3uRSClDaKUs+otZ0hpiyjlLDukI6VN4oycnkM6UtomOjl9btVFyuEgOjmLlg+RcrhIQk6kHE6iklMlpM61dKQcbqKSM78iRdts1ZDBHZLDTXTD+rqvj7DNNhKikhMp44LDY8EsyAlmQU4wC3KCWZATzIKcYBbkBLMgJ5gFOcEsyAlmQU4wC3KCWZATzIKcYBbkBLMgJ5gFOcEsyAlmQU4wC3KCWZATzIKcYBbkBLMgJ5gFOcEsyAlmQU4wC3KCWZATzIKcgdFJdzq0FuqDnA0wcmgMQQOAnA2BoPVBzgZB0HogZ8MgaHWQsw8gaDWivdLaGhIUyjGr1Wq1+D/rH1OXrnIFjR8TyAlWmWDOCWZBTjALcoJZkBPMgpxgFuQEsyAnmAU5wSzICWZBTjALcoJZkBPMgpxgFuQEsyAnmAU5wSzICWbRHqIJfjxgjiz77T8hbd197bqGkwAAAABJRU5ErkJggg==
icon_mediatype=image/png
yq -i ".spec.icon[0].base64data |= (\"${icon_base64data}\")" bundles/redhat-marketplace/"$RELEASE"/manifests/"$package".clusterserviceversion.yaml
yq -i ".spec.icon[0].mediatype |= (\"${icon_mediatype}\")" bundles/redhat-marketplace/"$RELEASE"/manifests/"$package".clusterserviceversion.yaml
