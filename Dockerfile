FROM alpine:latest as certs

RUN apk add -U --no-cache ca-certificates

COPY direct-csi /direct-csi

WORKDIR /

ENTRYPOINT ["/direct-csi"]
