package main

// version is the plugin version reported via `docker buildx version`. Override
// at build time with
//
//	-ldflags "-X main.version=v1.2.3"
var version = "0.1.0-dev"

// buildxCompatVersion is the docker buildx version this plugin advertises in
// its CLI-plugin metadata handshake. Buildx-aware tooling gates features on the
// reported buildx version — notably `docker compose build` delegates to
// `docker buildx bake` only when buildx is >= 0.17.0 — so the shim must claim a
// compatible version even though it is buildah-backed. This is the buildx API
// surface we emulate, not the project's own version (see version above).
const buildxCompatVersion = "v0.18.0"
