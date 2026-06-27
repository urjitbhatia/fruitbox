#!/usr/bin/env bash
#
# compat-matrix.sh — measure fruitbox's CLI flag parity against several pinned
# docker compose versions, and render a markdown table.
#
# fruitbox does not depend on docker compose at runtime; it is used here only as
# a reference oracle. `docker compose <cmd> --help` and `config` run offline (no
# daemon), so we can diff against any released binary.
#
# Usage:
#   scripts/compat-matrix.sh                 # default version set
#   FRUITBOX_MATRIX_VERSIONS="v5.2.0 v5.0.2 v2.40.3" scripts/compat-matrix.sh
#
# Requires: gh (to download release binaries), go, python3.
set -euo pipefail

cd "$(dirname "$0")/.."

VERSIONS=(${FRUITBOX_MATRIX_VERSIONS:-v5.2.0 v5.1.4 v5.0.2 v2.40.3})

# Map host platform to docker/compose asset naming.
os=$(uname -s)
arch=$(uname -m)
case "$os" in
  Darwin) plat=darwin ;;
  Linux)  plat=linux ;;
  *) echo "unsupported OS: $os" >&2; exit 1 ;;
esac
case "$arch" in
  arm64|aarch64) marc=aarch64 ;;
  x86_64|amd64)  marc=x86_64 ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac
asset="docker-compose-${plat}-${marc}"

cache="${TMPDIR:-/tmp}/fruitbox-compose-bins"
mkdir -p "$cache"

report_dir=$(mktemp -d)
trap 'rm -rf "$report_dir"' EXIT

echo "Building flag-gap report against ${#VERSIONS[@]} compose versions ($asset)..." >&2

for v in "${VERSIONS[@]}"; do
  bin="$cache/docker-compose-$v"
  if [ ! -x "$bin" ]; then
    echo "  downloading $v..." >&2
    gh release download "$v" --repo docker/compose --pattern "$asset" --output "$bin" --clobber
    chmod +x "$bin"
  fi
  if ! "$bin" version >/dev/null 2>&1; then
    echo "  WARN: $v binary failed to run; skipping" >&2
    continue
  fi
  echo "  testing $v..." >&2
  FRUITBOX_COMPAT=1 FRUITBOX_COMPOSE_BIN="$bin" \
    go test ./internal/cli/ -run '^TestFlagGapReport$' -v -count=1 2>&1 \
    | grep -oE 'GAP\|[a-z]+\|[a-z,-]*' > "$report_dir/$v.txt" || true
done

# Render the markdown table.
python3 - "$report_dir" "${VERSIONS[@]}" <<'PY'
import os, sys
report_dir = sys.argv[1]
versions = sys.argv[2:]

# data[version][command] = sorted list of missing flags
data = {}
commands = set()
for v in versions:
    path = os.path.join(report_dir, f"{v}.txt")
    if not os.path.exists(path):
        continue
    data[v] = {}
    for line in open(path):
        line = line.strip()
        if not line.startswith("GAP|"):
            continue
        _, cmd, flags = line.split("|", 2)
        gaps = [f for f in flags.split(",") if f]
        data[v][cmd] = gaps
        commands.add(cmd)

versions = [v for v in versions if v in data]
if not versions:
    print("no data collected", file=sys.stderr); sys.exit(1)

# Only show commands that have a gap in at least one version.
def cell(v, cmd):
    gaps = data[v].get(cmd)
    if gaps is None:
        return "—"        # command absent in this compose version
    if not gaps:
        return "✓"        # full parity
    return ", ".join(f"`{g}`" for g in gaps)

rows = sorted(c for c in commands if any(data[v].get(c) for v in versions))

header = "| Command | " + " | ".join(versions) + " |"
sep = "|" + "---|" * (len(versions) + 1)
print(header)
print(sep)
for cmd in rows:
    print(f"| `{cmd}` | " + " | ".join(cell(v, cmd) for v in versions) + " |")

# Verdict: is the gap set identical across all tested versions?
def gapset(v):
    return {c: tuple(data[v].get(c, [])) for c in commands}
identical = all(gapset(v) == gapset(versions[0]) for v in versions)
print()
total = sum(len(data[versions[0]].get(c, [])) for c in commands)
if identical:
    print(f"Gap set is **identical** across {', '.join(versions)} "
          f"({total} unimplementable flags, all documented).")
else:
    print(f"Gap set **differs** across versions — see the table. "
          f"({versions[0]} has {total} gaps.)")
PY
