---
title: Security
---

Security
---------

**Important**: Report security issues to security@min.io. Please do not report security issues here.

## Supported Versions

We always provide security updates for the [latest release](https://github.com/minio/minio/releases/latest).
Whenever there is a security update you just need to upgrade to the latest version.

## Reporting a Vulnerability

All security bugs in [minio/direct-csi](https://github,com/minio/direct-csi) (or other minio/* repositories)
should be reported by email to security@min.io. Your email will be acknowledged within 48 hours,
and you'll receive a more detailed response to your email within 72 hours indicating the next steps
in handling your report.

Please, provide a detailed explanation of the issue. In particular, outline the type of the security
issue (DoS, authentication bypass, information disclose, ...) and the assumptions you're making (e.g. do
you need access credentials for a successful exploit).

If you have not received a reply to your email within 48 hours or you have not heard from the security team
for the past five days please contact the security team directly:
   - Primary security coordinators: sid@min.io
   - Secondary coordinator: harsha@min.io
   - If you receive no response: dev@min.io

### Disclosure Process

MinIO uses the following disclosure process:

1. Once the security report is received one member of the security team tries to verify and reproduce
   the issue and determines the impact it has.
2. A member of the security team will respond and either confirm or reject the security report.
   If the report is rejected the response explains why.
3. Code is audited to find any potential similar problems.
4. Fixes are prepared for the latest release.
5. On the date that the fixes are applied a security advisory will be published on https://blog.min.io.
   Please inform us in your report email whether MinIO should mention your contribution w.r.t. fixing
   the security issue. By default MinIO will **not** publish this information to protect your privacy.

This process can take some time, especially when coordination is required with maintainers of other projects.
Every effort will be made to handle the bug in as timely a manner as possible, however it's important that we
follow the process described above to ensure that disclosures are handled consistently.

## Vulnerability Management Policy
-------------------------------------

This document formally describes the process of addressing and managing a
reported vulnerability that has been found in the MinIO server code base,
any directly connected ecosystem component or a direct / indirect dependency
of the code base.

### Scope

The vulnerability management policy described in this document covers the
process of investigating, assessing and resolving a vulnerability report
opened by a MinIO employee or an external third party.

Therefore, it lists pre-conditions and actions that should be performed to
resolve and fix a reported vulnerability.

### Vulnerability Management Process

The vulnerability management process requires that the vulnerability report
contains the following information:

 - The project / component that contains the reported vulnerability.
 - A description of the vulnerability. In particular, the type of the
   reported vulnerability and how it might be exploited. Alternatively,
   a well-established vulnerability identifier, e.g. CVE number, can be
   used instead.

Based on the description mentioned above, a MinIO engineer or security team
member investigates:

 - Whether the reported vulnerability exists.
 - The conditions that are required such that the vulnerability can be exploited.
 - The steps required to fix the vulnerability.

In general, if the vulnerability exists in one of the MinIO code bases
itself - not in a code dependency - then MinIO will, if possible, fix
the vulnerability or implement reasonable countermeasures such that the
vulnerability cannot be exploited anymore.
