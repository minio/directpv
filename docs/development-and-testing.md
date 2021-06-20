## Development and Testing
0. You would need to have `quay.io` account, running docker and kubernetes; and make sure `quay.io/<QUAY_USERNAME>/direct-csi` repository is `public`.
1. Go to project root
```bash
$ cd $GOPATH/src/github.com/minio/direct-csi
```
2. Hack, hack, hack...
3. Run go build
```bash
$ go build -v ./...
```
4. Run ./build.sh
```bash
$ ./build.sh
```
5. Docker login to your `quay.io` account
```bash
$ docker login --username <QUAY_USERNAME> https://quay.io
```
6. Run docker build to tag image
```bash
$ docker build -t quay.io/<QUAY_USERNAME>/direct-csi:<NEW_BUILD_TAG> .
```
7. Push newly created image to `quay.io`
```bash
$ docker push quay.io/<QUAY_USERNAME>/direct-csi:<NEW_BUILD_TAG>
```
8. Install direct-csi
```bash
$ ./kubectl-direct_csi --kubeconfig <PATH-TO-KUBECONFIG-FILE> install --image direct-csi:<NEW_BUILD_TAG> --org <QUAY_USERNAME>
```

## Loopback Devices

DirectCSI can automatically provision loopback devices for setups where extra drives are not available. The loopback interface is intended for use with automated testing and continuous integration, and is not recommended for use in regular development or production environments. Some operating systems, such as macOS, place limits on the number of loop devices and can cause DirectCSI to hang while attempting to provision persistent volumes. This issue is particularly noticeable on Kubernetes deployment tools like `kind` or `minikube`, where the deployed infrastructure takes up most if not all of the available loop devices and prevents DirectCSI from provisioning drives entirely.
