# Contributing to fruitbox

Thanks for your interest! fruitbox is early, so bug reports (especially
`compose.yaml` files that don't work) and PRs are genuinely valuable.

## Getting set up

```bash
git clone https://github.com/urjitbhatia/fruitbox && cd fruitbox
go build ./cmd/fruitbox
go test ./...
```

To exercise the real runtime (macOS on Apple silicon with `container` installed
and `container system start` run):

```bash
make test-integration
```

## How the codebase is laid out

- `internal/compose` — loads compose files via `compose-spec/compose-go` (the
  reference parser). We do **not** reimplement parsing.
- `internal/translate` — pure functions turning the resolved compose model into
  `container` argument vectors. No side effects; easy to unit-test.
- `internal/runner` — the thin shell-out layer, with a `Fake` for tests.
- `internal/engine` — orchestration (up/down/recreate/health/restart/logs/…).
- `internal/cli` — the Cobra command tree.

## Conventions

- **Test-first.** Translation/orchestration changes get a `runner.Fake` test
  (no real runtime). Pure helpers get table tests.
- **Flag changes update the ratchet.** `internal/cli/flag_parity_test.go` holds
  `knownFlagGaps`, the machine-checked record of which `docker compose` flags
  fruitbox doesn't implement. Implementing a flag means removing it from that
  map; the test fails until you do, so a gap can never silently reopen or close.
- **Be honest about runtime gaps.** If Apple's `container` can't express
  something, don't ship a flag that quietly diverges — leave it in `knownFlagGaps`
  with a reason in `COMPATIBILITY.md`, or emit a `WARNING`. See the existing
  `translate.UnsupportedWarnings` for the pattern.
- **Keep it clean.** `gofmt -w` and `go vet ./...` must pass. Stage files by
  name (other work may be in flight).

## Regenerating the compatibility audit

With a local `docker compose` installed:

```bash
make compat        # diff fruitbox's flag surface against real docker compose
```

This is the source of truth behind `COMPATIBILITY.md` and the `TestFlagParity`
ratchet.

## Opening a PR

1. Branch from `main`.
2. Make the change with tests.
3. `go test ./...` green, `gofmt`/`go vet` clean.
4. Describe what changed and, if it touches compatibility, what the ratchet now
   shows.
