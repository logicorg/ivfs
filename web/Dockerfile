# The gateway, with the web UI's build embedded (ADR 0007). Built from the
# repository's root, which holds the module: docker build -f web/Dockerfile .
#
# The builders run on the build machine's platform: the JavaScript build is
# the same on every platform, and Go cross-compiles, which is faster than
# emulating the target's.
FROM --platform=$BUILDPLATFORM node:24.11-trixie-slim@sha256:4623e8ac1eb5f35b0a91b85424283c1b97a416a77375dfa6a2e9bfb0b2c351c9 AS ui

ENV COREPACK_ENABLE_DOWNLOAD_PROMPT=0

WORKDIR /build/web/ui

# The lockfile alone first, so a source change reuses the installed
# dependencies.
COPY web/ui/package.json web/ui/pnpm-lock.yaml ./

RUN corepack pnpm install --frozen-lockfile

COPY web/ui ./

RUN corepack pnpm build

FROM --platform=$BUILDPLATFORM golang:1.27.3-trixie@sha256:af1bfc2614ebb1dbd31af514b78740fd678850e76498762b425ecee4c5c6ea80 AS builder

ARG TARGETOS TARGETARCH

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY --parents ./finance ./internal ./pkg ./provisioning ./web ./
COPY --from=ui /build/web/ui/dist ./web/ui/dist

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
	go build -trimpath -ldflags="-s -w" -o /out/gateway ./web

# Static, so no libc or shell; the CA certificates are for the identity
# provider's TLS.
FROM gcr.io/distroless/static-debian13:nonroot@sha256:aafe993665bf6200db38a02c11a9351e782a84b6a45e7f33b6695638c88e3931

LABEL org.opencontainers.image.source=https://github.com/logicorg/ivfs

COPY --from=builder /out/gateway /gateway

ENTRYPOINT [ "/gateway" ]
