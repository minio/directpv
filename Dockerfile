FROM golang:1.14

WORKDIR	/go/src/github.com/minio/direct-csi
ADD .   /go/src/github.com/minio/direct-csi
RUN ./build.sh

FROM alpine:latest

WORKDIR /
RUN apk add xfsprogs 
RUN apk add e2fsprogs
RUN apk add dosfstools
COPY --from=0 /go/src/github.com/minio/direct-csi/direct-csi /direct-csi

ENTRYPOINT ["/direct-csi"]
