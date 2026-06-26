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

Implemented (29): attach, build, config, cp, create, down, events, exec, images,
kill, logs, ls, pause, port, ps, pull, push, restart, rm, run, scale, start,
stop, top, unpause, up, version, wait, watch.

**Not yet implemented (6):**

| Command | Notes |
|---|---|
| `volumes` | list project volumes — straightforward, planned next |
| `stats` | live resource usage — maps to `container stats` |
| `export` | export a container filesystem — maps to `container export` |
| `commit` | create an image from a container — Apple `container` has no commit |
| `publish` | publish a compose app to an OCI registry — compose-specific, complex |
| `bridge` | convert compose to Kubernetes/other — Docker Desktop feature, N/A |

## Flag coverage

Per-command flag gaps are enforced by `TestFlagParity` (the `knownFlagGaps`
baseline) and can be regenerated with `scripts/compat-audit.sh`.

**Progress: 138 → 73 recorded gaps (16 of 29 commands at full flag parity).**
Fully closed: `version`, `down`, `stop`, `restart`, `kill`, `images`, `scale`,
`push`, `start`, `wait`, `rm`, `ls`, `ps`. Substantially closed: `run` (18→4),
`up` (25→21), `exec` (4→1), `pull` (5→1), `create` (8→5), `build` (13→10),
`logs` (7→5), `events` (3→2), `cp` (3→2), `config`*.

(*config minus digest-pinning/`--variables`, which need registry access.)

Largest remaining gaps: `up` (21 — recreate semantics, foreground attach/log
formatting), `build` (10 — BuildKit features: `--sbom`/`--provenance`/
`--builder`/`--check`/`--push`), `logs` (5 — time filters & log formatting),
`create` (5 — recreate semantics), `run` (4), `config` (4). These are mostly
features the Apple `container` runtime doesn't expose (recreate diffing, log
multiplexing, BuildKit attestations) or that need registry access.

### Priority order for closing flag gaps

1. **Observed-behavior flags** (change what containers do): `up` (`--build`,
   `--force-recreate`, `--no-recreate`, `--no-start`, `--wait`, `--timeout`,
   `--pull`, `--no-deps`, `--abort-on-container-exit`, `--exit-code-from`),
   `run` (`--entrypoint`, `--user`, `--volume`, `--publish`, `--label`,
   `--service-ports`, `--no-deps`, `--build`), `down` (`--timeout`,
   `--remove-orphans`, `--rmi`), `rm --volumes`, `exec` (`--index`, `--detach`).
2. **Output/formatting flags**: `logs` (`--tail`, `--since`, `--until`,
   `--timestamps`, `--no-color`), `ps` (`--format`, `--filter`, `--status`,
   `--all`), `images --format`, `version` (`--format`, `--short`),
   `events` (`--json`, `--since`, `--until`).
3. **Build/registry flags** (some require BuildKit/registry features Apple
   `container` may not expose): `build` (`--build-arg`, `--no-cache`, `--pull`,
   `--push`, `--target`, `--ssh`), `pull`/`push` policy & failure flags.
4. **Likely-unmappable** (documented, will warn): `build --provenance/--sbom`,
   `up --menu`, `attach --sig-proxy`, digest-pinning in `config`.

## Service-attribute coverage

Translation of the compose service model is tracked in the README. The four
attributes with no runtime equivalent (`privileged`, `devices`, `mac_address`,
`group_add`) emit explicit warnings rather than being silently dropped.
