FROM alpine:latest

WORKDIR /

RUN apk add -U --no-cache ca-certificates xfsprogs xfsprogs-extra

COPY direct-csi /direct-csi
COPY CREDITS /licenses/CREDITS
COPY LICENSE /licenses/LICENSE

ENTRYPOINT ["/direct-csi"]
