FROM golang:1.14 AS build

WORKDIR /go/src/github.com/minio/direct-csi
ADD ./ ./
RUN hack/update-codegen.sh
RUN \
    REPOSITORY=github.com/minio/direct-csi \
    CSI_VERSION=$(git describe --tags --always --dirty) \
    CGO_ENABLED=0 \
    set -x && go build -tags "osusergo netgo static_build" -ldflags="-X ${REPOSITORY}/cmd/direct-csi/cmd.Version=${CSI_VERSION} -extldflags=-static" ${REPOSITORY}/cmd/direct-csi

FROM alpine:latest

WORKDIR /
COPY CREDITS /licenses/CREDITS
COPY LICENSE /licenses/LICENSE

RUN apk add -U --no-cache ca-certificates
RUN apk add xfsprogs
RUN apk add xfsprogs-extra
COPY --from=build /go/src/github.com/minio/direct-csi/direct-csi /direct-csi

ENTRYPOINT ["/direct-csi"]
