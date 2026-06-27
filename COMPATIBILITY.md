# Docker Compose compatibility status

This file tracks fruitbox's CLI compatibility with `docker compose`, **measured**
by diffing against a real `docker compose` install — not asserted. Regenerate it
any time with:

```bash
go install ./cmd/fruitbox
./scripts/compat-audit.sh "$(go env GOPATH)/bin/fruitbox"
```

Baseline reference: `docker compose` v5.0.2 (also cross-checked against
<https://docs.docker.com/reference/cli/docker/compose/>).

## Automated compatibility tests

Two tests in `internal/cli` enforce this against a real `docker compose`
(they `t.Skip` when it isn't installed, so CI stays hermetic):

- **`TestConfigMatchesDockerCompose`** — differential output test. Runs fruitbox
  in-process and `docker compose` as a subprocess over a fixture matrix
  (`--services/--networks/--volumes/--profiles/--images`, profile activation,
  multi-file merge, and full YAML render) and asserts **identical output**. The
  full `config` YAML is byte-for-byte identical because both use compose-go's
  marshaller.
- **`TestFlagParity`** — a *ratchet*. It computes, per command, the docker
  compose flags fruitbox is missing and asserts the set exactly equals the
  `knownFlagGaps` baseline. Closing a gap, regressing one (silently losing a
  flag — like the `-f` crash did), or a docker change all force a visible update.

> **Honesty note:** an earlier revision claimed "full command parity." That was
> wrong — it compared command *names* only. The flag surface was far from
> complete. This document exists so the claim stays grounded in a reproducible
> measurement.

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

Per-command flag gaps are enforced by `TestFlagParity` (the `knownFlagGaps`
baseline) and can be regenerated with `scripts/compat-audit.sh`.

**Progress: 138 → 36 recorded gaps. Every implementable flag is implemented**
— 22 of the 32 commands are at full flag parity, and the remaining 36 gaps are
all genuine runtime limitations or out-of-scope features (enumerated below),
not unfinished work.

> Validated against the real Apple `container` v1.0.0 runtime (see the
> integration lane, `make test-integration`). One discovered limitation:
> `container inspect` reports only `state` ("stopped"), never a process exit
> code, so `up --abort-on-container-failure` / `--exit-code-from` fall back to
> 0 with a warning. `--abort-on-container-exit` works fully.

### Why each remaining gap can't be closed

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
