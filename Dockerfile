FROM golang:1.14 

WORKDIR	/go/src/github.com/minio/direct-csi
ADD . /go/src/github.com/minio/direct-csi
RUN go get github.com/google/addlicense && ./build.sh

ENTRYPOINT ["./direct-csi"]
