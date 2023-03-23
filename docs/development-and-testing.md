## Development and Testing
1. Have `quay.io` account, running docker and kubernetes. You could use any registry account and replace `quay.io` with such registry name in below steps.
2. Docker login to your `quay.io` account.
```bash
$ docker login --username <QUAY_USERNAME> https://quay.io
```
3. Create `csi-provisioner`, `livenessprobe` and `csi-node-driver-registrar` repositories by pull/tag/push respective images to your `quay.io` account.
```bash
$ docker pull quay.io/minio/csi-provisioner:v3.4.0 && \
docker tag quay.io/minio/csi-provisioner:v3.4.0 quay.io/<QUAY_USERNAME>/csi-provisioner:v3.4.0 && \
docker push quay.io/<QUAY_USERNAME>/csi-provisioner:v3.4.0

$ docker pull quay.io/minio/livenessprobe:v2.9.0 && \
docker tag quay.io/minio/livenessprobe:v2.9.0 quay.io/<QUAY_USERNAME>/livenessprobe:v2.9.0 && \
docker push quay.io/<QUAY_USERNAME>/livenessprobe:v2.9.0

$ docker pull quay.io/minio/csi-node-driver-registrar:v2.6.3 && \
docker tag quay.io/minio/csi-node-driver-registrar:v2.6.3 quay.io/<QUAY_USERNAME>/csi-node-driver-registrar:v2.6.3 && \
docker push quay.io/<QUAY_USERNAME>/csi-node-driver-registrar:v2.6.3

$ docker pull quay.io/minio/csi-resizer:v1.7.0 && \
docker tag quay.io/minio/csi-resizer:v1.7.0 quay.io/<QUAY_USERNAME>/csi-resizer:v1.7.0 && \
docker push quay.io/<QUAY_USERNAME>/csi-resizer:v1.7.0
```
4. Make sure `csi-provisioner`, `livenessprobe`, `csi-node-driver-registrar` and `csi-resizer` repositories are `public` in your `quay.io` account.
5. Go to your DirectPV project root.
```bash
$ cd $GOPATH/src/github.com/minio/directpv
```
6. Hack, hack, hack...
7. Run ./build.sh
```bash
$ ./build.sh
```
8. Run docker build to tag image.
```bash
$ docker build -t quay.io/<QUAY_USERNAME>/directpv:<NEW_BUILD_TAG> .
```
9. Push newly created image to your `quay.io` account.
```bash
$ docker push quay.io/<QUAY_USERNAME>/directpv:<NEW_BUILD_TAG>
```
10. Make sure `directpv` repository is `public` in your `quay.io` account.
11. Install DirectPV.
```bash
$ ./kubectl-directpv --kubeconfig <PATH-TO-KUBECONFIG-FILE> install \
--image directpv:<NEW_BUILD_TAG> --org <QUAY_USERNAME> --registry quay.io
```
12. Check running DirectPV
```bash
$ ./kubectl-directpv --kubeconfig <PATH-TO-KUBECONFIG-FILE> info

$ ./kubectl-directpv --kubeconfig <PATH-TO-KUBECONFIG-FILE> list drives
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

3. Install DirectPV

Install the freshly built version

```bash
./kubectl-directpv install --image directpv:<NEW_BUILD_TAG> --org <QUAY_USERNAME> --registry quay.io
```

4. Discover and initialize the drives

```bash
./kubectl-directpv discover --output-file drives.yaml
./kubectl-directpv init drives.yaml
```

5. Check if the drives are showing up

```bash
./kubectl-directpv list drives
```

6. Apply the minio.yaml file

Download and apply a sample MinIO deployment file available [here](https://github.com/minio/directpv/blob/master/functests/minio.yaml)

```bash
kubectl apply -f minio.yaml
```

7. Check if the pods are up and running

```bash
kubectl get pods
```

8. Check the volumes

```bash
./kubectl-directpv list volumes
```

9. Check the drives contain volumes

```bash
./kubectl-directpv list drives
```

10. Uninstall the MinIO deployment

```bash
kubectl delete -f minio.yaml
```

11. Delete the PVCs

```bash
kubectl delete pvc --all
```

After deleting the PVCs, check if the drives are freed up.

12. Release freed drives

```bash
./kubectl-directpv remove --all
```

13. Cleanup LV setup

```sh
sudo lvremove vg0 -y
sudo vgremove vg0 -y
sudo pvremove /dev/loop<n> /dev/loop<n> /dev/loop<n> /dev/loop<n> # n can be replaced with the loopbacks created
sudo losetup --detach-all
```

Please refer [here](./troubleshooting.md) for any trouble shooting guidelines.
