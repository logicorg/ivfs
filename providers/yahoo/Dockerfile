# Built from the repository's root, which holds the module:
# docker build -f providers/yahoo/Dockerfile .
#
# The builder runs on the build machine's platform and cross-compiles, which
# is faster than emulating the target's.
FROM --platform=$BUILDPLATFORM golang:1.27.4-trixie@sha256:f18c3d73a184fb0582828510722e723b63a93f7188d11dd223d9be6f6880e65d AS builder

ARG TARGETOS TARGETARCH

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY --parents ./internal ./pkg ./providers ./

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
	go build -trimpath -ldflags="-s -w" -o /out/yahoo-provider ./providers/yahoo

# Static, so no libc; the CA certificates are for Yahoo's TLS.
FROM gcr.io/distroless/static-debian13:nonroot@sha256:8ce6dd87e9823c5b48dcfb2c8604714ccb351ce59607327bcc4957448da279b5

LABEL org.opencontainers.image.source=https://github.com/logicorg/ivfs

COPY --from=builder /out/yahoo-provider /yahoo-provider

ENTRYPOINT [ "/yahoo-provider" ]
