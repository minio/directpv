FROM registry.access.redhat.com/ubi8/ubi-minimal:8.10

WORKDIR /

COPY directpv /directpv
COPY CREDITS /licenses/CREDITS
COPY LICENSE /licenses/LICENSE

RUN microdnf update --nodocs

COPY AlmaLinux.repo /etc/yum.repos.d/AlmaLinux.repo

RUN \
    curl -L https://repo.almalinux.org/almalinux/8/BaseOS/x86_64/os/RPM-GPG-KEY-AlmaLinux -o /etc/pki/rpm-gpg/RPM-GPG-KEY-AlmaLinux && \
    microdnf install xfsprogs --nodocs && \
    microdnf clean all && \
    rm -f /etc/yum.repos.d/AlmaLinux.repo

ENTRYPOINT ["/directpv"]
