# Docker Compose compatibility

This file records how fruitbox's CLI lines up with `docker compose`. The numbers
come from diffing against a real `docker compose` install rather than from claims.
Regenerate it with:

```bash
go install ./cmd/fruitbox
./scripts/compat-audit.sh "$(go env GOPATH)/bin/fruitbox"
```

Baseline reference: `docker compose` v5.0.2, cross-checked against
<https://docs.docker.com/reference/cli/docker/compose/>.

## Automated tests

Two tests in `internal/cli` check this against a real `docker compose`. They
depend on the installed docker compose version, so they're opt-in: set
`FRUITBOX_COMPAT=1` (or run `make test-compat`) to run them. The default
`go test ./...` skips them, so CI stays hermetic.

- `TestConfigMatchesDockerCompose` runs fruitbox in-process and `docker compose`
  as a subprocess over a fixture matrix (`--services/--networks/--volumes/--profiles/--images`,
  profile activation, multi-file merge, and full YAML render) and asserts the
  output is identical. The rendered `config` YAML matches byte for byte because
  both use the compose-go marshaller.
- `TestFlagParity` computes the `docker compose` flags fruitbox is missing per
  command and asserts the set equals the `knownFlagGaps` baseline. Closing a gap,
  losing a flag (as the early `-f` crash did), or a change on the docker side all
  force a visible update to the baseline.

## Commands

**Implemented (32):** attach, build, config, cp, create, down, events, exec,
export, images, kill, logs, ls, pause, port, ps, pull, push, restart, rm, run,
scale, start, stats, stop, top, unpause, up, version, volumes, wait, watch.

**Not implemented (3) — no Apple `container` equivalent:**

| Command | Notes |
|---|---|
| `commit` | create an image from a container — the runtime has no commit |
| `publish` | publish a compose app to an OCI registry — compose-specific packaging |
| `bridge` | convert compose to Kubernetes/other — Docker Desktop feature, N/A |

## Flag coverage

Per-command flag gaps are tracked by `TestFlagParity` (the `knownFlagGaps`
baseline) and can be regenerated with `scripts/compat-audit.sh`.

Coverage started at 138 recorded gaps and is now 24. 21 of the 32 commands are at
full flag parity, and the remaining 24 gaps are runtime limitations or
out-of-scope features (listed below), not unfinished work.

These were checked against the Apple `container` v1.0.0 runtime through the
integration lane (`make test-integration`). One limitation came out of that:
`container inspect` reports only `state` ("stopped"), never a process exit code,
so `up --abort-on-container-failure` and `--exit-code-from` fall back to 0 with a
warning. `--abort-on-container-exit` works fully.

### Across docker compose versions

fruitbox doesn't use `docker compose` at runtime, so "which version" splits in
two. Compose *file* parsing comes from the vendored `compose-go` library, not the
CLI, so files written for any recent compose era load the same. The CLI *flag*
surface is what can drift between releases, so `make compat-matrix` runs the gap
report against several pinned versions. The gap set is identical across all of
them:

| Command | v5.2.0 | v5.1.4 | v5.0.2 | v2.40.3 |
|---|---|---|---|---|
| `attach` | `detach-keys`, `no-stdin`, `sig-proxy` | same | same | same |
| `build` | `builder`, `check`, `print`, `provenance`, `sbom`, `ssh` | same | same | same |
| `config` | `lock-image-digests`, `resolve-image-digests`, `variables` | same | same | same |
| `cp` | `archive`, `follow-link` | same | same | same |
| `events` | `since`, `until` | same | same | same |
| `exec` | `privileged` | same | same | same |
| `logs` | `since`, `until` | same | same | same |
| `port` | `index` | same | same | same |
| `run` | `use-aliases` | same | same | same |
| `stats` | `all`, `no-trunc` | same | same | same |
| `up` | `menu` | same | same | same |

Every other command is at full flag parity across all four versions. Regenerate
this table with `make compat-matrix` (set `FRUITBOX_MATRIX_VERSIONS` to choose
versions).

### Why each remaining gap stays open

| Command | Gap | Reason |
|---|---|---|
| `attach` | `--detach-keys`/`--no-stdin`/`--sig-proxy` | `container start --attach` exposes no attach-session controls |
| `build` | `--builder`/`--check`/`--print`/`--provenance`/`--sbom`/`--ssh` | BuildKit features Apple's builder doesn't implement |
| `config` | `--resolve-image-digests`/`--lock-image-digests` | require registry access to resolve digests |
| `config` | `--variables` | needs interpolation-variable introspection compose-go doesn't expose |
| `cp` | `--archive`/`--follow-link` | `container cp` has no such flags |
| `events` | `--since`/`--until` | events are synthesized live; there is no historical event store |
| `exec` | `--privileged` | the runtime has no privileged exec |
| `logs` | `--since`/`--until` | `container logs` has no time filter |
| `port` | `--index` | port resolution is from the compose model; replicas share the mapping |
| `run` | `--use-aliases` | `container run` has no network-alias flag |
| `up` | `--menu` | an interactive TUI, out of scope for a non-interactive CLI |
| `stats` | `--all`/`--no-trunc` | `container stats` exposes neither |

## Service-attribute coverage

Translation of the compose service model is tracked in the README. The four
attributes with no runtime equivalent (`privileged`, `devices`, `mac_address`,
`group_add`) emit explicit warnings rather than being silently dropped.
