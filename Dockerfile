FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

WORKDIR /

COPY directpv /directpv
COPY CREDITS /licenses/CREDITS
COPY LICENSE /licenses/LICENSE

RUN microdnf update --nodocs

COPY centos.repo /etc/yum.repos.d/CentOS.repo

RUN \
    curl -L https://www.centos.org/keys/RPM-GPG-KEY-CentOS-Official-SHA256 -o /etc/pki/rpm-gpg/RPM-GPG-KEY-CentOS-Official-SHA256 && \
    microdnf install xfsprogs -y --nodocs && \
    microdnf clean all && \
    rm -f /etc/yum.repos.d/CentOS.repo

ENTRYPOINT ["/directpv"]
