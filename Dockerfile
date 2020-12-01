FROM golang:1.14 AS build

WORKDIR /go/src/github.com/minio/direct-csi
ADD ./ ./
RUN hack/update-codegen.sh
RUN \
    REPOSITORY=github.com/minio/direct-csi \
    CSI_VERSION=$(git describe --tags --always --dirty) \
    CGO_ENABLED=0 \
    go build -tags "osusergo netgo static_build" -ldflags="-X ${REPOSITORY}/cmd.Version=${CSI_VERSION} -extldflags=-static"

FROM alpine:latest

WORKDIR /
RUN apk add xfsprogs
RUN apk add e2fsprogs
RUN apk add dosfstools
COPY --from=build /go/src/github.com/minio/direct-csi/direct-csi /direct-csi

ENTRYPOINT ["/direct-csi"]
