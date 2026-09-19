# Production image for the Go service.
#
# Two properties this file exists to guarantee, both of them checkable in the finished image:
#
#   1. Identity comes from verifying a Better Auth session against the database. There is no
#      development resolver in any build any more — the file was deleted in feature 004, so the
#      `production` build tag that used to exclude it guards nothing and is gone with it. A
#      binary that cannot verify a session refuses to start (FR-019, FR-019a).
#   2. The runtime stage is distroless: no shell, no package manager, no compiler, and a non-root
#      user (FR-017, FR-018). There is nothing in it to exec into.
#
# The development image is a separate file (Dockerfile.dev). One file trying to be both is how a
# compiler ends up in a shipped image.

# ------------------------------------------------------------------------------------------------
# Build
# ------------------------------------------------------------------------------------------------
# --platform=$BUILDPLATFORM pins the build stage to the host's architecture so the Go compiler runs
# natively and cross-compiles via the TARGETOS/TARGETARCH args below. Without it, building for a
# foreign architecture runs the whole toolchain under QEMU emulation — minutes instead of seconds,
# for an identical result.
FROM --platform=$BUILDPLATFORM golang:1.26-bookworm AS build

WORKDIR /src

# Dependencies as their own layer, so a source change does not re-download the module tree.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO off produces a static binary, which is what distroless/static can actually run.
# -trimpath drops build-machine paths; -s -w drop the symbol and DWARF tables, which is most of
# the binary's size and none of its behaviour.
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/vaultory-api ./cmd/vaultory-api

# The image store root, created here so it carries the right ownership into the runtime stage.
# Distroless has no shell, so there is no mkdir at runtime and no chown on a mounted volume: when
# Compose creates an empty named volume over this path it copies the image's ownership with it.
RUN mkdir -p /out/images && chown 65532:65532 /out/images

# ------------------------------------------------------------------------------------------------
# Runtime
# ------------------------------------------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot

# CA certificates and /etc/passwd come with the base image; nonroot is uid/gid 65532.
COPY --from=build /out/vaultory-api /usr/local/bin/vaultory-api
COPY --from=build --chown=65532:65532 /out/images /var/lib/vaultory/images

USER nonroot:nonroot

# 0.0.0.0, not 127.0.0.1: bound to loopback the service would be unreachable from every other
# container, including the frontend.
ENV VAULTORY_LISTEN_ADDR=0.0.0.0:8080 \
    VAULTORY_IMAGE_STORE=filesystem \
    VAULTORY_IMAGE_STORE_PATH=/var/lib/vaultory/images

EXPOSE 8080

# Exec form and no shell wrapper: PID 1 is the server itself, so SIGTERM reaches the graceful
# shutdown in main.go rather than a shell that ignores it.
ENTRYPOINT ["/usr/local/bin/vaultory-api"]
