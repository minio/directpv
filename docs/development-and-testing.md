## Development and Testing
1. Have `quay.io` account, running docker and kubernetes. You could use any registry account and replace `quay.io` with such registry name in below steps.
2. Docker login to your `quay.io` account.
```bash
$ docker login --username <QUAY_USERNAME> https://quay.io
```
3. Create `csi-provisioner`, `livenessprobe` and `csi-node-driver-registrar` repositories by pull/tag/push respective images to your `quay.io` account.
```bash
$ docker pull quay.io/minio/csi-provisioner@sha256:c185db49ba02c384633165894147f8d7041b34b173e82a49d7145e50e809b8d6 && \
docker tag quay.io/minio/csi-provisioner@sha256:c185db49ba02c384633165894147f8d7041b34b173e82a49d7145e50e809b8d6 quay.io/<QUAY_USERNAME>/csi-provisioner && \
docker push quay.io/<QUAY_USERNAME>/csi-provisioner

$ docker pull quay.io/minio/livenessprobe@sha256:a3a5f8e046ece910505a7f9529c615547b1152c661f34a64b13ac7d9e13df4a7 && \
docker tag quay.io/minio/livenessprobe@sha256:a3a5f8e046ece910505a7f9529c615547b1152c661f34a64b13ac7d9e13df4a7 quay.io/<QUAY_USERNAME>/livenessprobe && \
docker push quay.io/<QUAY_USERNAME>/livenessprobe

$ docker pull quay.io/minio/csi-node-driver-registrar@sha256:d46524376ffccf2c29f2fb373a67faa0d14a875ae01380fa148b4c5a8d47a6c6 && \
docker tag quay.io/minio/csi-node-driver-registrar@sha256:d46524376ffccf2c29f2fb373a67faa0d14a875ae01380fa148b4c5a8d47a6c6 quay.io/<QUAY_USERNAME>/csi-node-driver-registrar && \
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
$ docker build -t quay.io/<QUAY_USERNAME>/directpv:<NEW_BUILD_TAG> .
```
10. Push newly created image to your `quay.io` account.
```bash
$ docker push quay.io/<QUAY_USERNAME>/directpv:<NEW_BUILD_TAG>
```
11. Make sure `directpv` repository is `public` in your `quay.io` account.
12. Install directpv.
```bash
$ ./kubectl-directpv --kubeconfig <PATH-TO-KUBECONFIG-FILE> install \
--image directpv:<NEW_BUILD_TAG> --org <QUAY_USERNAME> --registry quay.io
```
13. Check running directpv
```bash
$ ./kubectl-directpv --kubeconfig <PATH-TO-KUBECONFIG-FILE> info

$ ./kubectl-directpv --kubeconfig <PATH-TO-KUBECONFIG-FILE> drives list
```

## Testing with minikube

1. Setup LVs

The following script will create 4 LVs backed up by 4 loopback devices

```bash
sudo truncate --size=1G /tmp/disk-{1..4}.img
for disk in /tmp/disk-{1..4}.img; do sudo losetup --find $disk; done
devices=( $(for disk in /tmp/disk-{1..4}.img; do sudo losetup --noheadings --output NAME --associated $disk; done) )
sudo pvcreate "${devices[@]}"
vgname="vg0"
sudo vgcreate "$vgname" "${devices[@]}"
for lvname in lv-{0..3}; do sudo lvcreate --name="$lvname" --size=800MiB "$vgname"; done
```

2. Start minikube

```bash
minikube start --driver=none
```

3. Install directpv

Install the freshly built version

```bash
./kubectl-directpv install --image directpv:<NEW_BUILD_TAG> --org <QUAY_USERNAME> --registry quay.io
```

4. Check if the drives are showing up

```bash
./kubectl-directpv drives list
```

5. Format the drives

```bash
./kubectl-directpv drives format --all
```

6. Apply the minio.yaml file

Download and apply a sample MinIO deployment file available [here](https://github.com/minio/directpv/blob/master/minio.yaml)

```bash
kubectl apply -f minio.yaml
```

7. Check if the pods are up and running

```bash
kubectl get pods
```

8. Check the volumes

```bash
./kubectl-directpv volumes list
```

9. Check the drives if they are in "InUse" state

```bash
./kubectl-directpv drives list
```

10. Uninstall the MinIO deployment

```bash
kubectl delete -f minio.yaml
```

11. Delete the PVCs

```bash
kubectl delete pvc --all
```

After deleting the PVCs, check if the drives are back in "Ready" state.

12. Release the "Ready" drives

```bash
./kubectl-directpv drives release --all
```

This should make all the "Ready" drives "Available" by umounting the drives in the host.

13. Cleanup LV setup

```sh
sudo lvremove vg0 -y
sudo vgremove vg0 -y
sudo pvremove /dev/loop<n> /dev/loop<n> /dev/loop<n> /dev/loop<n> # n can be replaced with the loopbacks created
sudo losetup --detach-all
```

Please refer [here](./troubleshooting.md) for any trouble shooting guidelines.

## Loopback Devices

DirectPV can automatically provision loopback devices for setups where extra drives are not available. The loopback interface is intended for use with automated testing and continuous integration, and is not recommended for use in regular development or production environments. Some operating systems, such as macOS, place limits on the number of loop devices and can cause DirectPV to hang while attempting to provision persistent volumes. This issue is particularly noticeable on Kubernetes deployment tools like `kind` or `minikube`, where the deployed infrastructure takes up most if not all of the available loop devices and prevents DirectPV from provisioning drives entirely.
