#!/usr/bin/env bash
# Host-side driver for the E2E test. Builds the test image (e2e/Dockerfile) and
# runs it, which executes e2e/entrypoint.sh inside the container.
#
# Usage:
#   e2e/run.sh                 # auto-detect podman/docker
#   ENGINE=docker e2e/run.sh   # force a specific engine
#
# The container needs --privileged so the nested podman/buildah can build.
set -euo pipefail

cd "$(dirname "$0")/.."   # repository root (build context)

ENGINE="${ENGINE:-}"
if [ -z "$ENGINE" ]; then
  if command -v podman >/dev/null 2>&1; then
    ENGINE=podman
  elif command -v docker >/dev/null 2>&1; then
    ENGINE=docker
  else
    echo "neither podman nor docker found; set ENGINE=..." >&2
    exit 1
  fi
fi

IMAGE="${IMAGE:-buildah-buildx-shim-e2e-runner}"

echo ">> building test image with $ENGINE"
"$ENGINE" build -t "$IMAGE" -f e2e/Dockerfile .

echo ">> running E2E test"
exec "$ENGINE" run --rm --privileged "$IMAGE"
