FROM alpine:3.20

ARG TARGETOS
ARG TARGETARCH

RUN apk --no-cache add ca-certificates tzdata

COPY ${TARGETOS}/${TARGETARCH}/container-ctl /usr/local/bin/container-ctl

ENTRYPOINT ["/usr/local/bin/container-ctl"]
