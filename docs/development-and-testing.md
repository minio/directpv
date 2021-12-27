## Development and Testing
1. Have `quay.io` account, running docker and kubernetes. You could use any registry account and replace `quay.io` with such registry name in below steps.
2. Docker login to your `quay.io` account.
```bash
$ docker login --username <QUAY_USERNAME> https://quay.io
```
3. Create `csi-provisioner`, `livenessprobe` and `csi-node-driver-registrar` repositories by pull/tag/push respective images to your `quay.io` account.
```bash
$ docker pull quay.io/minio/csi-provisioner@sha256:d4f94539565cf62aea57062b6a42c5156337003133fd3f51b93df9a789e69840 && \
docker tag quay.io/minio/csi-provisioner@sha256:d4f94539565cf62aea57062b6a42c5156337003133fd3f51b93df9a789e69840 quay.io/<QUAY_USERNAME>/csi-provisioner && \
docker push quay.io/<QUAY_USERNAME>/csi-provisioner

$ docker pull quay.io/minio/livenessprobe@sha256:928a80be4d363e0e438ff28dcdb00d8d674d3059c6149a8cda64ce6016a9a3f8 && \
docker tag quay.io/minio/livenessprobe@sha256:928a80be4d363e0e438ff28dcdb00d8d674d3059c6149a8cda64ce6016a9a3f8 quay.io/<QUAY_USERNAME>/livenessprobe && \
docker push quay.io/<QUAY_USERNAME>/livenessprobe

$ docker pull quay.io/minio/csi-node-driver-registrar@sha256:843fb23b1a3fa1de986378b0b8c08c35f8e62499d386de8ec57801fd029afe6d && \
docker tag quay.io/minio/csi-node-driver-registrar@sha256:843fb23b1a3fa1de986378b0b8c08c35f8e62499d386de8ec57801fd029afe6d quay.io/<QUAY_USERNAME>/csi-node-driver-registrar && \
docker push quay.io/<QUAY_USERNAME>/csi-node-driver-registrar
```
4. Make sure `csi-provisioner`, `livenessprobe` and `csi-node-driver-registrar` repositories are `public` in your `quay.io` account.
5. Go to your direct-csi project root.
```bash
$ cd $GOPATH/src/github.com/minio/directpv
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
