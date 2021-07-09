FROM registry.access.redhat.com/ubi8/ubi-minimal:8.4

WORKDIR /

COPY direct-csi /direct-csi
COPY CREDITS /licenses/CREDITS
COPY LICENSE /licenses/LICENSE

RUN microdnf update --nodocs

COPY centos.repo /etc/yum.repos.d/CentOS.repo

RUN \
    curl -L https://www.centos.org/keys/RPM-GPG-KEY-CentOS-Official -o /etc/pki/rpm-gpg/RPM-GPG-KEY-CentOS-Official && \
    microdnf install xfsprogs --nodocs && \
    microdnf clean all && \
    rm -f /etc/yum.repos.d/CentOS.repo

ENTRYPOINT ["/direct-csi"]
