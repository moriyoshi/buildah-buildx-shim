# buildah-buildx-shim

A Docker CLI plugin that implements the `docker buildx` command surface but
executes [**buildah**](https://buildah.io) underneath instead of BuildKit.

It lets buildx-style workflows — and the tooling that assumes them (Compose, CI
systems, `imagetools`) — run on hosts where buildah/podman is the container
engine.

## How it works

The plugin is a single binary, `docker-buildx`, built with the official
`github.com/docker/cli` plugin framework, so the metadata handshake and Docker
global-flag handling are exactly what the Docker CLI expects. Each `buildx`
subcommand is translated into one or more `buildah` invocations:

| buildx                         | buildah                                                           |
| ------------------------------ | ----------------------------------------------------------------- |
| `buildx build`                 | `buildah build` (flags mapped 1:1; see below)                     |
| `buildx build --push`          | `buildah build` then `buildah push` / `buildah manifest push`     |
| `buildx build --platform a,b`  | `buildah build --manifest` (assembles a manifest list)            |
| `buildx bake`                  | resolve HCL/JSON/compose targets → multiple `buildah build`s      |
| `buildx imagetools inspect`    | `buildah manifest inspect`                                        |
| `buildx imagetools create`     | `buildah manifest create` + `add` + `push`                        |
| `buildx version/ls/inspect`    | reported from the local buildah (single synthetic builder)        |
| `buildx create/use/rm/stop`    | no-ops (buildah has no separate builder instances)                |
| `buildx prune`                 | `buildah prune`                                                   |

## Build & install

```sh
go build -o docker-buildx .

# Install as a Docker CLI plugin:
mkdir -p ~/.docker/cli-plugins
cp docker-buildx ~/.docker/cli-plugins/docker-buildx

# Verify the plugin is recognised:
docker buildx version
```

`buildah` must be on `PATH`. Override the binary with `BUILDAH_BINARY=/path/to/buildah`.

## `build` flag support

**Mapped directly to buildah:** `-t/--tag`, `-f/--file`, `--build-arg`,
`--build-context`, `--target`, `--platform`, `--no-cache`, `--label`,
`--annotation`, `--network`, `--add-host`, `--cgroup-parent`, `--shm-size`,
`--ulimit`, `--secret`, `--ssh`, `-q/--quiet`, `--iidfile`, and the positional
context (`PATH | URL | -`).

**Adapted:** `--pull` → `buildah --pull=always`; `--cache-from`/`--cache-to`
(`type=registry,ref=…` or a bare ref) → buildah's bare-reference cache flags.

**Output:** `--push`, `--load`, and `--output` (`type=registry|image|docker|
oci|local|tar`) are resolved into the right build flags plus post-build
`buildah push`/`manifest push` steps. `--metadata-file` is synthesised from the
image ID.

**Accepted but ignored (with a warning):** `--progress`, `--provenance`,
`--sbom`, `--attest`, `--no-cache-filter`, `--allow`, `--builder`.

## bake

Reads `docker-bake.hcl` / `.json` / `docker-compose.yml` (or `-f`). Supports
`variable`/`group`/`target` blocks, variable interpolation (env overrides
defaults), `inherits`, `--set` overrides, `--print`, `--push`, and `--load`.

## Limitations

- The local **store is buildah/podman's** containers-storage. Against a *real*
  dockerd, `--load` does **not** copy the image into the Docker daemon's store
  (that would need `buildah push docker-daemon:…`; not yet implemented).
- `imagetools inspect --format` templates are not supported (raw manifest is
  printed); remote inspection is limited to what `buildah manifest inspect`
  returns, as `skopeo` is not required.
- bake **matrix** builds and the full bake **function library** are not
  implemented (reported and skipped).
- SBOM / provenance / attestations are not produced.

## Development

```sh
go test ./...
```

`internal/build` holds the build flag translation and output planning (with unit
tests); `internal/bake` the bake file parsing/resolution; `internal/cmd` the
cobra command tree; `internal/buildahexec` the buildah process wrapper.

### End-to-end test

`e2e/` contains a containerised end-to-end test that runs the official Docker
Compose plugin against podman/buildah with this shim installed as `docker
buildx`, proving `docker compose build` builds through buildah. Run it with:

```sh
e2e/run.sh
```

See [`e2e/README.md`](e2e/README.md) for details.

## License

This project is licensed under the [MIT License](LICENSE).
