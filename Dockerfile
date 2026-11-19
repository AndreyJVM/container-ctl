FROM alpine:3.20

ARG TARGETOS
ARG TARGETARCH

RUN apk --no-cache add ca-certificates tzdata

COPY ${TARGETOS}/${TARGETARCH}/dockpulse /usr/local/bin/dockpulse

ENTRYPOINT ["/usr/local/bin/dockpulse"]
