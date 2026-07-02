<div align="center">

<img src="assets/brand/wordmark.png" alt="fruitbox" width="420" />

**Docker Compose for Apple's `container` runtime.**

Run the `compose.yaml` you already have on Apple silicon, without a Docker daemon.

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Compose Spec](https://img.shields.io/badge/Compose-spec--faithful-6B9B8A)](https://compose-spec.io)
[![Status](https://img.shields.io/badge/status-early-C4874B)](#status)

[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/urjit)


</div>

---

Apple's [`container`](https://github.com/apple/container) runs each container in its
own lightweight VM on Apple silicon, but it doesn't read Compose files. fruitbox is the
piece in between: point it at an existing `compose.yaml` and it brings the whole stack up
with the commands you already use from `docker compose`.

Compose files are parsed with [`compose-go`](https://github.com/compose-spec/compose-go),
the same library Docker Compose uses, so fruitbox sees exactly what Docker Compose sees:
interpolation, merging, profiles, and `extends` all behave the same way. From there it
translates the resolved project into `container` commands and handles startup order,
health, restarts, and teardown.

<div align="center">

<img src="assets/demo.gif" alt="fruitbox up → ps → logs → down" width="760" />

</div>

## Requirements

- macOS 15 or newer on Apple silicon
- Apple's [`container`](https://github.com/apple/container) installed and started
  (`container system start`)
- Go 1.25+ to build

## Install

Download the latest `darwin_arm64` archive from the
[releases page](https://github.com/urjitbhatia/fruitbox/releases), then:

```bash
tar -xzf fruitbox_*_darwin_arm64.tar.gz
sudo mv fruitbox /usr/local/bin/
xattr -d com.apple.quarantine /usr/local/bin/fruitbox   # the binary isn't notarized yet
```

Or with Go:

```bash
go install github.com/urjitbhatia/fruitbox/cmd/fruitbox@latest
```

Or build from source:

```bash
git clone https://github.com/urjitbhatia/fruitbox && cd fruitbox
go build -o fruitbox ./cmd/fruitbox
```

## Usage

In any directory with a `compose.yaml`:

```bash
fruitbox up -d          # build, create, and start everything
fruitbox ps             # see what's running
fruitbox logs -f        # tail logs, multiplexed and color-prefixed per service
fruitbox exec web sh    # shell into a service
fruitbox down           # tear it all down
```

The global flags match Docker Compose: `-f/--file`, `-p/--project-name`, `--profile`,
`--env-file`, and `--project-directory`. Use `--container-binary` to point at a
non-default `container`.

## How it works

fruitbox keeps parsing and orchestration separate. It reuses `compose-go` for loading so
it never reimplements the spec, and owns the translation into `container` calls:

| Package | Responsibility |
|---|---|
| `internal/compose` | Loads compose files into a resolved `types.Project` (wraps compose-go) |
| `internal/translate` | Converts services, networks, and volumes into `container` argument vectors |
| `internal/runner` | Runs the `container` CLI (mockable for tests) |
| `internal/engine` | Orchestration: dependency order, recreate, health, restart, logs |
| `internal/cli` | Cobra command tree mirroring `docker compose` |

Containers are named `<project>-<service>-<n>` and carry the standard
`com.docker.compose.*` labels alongside an `io.fruitbox.*` set, so they stay visible to
Docker tooling without depending on it.

A few orchestration details worth knowing:

- Re-running `up` reuses containers that are already up to date (compared by config hash)
  and skips networks and volumes that already exist.
- Lifecycle commands take a per-project advisory lock, so two commands that mutate the
  same project can't race. A blocked one fails fast and prints the holder's PID; read-only
  commands never lock.
- A foreground `up` stops the project's containers on the first `Ctrl-C` and force-quits
  on the second, matching `docker compose`.

## What works

The goal is to run your `compose.yaml` unchanged. fruitbox supports the parts of the
service model the runtime can express: `image`, `build`, `command`, `entrypoint`,
`environment`, `ports`, `volumes`, `networks`, `depends_on` (including `service_healthy`
and `service_completed_successfully`), `healthcheck`, `restart`, `secrets`, `configs`,
`cpus`, `mem_limit`, `cap_add`/`cap_drop`, `dns`, `scale`, and more.

Some things the runtime can't do, and fruitbox tells you instead of pretending:

- The runtime doesn't run healthchecks, so fruitbox runs them itself.
- There's no daemon to apply `restart:` policies, so a foreground `up` watches services
  and restarts them.
- `hostname`, `extra_hosts`, and `sysctls` are emulated (a generated `/etc/hosts`,
  `sysctl` after start). `privileged`, `devices`, `mac_address`, and `group_add` cross the
  VM isolation boundary and print a `WARNING`.
- A few `docker compose` flags map to features that don't exist, such as `logs --since`
  (the runtime keeps no per-line timestamps). fruitbox leaves these unimplemented rather
  than accepting and ignoring them.

[COMPATIBILITY.md](./COMPATIBILITY.md) has the full breakdown, kept in sync by a
flag-parity test and a differential test that runs against a real `docker compose`
install.

<details>
<summary><strong>All 32 commands</strong> (click to expand)</summary>

| Command | Notes |
|---|---|
| `config` | parse, resolve, and render canonical YAML/JSON; `--services/--networks/--volumes/--images/--profiles/--hash/--environment` |
| `up` | create and start in dependency order; recreate on config change; foreground log streaming with restart supervision |
| `down` | stop and remove containers (reverse order), networks, and optionally volumes/images |
| `ps` | live image/status/ports; `--format json`, `--filter`, `--status`, `--all`, `--services` |
| `logs` | concurrent multiplexed streaming with color per-service prefixes; `--tail/--follow/--timestamps` |
| `build` | `build:` → `container build`; `--build-arg/--no-cache/--pull/--push/--with-dependencies` |
| `run` | one-off with deps; `--entrypoint/--user/--volume/--publish/--service-ports/--rm` |
| `exec` | run a command in a service container (`-it`, `--index`, `--detach`) |
| `create` / `start` / `stop` / `restart` / `kill` | lifecycle control (dependency-ordered) |
| `pull` / `push` | image transfer (`--include-deps`, `--ignore-*-failures`, `--policy`) |
| `pause` / `unpause` | suspend/resume via `SIGSTOP`/`SIGCONT` |
| `scale` | scale services up or down |
| `cp` | copy files to/from a service container |
| `port` | resolve the published host port |
| `top` | in-container `ps` |
| `images` / `volumes` | list images / volumes used by the project |
| `ls` | list running compose projects (runtime-wide) |
| `stats` | live resource usage (`container stats`) |
| `export` | export a container filesystem to a tar |
| `events` | stream lifecycle events |
| `wait` | block until containers stop, then print the exit code |
| `attach` | attach to a container's I/O |
| `watch` | sync/restart/rebuild on source change (`develop.watch`) |
| `version` | |

</details>

## Status

fruitbox is early. 32 of `docker compose`'s commands work against the real runtime,
including `up`, `down`, `ps`, `logs`, `exec`, `build`, `run`, and `watch`, with
config-hash recreation. It isn't at 1.0 yet, so expect sharp edges. Bug reports are
welcome, especially `compose.yaml` files that don't work.

## Development

```bash
go build ./cmd/fruitbox   # build
go test ./...             # unit suite (no container runtime needed)
make test-integration     # integration lane against the real container runtime
make compat               # flag audit against a local docker compose
```

The unit suite runs translation and orchestration against a fake runner, so you don't
need `container` installed to work on fruitbox. The integration lane (build tag
`integration`) drives the real runtime end to end and skips itself when it isn't
available.

## Contributing

Bug reports, compose files that don't work, and PRs are all welcome. See
[CONTRIBUTING.md](./CONTRIBUTING.md) for the details. In short: changes come with a test,
flag changes update the parity ratchet in `internal/cli`, and `gofmt`/`go vet` stay clean.

## License

[MIT](./LICENSE) © fruitbox contributors.

## Acknowledgements

- [Apple `container`](https://github.com/apple/container), the runtime fruitbox drives.
- [`compose-spec/compose-go`](https://github.com/compose-spec/compose-go), the Compose
  reference parser that makes spec-faithful loading possible.

---

<sub>fruitbox is an independent project, not affiliated with, endorsed by, or sponsored by
Apple Inc. or Docker, Inc. "Apple" and "`container`" are referenced descriptively to
indicate runtime compatibility; "Docker" and "Docker Compose" are trademarks of Docker,
Inc.</sub>
</content>
