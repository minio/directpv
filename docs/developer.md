# Development and Testing
We welcome any contribution to DirectPV. All your pull requests are welcomed.

## Building DirectPV from source
1. Get source code from [DirectPV Github repository](https://github.com/minio/directpv)
2. Run `./build.sh`. You will get `directpv` CSI driver binary and `kubectl-directpv` plugin binary.

## Building DirectPV container image
1. Run `podman build -t quay.io/minio/directpv:$(./kubectl-directpv --version | awk '{ print $NF }') .`

## Pushing container image to kubernetes nodes
1. Run `podman save --output directpv.tar quay.io/minio/directpv:$(./kubectl-directpv --version | awk '{ print $NF }')` to get image tarball.
2. Copy `directpv.tar` to all your kubernetes nodes.
3. Import `directpv.tar` by running `/usr/local/bin/ctr images import --digests --base-name quay.io/minio/directpv directpv.tar` on each node. This step works for `K3S` cluster.

## Installing DirectPV build
1. Run `./kubectl-directpv install`
