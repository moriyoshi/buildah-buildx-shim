# End-to-end test

This test exercises the whole point of the project: that the **official Docker
Compose plugin** can build images on a host where the engine is **podman /
buildah**, using this project's `docker-buildx` shim in place of BuildKit-backed
buildx.

The path under test:

```
docker compose build            # official compose plugin
  └─ docker buildx bake         # this project's shim (COMPOSE_BAKE=true)
       └─ buildah build         # the real builder
            └─ image in podman/buildah containers-storage
```

## Layout

| file                        | role                                                        |
| --------------------------- | ----------------------------------------------------------- |
| `Dockerfile`                | builds a container with podman, buildah, the official docker CLI + compose plugin, and the shim installed as a cli-plugin |
| `entrypoint.sh`             | runs `docker compose build` and asserts the image was built |
| `fixtures/docker-compose.yml` + `fixtures/Dockerfile` | the sample project that gets built |
| `run.sh`                    | host-side driver: build the image and run it `--privileged` |

## Running

```sh
e2e/run.sh                 # auto-detects podman or docker on the host
ENGINE=docker e2e/run.sh   # force the engine used to build/run the test image
```

The container runs `--privileged` because the nested podman/buildah needs it to
build. A successful run ends with `=== E2E PASSED ===`.

## What it asserts

1. The shim is discovered as the `docker buildx` plugin and advertises a buildx
   version compose accepts (compose requires buildx ≥ 0.17.0 for its `bake`
   path — see `buildxCompatVersion` in `version.go`).
2. `docker compose build` (with `COMPOSE_BAKE=true`) completes successfully.
3. The resulting image exists in podman/buildah storage.
4. The image's output reflects the `GREETING` build arg, proving the value was
   plumbed all the way through compose → bake → buildah.

> `COMPOSE_BAKE=true` is required: without it, compose tries to drive BuildKit
> through the daemon directly, which a buildah/podman host does not provide. The
> bake path is exactly what routes the build through this shim.
