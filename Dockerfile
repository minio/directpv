FROM golang:1.14

ADD      . /go/src/github.com/minio/jbod-csi-driver
WORKDIR	 /go/src/github.com/minio/jbod-csi-driver
RUN      ./build.sh

FROM alpine:latest

COPY --from=0 /go/src/github.com/minio/jbod-csi-driver/jbod-csi-driver /
WORKDIR /
ENTRYPOINT ["/jbod-csi-driver"]

