# 🍎📦 fruitbox

**Docker Compose for Apple's native `container` runtime.**

fruitbox runs multi-container applications defined in [Compose](https://compose-spec.io)
files on top of [Apple's `container`](https://github.com/apple/container) CLI —
the lightweight-VM container runtime for Apple silicon. The goal is **100%
Docker Compose API compatibility**: take any `compose.yaml` that works with
`docker compose` and run it unchanged with `fruitbox`.

```
fruitbox -f compose.yaml up -d
```

## Why

`docker compose` targets the Docker Engine API. Apple's `container` speaks a
different CLI and runs each container in its own lightweight VM. fruitbox is the
orchestration layer in between: it parses Compose files exactly the way Docker
Compose does, then translates the resolved project model into `container`
invocations and manages lifecycle, dependency ordering and resource grouping.

## Design

fruitbox reuses the **official Compose reference parser**
([`compose-spec/compose-go`](https://github.com/compose-spec/compose-go) — the
same library Docker Compose itself uses) for loading, interpolation, merging,
profiles and `extends`. This guarantees parsing fidelity with the latest spec.
fruitbox then owns translation and orchestration:

| Package | Responsibility |
|---|---|
| `internal/compose` | Loads compose files into a resolved `types.Project` (wraps compose-go) |
| `internal/translate` | Converts services/networks/volumes → `container` argument vectors |
| `internal/runner` | Executes the `container` CLI (mockable for tests) |
| `internal/engine` | Orchestration: dependency ordering, up / down / ps / logs |
| `internal/cli` | Cobra command tree mirroring `docker compose` |

Resources carry the canonical `com.docker.compose.*` labels and containers are
named `<project>-<service>-<n>`, so inspection and grouping stay compatible.

## Commands

| Command | Status |
|---|---|
| `fruitbox config` | ✅ parse, resolve & render canonical YAML/JSON; `--services`, `--volumes`, `-q` |
| `fruitbox up [-d]` | ✅ create networks & volumes, start services in dependency order |
| `fruitbox down [-v]` | ✅ stop & remove containers (reverse order), networks, optional volumes |
| `fruitbox ps [-q]` | ✅ list expected containers and live status |
| `fruitbox logs [-f] [svc...]` | ✅ stream container logs |
| `fruitbox build [svc...]` | ✅ build images (`build:` → `container build`) |
| `fruitbox start/stop/restart [svc...]` | ✅ lifecycle control (dependency-ordered) |
| `fruitbox kill [-s SIG] [svc...]` | ✅ signal containers |
| `fruitbox pull [svc...]` | ✅ pull service images (deduped) |
| `fruitbox exec [-it] SERVICE CMD` | ✅ run a command in a service container |
| `fruitbox run [--rm] SERVICE [CMD]` | ✅ one-off run; starts deps, command override |
| `fruitbox images [-q]` | ✅ list images used by services |
| `fruitbox port SERVICE PORT` | ✅ resolve published host port |
| `fruitbox cp SRC DEST` | ✅ copy files to/from a service container |
| `fruitbox ls [-q]` | ✅ list running compose projects (runtime-wide) |
| `fruitbox wait [svc...]` | ✅ block until containers stop, print exit code |
| `fruitbox version` | ✅ |

`up` supports `-d`, `--no-build`, `--scale SERVICE=N`, `--remove-orphans`.
`depends_on` conditions (`service_healthy`, `service_completed_successfully`)
are honored — fruitbox supervises healthchecks itself since the runtime does not.

Global flags mirror Docker Compose: `-f/--file`, `-p/--project-name`,
`--project-directory`, `--profile`, `--env-file`, plus `--container-binary`.

## Translation coverage

Service attributes mapped to `container run`/`create` today: `image`,
`command`, `entrypoint`, `environment`, `ports`, `volumes` (named/bind/tmpfs,
with project-scoped name resolution), `networks`, `user`, `working_dir`,
`read_only`, `init`, `cpus`, `mem_limit`, `shm_size`, `cap_add`/`cap_drop`,
`dns`/`dns_search`/`dns_opt`, `container_name`, `labels`, `scale`/`deploy.replicas`.

## Building & testing

```
go build ./cmd/fruitbox      # build the binary
go test ./...                # run the test suite (no container binary needed)
```

The test suite is hermetic: translation and orchestration are tested against a
fake runner, so no `container` install is required to develop fruitbox.

## Restart policies

A foreground `fruitbox up` (without `-d`) supervises the project: it blocks
until services stop and restarts containers per their `restart:` policy
(`no` / `always` / `unless-stopped` / `on-failure[:max]`, plus
`deploy.restart_policy`). Since Apple's runtime has no restart-policy daemon,
fruitbox performs supervision itself.

## Roadmap

- `top` / `events`
- live `ps` enrichment via `container ls` label queries
- runtime gaps Apple `container` can't express today (`hostname`, `extra_hosts`,
  `privileged`, `mac_address`, `sysctls`, `devices`) — currently warned, not dropped silently
- older Compose spec versions

## Requirements

- macOS 15+ on Apple silicon, with [`container`](https://github.com/apple/container) installed
- Go 1.25+ to build
