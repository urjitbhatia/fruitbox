<div align="center">

<img src="assets/brand/wordmark.png" alt="fruitbox" width="420" />

**A `docker compose` for Apple's native `container` runtime.**

Run your multi-container apps from a `compose.yaml` on Apple silicon — no Docker daemon required.

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Compose Spec](https://img.shields.io/badge/Compose-spec--faithful-6B9B8A)](https://compose-spec.io)
[![Status](https://img.shields.io/badge/status-early-C4874B)](#status)

</div>

---

fruitbox takes the [Compose](https://compose-spec.io) file you already have and runs it on
[Apple's `container`](https://github.com/apple/container) — the lightweight-VM container
runtime built into macOS for Apple silicon. Same `compose.yaml`, same commands you know from
`docker compose`, but each service runs in its own fast native VM instead of a Docker daemon.

```console
$ fruitbox up -d
Creating network "myapp_default"
Starting myapp-db-1
Starting myapp-web-1

$ fruitbox ps
NAME          IMAGE          SERVICE  STATUS   PORTS
myapp-db-1    postgres:16    db       running
myapp-web-1   nginx:1.27     web      running  0.0.0.0:8080->80/tcp
```

## Quick start

You need **macOS 15+ on Apple silicon** with Apple's [`container`](https://github.com/apple/container)
installed and started (`container system start`), plus **Go 1.25+** to build.

```bash
# Install
go install github.com/urjitbhatia/fruitbox/cmd/fruitbox@latest

# …or build from source
git clone https://github.com/urjitbhatia/fruitbox && cd fruitbox
go build -o fruitbox ./cmd/fruitbox
```

Then, in any directory with a `compose.yaml`:

```bash
fruitbox up -d          # build, create, and start everything
fruitbox ps             # see what's running
fruitbox logs -f        # tail logs (multiplexed, color-prefixed per service)
fruitbox exec web sh    # shell into a service
fruitbox down           # tear it all down
```

Global flags mirror Docker Compose: `-f/--file`, `-p/--project-name`, `--profile`,
`--env-file`, `--project-directory`. (`--container-binary` points at a non-default `container`.)

## Why

`docker compose` talks to the Docker Engine API. Apple's `container` speaks a different CLI and
runs each container in its own lightweight VM. fruitbox is the orchestration layer in between:
it parses Compose files **exactly** the way Docker Compose does — using the same reference
library — then translates the resolved model into `container` invocations and manages lifecycle,
dependency ordering, health, restart policies, and resource grouping.

The headline benefit: keep your existing Compose workflow, drop the Docker daemon.

## Status

fruitbox is **early but already broadly usable** — 32 of `docker compose`'s commands are
implemented and exercised against the real runtime, including `up`/`down`/`ps`/`logs`/`exec`/
`build`/`run`/`watch` and config-hash-based recreation. It is **not yet at a 1.0**; expect rough
edges, and see [COMPATIBILITY.md](./COMPATIBILITY.md) for a precise, machine-checked breakdown of
exactly which commands and flags are supported and which can't be (and why).

Issues and PRs welcome — see [Contributing](#contributing).

## How it works

fruitbox reuses the **official Compose reference parser**
([`compose-spec/compose-go`](https://github.com/compose-spec/compose-go) — the same library
Docker Compose itself uses) for loading, interpolation, merging, profiles and `extends`. That
guarantees parsing fidelity with the latest spec. fruitbox owns the translation and orchestration:

| Package | Responsibility |
|---|---|
| `internal/compose` | Loads compose files into a resolved `types.Project` (wraps compose-go) |
| `internal/translate` | Converts services/networks/volumes → `container` argument vectors |
| `internal/runner` | Executes the `container` CLI (mockable for tests) |
| `internal/engine` | Orchestration: dependency order, recreate, health, restart, logs |
| `internal/cli` | Cobra command tree mirroring `docker compose` |

Resources carry the canonical `com.docker.compose.*` labels **and** a fruitbox-native
`io.fruitbox.*` mirror, and containers are named `<project>-<service>-<n>` — so they stay
compatible with Docker tooling while also being identifiable without it.

<details>
<summary><strong>All 32 commands</strong> (click to expand)</summary>

| Command | Notes |
|---|---|
| `config` | parse, resolve & render canonical YAML/JSON; `--services/--networks/--volumes/--images/--profiles/--hash/--environment` |
| `up` | create + start in dependency order; recreate on config change; foreground log streaming + restart supervision |
| `down` | stop & remove containers (reverse order), networks, optional volumes/images |
| `ps` | live image/status/ports; `--format json`, `--filter`, `--status`, `--all`, `--services` |
| `logs` | concurrent multiplexed streaming with color per-service prefixes; `--tail/--follow/--timestamps` |
| `build` | `build:` → `container build`; `--build-arg/--no-cache/--pull/--push/--with-dependencies` |
| `run` | one-off with deps; `--entrypoint/--user/--volume/--publish/--service-ports/--rm` |
| `exec` | run a command in a service container (`-it`, `--index`, `--detach`) |
| `create` / `start` / `stop` / `restart` / `kill` | lifecycle control (dependency-ordered) |
| `pull` / `push` | image transfer (`--include-deps`, `--ignore-*-failures`, `--policy`) |
| `pause` / `unpause` | suspend/resume via `SIGSTOP`/`SIGCONT` |
| `scale` | scale services up/down |
| `cp` | copy files to/from a service container |
| `port` | resolve the published host port |
| `top` | in-container `ps` |
| `images` / `volumes` | list images / volumes used by the project |
| `ls` | list running compose projects (runtime-wide) |
| `stats` | live resource usage (`container stats`) |
| `export` | export a container filesystem to a tar |
| `events` | stream lifecycle events |
| `wait` | block until containers stop, print exit code |
| `attach` | attach to a container's I/O |
| `watch` | sync/restart/rebuild on source change (`develop.watch`) |
| `version` | |

</details>

## Compatibility & the honest bits

The goal is to run your existing `compose.yaml` unchanged. fruitbox supports the full Compose
**service model** the runtime can express — `image`, `build`, `command`, `entrypoint`,
`environment`, `ports`, `volumes`, `networks`, `depends_on` (incl. `service_healthy` /
`service_completed_successfully`), `healthcheck`, `restart`, `secrets`/`configs`, `cpus`,
`mem_limit`, `cap_add/drop`, `dns`, `scale`, and more.

Some things Apple's runtime simply can't do, and fruitbox is **upfront** about them rather than
faking them:

- **Healthchecks** aren't run by the runtime, so fruitbox supervises them itself.
- **`restart:` policies** have no daemon, so a foreground `up` supervises and restarts.
- A few attributes have no runtime equivalent — `hostname`, `extra_hosts`, `sysctls` are
  **emulated** (generated `/etc/hosts`, post-start `sysctl`, …); `privileged`, `devices`,
  `mac_address`, `group_add` are **true VM-isolation boundaries** and emit an explicit `WARNING`.
- A handful of `docker compose` flags map to runtime features that don't exist (e.g. `logs --since`
  — the runtime stores no per-line timestamps). These are documented, not silently ignored.

The full, **machine-verified** breakdown — including a `TestFlagParity` ratchet and a differential
test against a real `docker compose` install — lives in [COMPATIBILITY.md](./COMPATIBILITY.md).

## Robustness

- **Concurrency-safe**: lifecycle commands take a per-project advisory `flock`, so two mutating
  commands on the same project can't race; a blocked one fails fast with the holder's PID, and
  read-only commands never lock.
- **Graceful Ctrl-C**: a foreground `up` stops the project's containers on the first `SIGINT` and
  force-quits on the second — just like `docker compose`.
- **Idempotent**: re-running `up` reuses up-to-date containers (config-hash) and skips
  already-created networks/volumes.

## Development

```bash
go build ./cmd/fruitbox   # build
go test ./...             # hermetic unit suite (no container runtime needed)
make test-integration     # integration lane against the real `container` runtime
make compat               # differential flag-audit vs. a local `docker compose`
```

The unit suite is hermetic — translation and orchestration run against a fake runner, so no
`container` install is required to hack on fruitbox. The integration lane (build-tagged
`integration`) drives the real runtime end-to-end and skips automatically when it's unavailable.

## Contributing

Contributions are very welcome — bug reports, compose files that don't work, and PRs.

- Run `go test ./...` (and `make test-integration` if you have the runtime) before opening a PR.
- New behavior is test-first: translation/orchestration changes get a `runner.Fake` test; flag
  changes update the `TestFlagParity` ratchet in `internal/cli`.
- Keep `gofmt`/`go vet` clean.

See [CONTRIBUTING.md](./CONTRIBUTING.md) for the longer version.

## License

[MIT](./LICENSE) © fruitbox contributors.

## Acknowledgements

- [Apple `container`](https://github.com/apple/container) — the runtime fruitbox drives.
- [`compose-spec/compose-go`](https://github.com/compose-spec/compose-go) — the Compose
  reference parser that makes spec-faithful loading possible.

---

<sub>fruitbox is an independent project, not affiliated with, endorsed by, or sponsored by
Apple Inc. or Docker, Inc. "Apple" and "`container`" are referenced descriptively to indicate
runtime compatibility; "Docker" and "Docker Compose" are trademarks of Docker, Inc.</sub>
</content>
