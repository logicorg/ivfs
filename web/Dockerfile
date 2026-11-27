# The gateway, with the web UI's build embedded (ADR 0007). Built from the
# repository's root, which holds the module: docker build -f web/Dockerfile .
#
# The builders run on the build machine's platform: the JavaScript build is
# the same on every platform, and Go cross-compiles, which is faster than
# emulating the target's.
FROM --platform=$BUILDPLATFORM node:26.13-trixie-slim@sha256:e6bedef217677b276c591795e169840edd9096dbf13f29df8ef8d1856b8820bc AS ui

ENV COREPACK_ENABLE_DOWNLOAD_PROMPT=0

WORKDIR /build/web/ui

RUN npm install --global pnpm@10.34.9

# The lockfile alone first, so a source change reuses the installed
# dependencies.
COPY web/ui/package.json web/ui/pnpm-lock.yaml ./

RUN pnpm install --frozen-lockfile

COPY web/ui ./

RUN pnpm build

FROM --platform=$BUILDPLATFORM golang:1.27.4-trixie@sha256:f18c3d73a184fb0582828510722e723b63a93f7188d11dd223d9be6f6880e65d AS builder

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
FROM gcr.io/distroless/static-debian13:nonroot@sha256:8ce6dd87e9823c5b48dcfb2c8604714ccb351ce59607327bcc4957448da279b5

LABEL org.opencontainers.image.source=https://github.com/logicorg/ivfs

COPY --from=builder /out/gateway /gateway

ENTRYPOINT [ "/gateway" ]
