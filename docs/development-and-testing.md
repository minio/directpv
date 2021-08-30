## Development and Testing
1. Have `quay.io` account, running docker and kubernetes. You could use any registry account and replace `quay.io` with such registry name in below steps.
2. Docker login to your `quay.io` account.
```bash
$ docker login --username <QUAY_USERNAME> https://quay.io
```
3. Create `csi-provisioner`, `livenessprobe` and `csi-node-driver-registrar` repositories by pull/tag/push respective images to your `quay.io` account.
```bash
$ docker pull quay.io/minio/csi-provisioner@sha256:3b465cbcadf7d437fc70c3b6aa2c93603a7eef0a3f5f1e861d91f303e4aabdee && \
docker tag quay.io/minio/csi-provisioner@sha256:3b465cbcadf7d437fc70c3b6aa2c93603a7eef0a3f5f1e861d91f303e4aabdee quay.io/<QUAY_USERNAME>/csi-provisioner && \
docker push quay.io/<QUAY_USERNAME>/csi-provisioner

$ docker pull quay.io/minio/livenessprobe@sha256:072e29e350ed7e870e119cbba37324348e1d00f0ba06d4ea288413466d1aa8e8 && \
docker tag quay.io/minio/livenessprobe@sha256:072e29e350ed7e870e119cbba37324348e1d00f0ba06d4ea288413466d1aa8e8 quay.io/<QUAY_USERNAME>/livenessprobe && \
docker push quay.io/<QUAY_USERNAME>/livenessprobe

$ docker pull quay.io/minio/csi-node-driver-registrar@sha256:ba763bb01ddc09e312240c8abc310aa2e2dd6aee636d342f6dd9238a6bff179c && \
docker tag quay.io/minio/csi-node-driver-registrar@sha256:ba763bb01ddc09e312240c8abc310aa2e2dd6aee636d342f6dd9238a6bff179c quay.io/<QUAY_USERNAME>/csi-node-driver-registrar && \
docker push quay.io/<QUAY_USERNAME>/csi-node-driver-registrar
```
4. Make sure `csi-provisioner`, `livenessprobe` and `csi-node-driver-registrar` repositories are `public` in your `quay.io` account.
5. Go to your direct-csi project root.
```bash
$ cd $GOPATH/src/github.com/minio/direct-csi
```
6. Hack, hack, hack...
7. Run go build
```bash
$ go build -v ./...
```
8. Run ./build.sh
```bash
$ ./build.sh
```
9. Run docker build to tag image.
```bash
$ docker build -t quay.io/<QUAY_USERNAME>/direct-csi:<NEW_BUILD_TAG> .
```
10. Push newly created image to your `quay.io` account.
```bash
$ docker push quay.io/<QUAY_USERNAME>/direct-csi:<NEW_BUILD_TAG>
```
11. Make sure `direct-csi` repository is `public` in your `quay.io` account.
12. Install direct-csi.
```bash
$ ./kubectl-direct_csi --kubeconfig <PATH-TO-KUBECONFIG-FILE> install \
--image direct-csi:<NEW_BUILD_TAG> --org <QUAY_USERNAME> --registry quay.io
```
13. Check running direct-csi
```bash
$ ./kubectl-direct_csi --kubeconfig <PATH-TO-KUBECONFIG-FILE> info

$ ./kubectl-direct_csi --kubeconfig <PATH-TO-KUBECONFIG-FILE> drives list
```

## Loopback Devices

DirectCSI can automatically provision loopback devices for setups where extra drives are not available. The loopback interface is intended for use with automated testing and continuous integration, and is not recommended for use in regular development or production environments. Some operating systems, such as macOS, place limits on the number of loop devices and can cause DirectCSI to hang while attempting to provision persistent volumes. This issue is particularly noticeable on Kubernetes deployment tools like `kind` or `minikube`, where the deployed infrastructure takes up most if not all of the available loop devices and prevents DirectCSI from provisioning drives entirely.
