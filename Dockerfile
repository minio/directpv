FROM golang:1.14 as builder

WORKDIR	/go/src/github.com/minio/direct-csi

ADD . /go/src/github.com/minio/direct-csi

RUN go get github.com/google/addlicense && ./build.sh

FROM alpine:latest as certs

RUN apk add -U --no-cache ca-certificates

COPY --from=builder /go/src/github.com/minio/direct-csi/direct-csi /

WORKDIR /

ENTRYPOINT ["/direct-csi"]
