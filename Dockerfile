FROM registry.access.redhat.com/ubi8/ubi-minimal:8.10

WORKDIR /

COPY directpv /directpv
COPY CREDITS /licenses/CREDITS
COPY LICENSE /licenses/LICENSE

RUN microdnf update --nodocs

COPY AlmaLinux.repo /AlmaLinux.repo

RUN \
    curl -L https://repo.almalinux.org/almalinux/8/BaseOS/x86_64/os/RPM-GPG-KEY-AlmaLinux -o /etc/pki/rpm-gpg/RPM-GPG-KEY-AlmaLinux && \
    microdnf install dnf --nodocs && \
    mv /AlmaLinux.repo /etc/yum.repos.d/AlmaLinux.repo && \
    dnf --quiet --assumeyes --nodocs install xfsprogs && \
    dnf --quiet --assumeyes clean all && \
    rpm -e --nodeps dnf dnf-data gdbm gdbm-libs ima-evm-utils libcomps libevent libreport-filesystem platform-python platform-python-pip platform-python-setuptools python3-dnf python3-gpg python3-hawkey python3-libcomps python3-libdnf python3-libs python3-pip-wheel python3-rpm python3-setuptools-wheel python3-unbound rpm-build-libs tpm2-tss unbound-libs && \
    microdnf clean all && \
    rm -f /etc/yum.repos.d/AlmaLinux.repo

ENTRYPOINT ["/directpv"]
