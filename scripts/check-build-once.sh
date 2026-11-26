#!/usr/bin/env bash
#
# Asserts that one task run starts web:build and prerender exactly once, for every
# target that needs the frontend. The frontend is reachable by several paths (a target
# lists web:build and prerender, and prerender lists web:build again), and Task starts a
# task once per path unless it carries run: once. Two vite builds on cmd/frontend/dist at
# the same time fail with ENOTEMPTY in vite:prepare-out-dir (issue #137).
#
# Reads the output of task --dry, which prints one line per command Task would start and
# applies the run: once rule exactly as a real run does.
#
# Usage: check-build-once.sh [Taskfile]    (default: Taskfile.yaml next to scripts/)
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
taskfile="${1:-${root}/Taskfile.yaml}"
targets=(lint:vet lint:golangci build test:go test:routes check)
fail=0

for target in "${targets[@]}"; do
  out="$(task --trusted-hosts github.com --taskfile "$taskfile" --dry "$target" 2>&1)"
  for step in web:build prerender; do
    count="$(grep -c -F "task: [${step}] " <<<"$out" || true)"
    if [[ "$count" -ne 1 ]]; then
      echo "FAIL: task ${target} starts ${step} ${count} times, want exactly 1" >&2
      fail=1
    fi
  done
done

if [[ $fail -ne 0 ]]; then
  exit 1
fi
echo "ok: ${targets[*]} each start web:build and prerender once"
