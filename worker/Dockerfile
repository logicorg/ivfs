# Built from the repository's root, which holds the module:
# docker build -f worker/Dockerfile .
#
# The builder runs on the build machine's platform and cross-compiles, which
# is faster than emulating the target's.
FROM --platform=$BUILDPLATFORM golang:1.26.5-trixie@sha256:f1a132429b98724a904e9b3bdbaed399d8f923203c3e5170e6def66d0a7cc04c AS builder

ARG TARGETOS TARGETARCH

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY --parents ./finance ./internal ./pkg ./provisioning ./worker ./

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
	go build -trimpath -ldflags="-s -w" -o /out/worker ./worker

# Static, so no libc or shell; nonroot runs it as 65532.
FROM gcr.io/distroless/static-debian13:nonroot@sha256:25ac8f0c3b2334a9eb05452266bd4697864a8c8ef0ef71e5e6480d7f92868329

LABEL org.opencontainers.image.source=https://github.com/logicorg/ivfs

COPY --from=builder /out/worker /worker

ENTRYPOINT [ "/worker" ]
