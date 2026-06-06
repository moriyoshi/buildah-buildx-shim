#!/usr/bin/env bash
# End-to-end test executed inside the container built from e2e/Dockerfile.
#
# It proves the full integration path works:
#
#   docker compose build           (official compose plugin)
#     -> docker buildx bake        (this project's shim, COMPOSE_BAKE=true)
#       -> buildah build           (the actual builder)
#         -> image in podman/buildah containers-storage
#
set -euo pipefail

log() { printf '\n=== %s ===\n' "$*"; }

IMAGE_REF="buildah-buildx-shim-e2e:web"
EXPECTED_OUTPUT="hello-from-bake"

# ---------------------------------------------------------------------------
# Expose podman's Docker-compatible API so the docker CLI and compose plugin
# have an engine to talk to. The build itself is handled by buildah via the
# shim, but compose/CLI still initialise an engine client.
# ---------------------------------------------------------------------------
log "Starting podman system service"
mkdir -p /run/podman
podman system service --time=0 unix:///run/podman/podman.sock &
for _ in $(seq 1 40); do
  [ -S /run/podman/podman.sock ] && break
  sleep 0.25
done
[ -S /run/podman/podman.sock ] || { echo "podman socket never appeared" >&2; exit 1; }
export DOCKER_HOST="unix:///run/podman/podman.sock"

log "Tool versions"
docker --version
docker compose version
buildah --version
podman --version

# The shim must be discovered as the buildx plugin and advertise a version that
# compose accepts (>= 0.17.0) — otherwise compose refuses to use bake.
log "docker buildx (provided by the shim)"
docker buildx version
docker buildx ls

# ---------------------------------------------------------------------------
# The actual test: compose build, routed through bake into buildah.
# ---------------------------------------------------------------------------
cd /work/fixtures
export COMPOSE_BAKE=true   # make compose delegate `build` to `docker buildx bake`

log "docker compose build"
docker compose build

# ---------------------------------------------------------------------------
# Assertions.
# ---------------------------------------------------------------------------
log "Verifying the image landed in buildah/podman storage"
if ! podman image exists "localhost/${IMAGE_REF}" && ! podman image exists "${IMAGE_REF}"; then
  echo "FAIL: expected image ${IMAGE_REF} not found in storage" >&2
  buildah images >&2
  exit 1
fi

log "Verifying the build arg propagated through compose -> bake -> buildah"
got="$(podman run --rm "localhost/${IMAGE_REF}")"
if [ "$got" != "$EXPECTED_OUTPUT" ]; then
  echo "FAIL: expected '${EXPECTED_OUTPUT}', got '${got}'" >&2
  exit 1
fi

log "E2E PASSED"
