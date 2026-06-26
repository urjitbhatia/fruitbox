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

Per-command flag gaps are enumerated by `scripts/compat-audit.sh`. As of the
last run, **128 flag gaps** remain across shared commands (down from 138). The
largest are `up` (25), `run` (18), `build` (13), `logs` (8), `create` (8),
`ps` (7).

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
