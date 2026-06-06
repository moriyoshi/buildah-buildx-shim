# E2E test image: podman + buildah (the engine), the official Docker CLI and
# Docker Compose plugin, and this project's buildah-backed `docker-buildx` shim
# installed as a Docker CLI plugin. Running this image executes the end-to-end
# test (see e2e/entrypoint.sh).
#
# Build context is the repository root:
#   podman build -t buildah-buildx-shim-e2e -f e2e/Dockerfile .

# ---------------------------------------------------------------------------
# Stage 1: build the shim from source.
# ---------------------------------------------------------------------------
FROM docker.io/library/golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Built like a normal release; the buildx-compatibility version is baked into
# the source (see version.go), so no special ldflags are needed.
RUN CGO_ENABLED=0 go build -o /out/docker-buildx .

# ---------------------------------------------------------------------------
# Stage 2: runtime with podman/buildah + the official docker CLI & compose.
# quay.io/podman/stable ships podman and buildah preconfigured for running
# containers inside a container.
# ---------------------------------------------------------------------------
FROM quay.io/podman/stable:latest

# Versions of the official Docker components to install. Pinned for
# reproducibility; bump as needed.
ARG DOCKER_CLI_VERSION=27.5.1
ARG COMPOSE_VERSION=v2.39.4
# Asset arch suffix shared by the docker static tarball and the compose binary
# (x86_64 / aarch64).
ARG ASSET_ARCH=x86_64

# buildah is the actual builder the shim drives. The podman/stable image ships
# podman but not the standalone buildah binary, so install it explicitly.
RUN dnf -y install --setopt=install_weak_deps=False buildah curl tar gzip \
 && dnf clean all \
 && buildah --version

# Official Docker CLI (static build) — provides the cli-plugin loader that
# discovers and runs `docker buildx` / `docker compose`.
RUN curl -fsSL "https://download.docker.com/linux/static/stable/${ASSET_ARCH}/docker-${DOCKER_CLI_VERSION}.tgz" \
      | tar -xz -C /usr/local/bin --strip-components=1 docker/docker \
 && docker --version

# Official Docker Compose plugin.
RUN mkdir -p /usr/local/lib/docker/cli-plugins \
 && curl -fsSL "https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-linux-${ASSET_ARCH}" \
      -o /usr/local/lib/docker/cli-plugins/docker-compose \
 && chmod +x /usr/local/lib/docker/cli-plugins/docker-compose

# This project's shim, installed as the `docker buildx` plugin.
COPY --from=build /out/docker-buildx /usr/local/lib/docker/cli-plugins/docker-buildx

COPY e2e/fixtures /work/fixtures
COPY e2e/entrypoint.sh /usr/local/bin/e2e-entrypoint.sh
RUN chmod +x /usr/local/bin/e2e-entrypoint.sh

WORKDIR /work
ENTRYPOINT ["/usr/local/bin/e2e-entrypoint.sh"]
