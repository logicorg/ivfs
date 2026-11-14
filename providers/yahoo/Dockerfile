# Built from the repository's root, which holds the module:
# docker build -f providers/yahoo/Dockerfile .
#
# The builder runs on the build machine's platform and cross-compiles, which
# is faster than emulating the target's.
FROM --platform=$BUILDPLATFORM golang:1.27.3-trixie@sha256:af1bfc2614ebb1dbd31af514b78740fd678850e76498762b425ecee4c5c6ea80 AS builder

ARG TARGETOS TARGETARCH

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY --parents ./internal ./pkg ./providers ./

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
	go build -trimpath -ldflags="-s -w" -o /out/yahoo-provider ./providers/yahoo

# Static, so no libc; the CA certificates are for Yahoo's TLS.
FROM gcr.io/distroless/static-debian13:nonroot@sha256:aafe993665bf6200db38a02c11a9351e782a84b6a45e7f33b6695638c88e3931

LABEL org.opencontainers.image.source=https://github.com/logicorg/ivfs

COPY --from=builder /out/yahoo-provider /yahoo-provider

ENTRYPOINT [ "/yahoo-provider" ]
