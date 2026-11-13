#!/usr/bin/env bash
#
# Starts the feed image given as $1 and asserts what can honestly be
# asserted without a real OpenBao and a real Meta access token: that the
# process starts and keeps running, that /healthz answers 200 while
# /readyz answers 503 the whole time (internal/feed.App.checkReady never
# passes without a usable token and a synced feed, see
# charts/ostendorf.tech/README.md's "Health, readiness and metrics"
# section), and that the posts and image routes already answer correctly
# off an empty, unsynced feed.
#
# What this does NOT prove: that the feed service ever reaches /readyz
# 200, that it can actually log in to OpenBao, or that it can actually
# sync from Meta. None of that is reachable in this sandbox, which has no
# OpenBao and no Meta token to give it (see the repository's own
# AGENTS.md on why there is no INSTAGRAM_TOKEN anywhere any more); a pod
# sitting at 0/1 forever, retrying, is this service's own deliberate,
# documented behaviour, not a failure this script can or should wait out.
# Proving OpenBao's Kubernetes auth actually admits this service's role
# needs a real cluster, same as issue #24's own plan names it.
#
# Run by the docker-build-feed job in .github/workflows/checks.yaml right
# after it builds and loads the image, so this never needs a registry.
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <image-ref>" >&2
  exit 1
fi

image="$1"
container="ostendorf-tech-feed-smoke-$$"
base_url="http://127.0.0.1:8081"
ready_timeout=10
fail=0

cleanup() {
  local status=$?
  if [[ $status -ne 0 ]]; then
    echo "---- ${container} logs ----" >&2
    docker logs "$container" 2>&1 || true
    echo "---- end ${container} logs ----" >&2
  fi
  docker rm --force "$container" >/dev/null 2>&1 || true
  exit "$status"
}
trap cleanup EXIT

# OPENBAO_ROLE and OPENBAO_JWT_FILE are required by feedconfig.Load with
# no default, but feed.New performs no IO: the OpenBao login only happens
# lazily, in the background sync loop Start launches (see internal/feed/
# feed.go's own comment), so the server starts and answers HTTP long
# before anything tries to use these. OPENBAO_ADDR is left pointing at a
# host nothing resolves, on purpose: it must stay unreachable, since this
# script exists to prove the service survives exactly that.
docker run \
  --detach \
  --name "$container" \
  --publish "127.0.0.1:8081:8081" \
  --env "OPENBAO_ADDR=https://openbao.invalid:8200" \
  --env "OPENBAO_ROLE=smoke-test" \
  --env "OPENBAO_JWT_FILE=/var/run/secrets/openbao/token" \
  "$image" >/dev/null

# wait_for_health polls /healthz for up to ready_timeout seconds, failing
# immediately once the container itself has exited rather than waiting
# out the full timeout for a process that is already gone. /healthz is
# govue's plain liveness check (health.Registry.Live): it answers 200
# while the process is up, independent of readiness, so it is reachable
# immediately, unlike /readyz.
wait_for_health() {
  local elapsed=0 state exit_code
  while ((elapsed < ready_timeout)); do
    state="$(docker inspect --format '{{.State.Running}}' "$container" 2>/dev/null || echo false)"
    if [[ $state != "true" ]]; then
      exit_code="$(docker inspect --format '{{.State.ExitCode}}' "$container" 2>/dev/null || echo unknown)"
      echo "${container} exited early with code ${exit_code}" >&2
      return 1
    fi
    if curl --fail --silent --show-error --max-time 2 "${base_url}/healthz" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
    elapsed=$((elapsed + 1))
  done
  echo "timed out after ${ready_timeout}s waiting for /healthz" >&2
  return 1
}

if ! wait_for_health; then
  exit 1
fi

# assert_status and assert_contains each record a failure and keep going,
# the same way internal/config aggregates every problem into one report
# instead of stopping at the first one.
assert_status() {
  local path="$1" want="$2" got
  got="$(curl --silent --output /dev/null --write-out '%{http_code}' "${base_url}${path}")" || got=""
  if [[ $got != "$want" ]]; then
    echo "GET ${base_url}${path}: want status ${want}, got \"${got}\"" >&2
    fail=1
  fi
}

assert_contains() {
  local path="$1" needle="$2" body
  body="$(curl --silent "${base_url}${path}")" || body=""
  if [[ $body != *"$needle"* ]]; then
    echo "GET ${base_url}${path}: missing \"${needle}\"" >&2
    fail=1
  fi
}

assert_status "/healthz" "200"

# This is the one assertion the rest of this script exists to make: the
# feed service's readiness contract is to stay unready without a usable
# OpenBao token and a synced feed, never to fall back to "ready enough".
# A 200 here, with no OpenBao and no Meta reachable, would be a bug this
# script must catch, not a pass.
assert_status "/readyz" "503"

# An unsynced feed is a legitimate, empty answer, not an error: the posts
# route answers from whatever internal/instagram.Feed currently holds,
# readiness or not (internal/feed/handlers.go's own comment, proven for
# the empty case by TestHandlePostsEmptyFeed). A 200 with an empty list
# here, rather than a 503 or a 500, is what actually distinguishes "the
# HTTP surface is up and wired correctly" from "the process merely
# started".
assert_status "/api/v1/posts" "200"
assert_contains "/api/v1/posts" '"posts":[]'

# A well formed but unknown post id answers 404 with the one API error
# shape, never a 200 or a 500, proving the id validation and the feed
# lookup both run correctly even against an empty, unsynced feed
# (mirrors TestHandleImageUnknownID).
assert_status "/api/v1/posts/17912345678901234/image" "404"
assert_contains "/api/v1/posts/17912345678901234/image" '"not_found"'

# /readyz must still be 503 at the end: a flap to 200 partway through,
# for example from a readiness flag that defaults true, would have been
# missed by the single check above.
assert_status "/readyz" "503"

if [[ $fail -ne 0 ]]; then
  echo "smoke test failed" >&2
  exit 1
fi

echo "smoke test passed (readiness against a real OpenBao and a real Meta token is not and cannot be proven here)"
