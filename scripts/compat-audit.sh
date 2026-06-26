#!/usr/bin/env bash
# compat-audit.sh — differential audit of fruitbox vs. real `docker compose`.
#
# Compares the command list and per-command flag surface so the compatibility
# claim is measured, not asserted. Requires `docker compose` and a built
# fruitbox binary (pass its path as $1, default: `fruitbox` on PATH).
set -euo pipefail

FB="${1:-fruitbox}"
DC_CMDS=$(docker compose --help 2>&1 | sed -n '/Commands:/,/^$/p' | grep -E '^  [a-z]' | awk '{print $1}')
FB_CMDS=$($FB --help 2>&1 | sed -n '/Available Commands/,/^Flags/p' \
  | grep -vE 'Available|Flags|completion|help|^$' | awk '{print $1}')

echo "## Commands docker compose has that fruitbox lacks"
comm -23 <(echo "$DC_CMDS" | sort) <(echo "$FB_CMDS" | sort) | sed 's/^/  - /'
echo

flags() { # tool, command -> sorted unique long flags
  "$@" --help 2>/dev/null | grep -oE -- '--[a-z][a-z0-9-]+' | sort -u
}

# Global flags are shared and excluded from per-command gap counts.
GLOBALS=$(docker compose --help 2>&1 | grep -oE -- '--[a-z][a-z0-9-]+' | sort -u)

echo "## Per-command flag gaps (docker compose has, fruitbox lacks)"
total=0
for c in $(comm -12 <(echo "$DC_CMDS" | sort) <(echo "$FB_CMDS" | sort)); do
  gap=$(comm -23 \
    <(docker compose "$c" --help 2>/dev/null | grep -oE -- '--[a-z][a-z0-9-]+' | sort -u) \
    <($FB "$c" --help 2>/dev/null | grep -oE -- '--[a-z][a-z0-9-]+' | sort -u) \
    | comm -23 - <(echo "$GLOBALS"))
  n=$(echo "$gap" | grep -c . || true)
  if [ "$n" -gt 0 ]; then
    total=$((total + n))
    printf '  %-10s (%2d): %s\n' "$c" "$n" "$(echo "$gap" | tr '\n' ' ')"
  fi
done
echo
echo "Total flag gaps on shared commands: $total"
