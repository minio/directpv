FROM golang:1.14 as builder

WORKDIR	/go/src/github.com/minio/jbod-csi-driver

ADD . /go/src/github.com/minio/jbod-csi-driver

RUN go get github.com/google/addlicense && ./build.sh

FROM alpine:latest as certs

RUN apk add -U --no-cache ca-certificates

FROM scratch

COPY --from=builder /go/src/github.com/minio/jbod-csi-driver/jbod-csi-driver /
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /

ENTRYPOINT ["/jbod-csi-driver"]

