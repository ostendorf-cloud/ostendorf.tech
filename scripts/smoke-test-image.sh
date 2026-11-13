#!/usr/bin/env bash
#
# Starts the container image given as $1 and asserts it actually serves
# the site, the way the Helm chart would run it: health, readiness and
# metrics on the admin listener, the home page in each locale with its
# prerendered content intact and robots.txt/sitemap.xml on the public
# one, and that the public listener itself no longer answers /metrics.
# Run by the docker-build job in .github/workflows/checks.yaml right
# after it builds and loads the image, so this never needs a registry.
#
# CSRF_KEY is generated here with openssl, exactly the way
# docker-compose.yaml's own comment documents; internal/config refuses
# to start the application without one, and this value is thrown away
# with the container.
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <image-ref>" >&2
  exit 1
fi

image="$1"
container="ostendorf-tech-smoke-$$"
base_url="http://127.0.0.1:8080"
admin_url="http://127.0.0.1:8081"
ready_timeout=30
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

csrf_key="$(openssl rand -hex 32)"

# ALLOW_PLAIN_HTTP matches docker-compose.yaml's own local run: this
# script talks to the container directly over plain HTTP, with nothing
# terminating TLS in front of it, unlike the chart's own default.
docker run \
  --detach \
  --name "$container" \
  --publish "127.0.0.1:8080:8080" \
  --publish "127.0.0.1:8081:8081" \
  --env "CSRF_KEY=${csrf_key}" \
  --env "ALLOW_PLAIN_HTTP=true" \
  "$image" >/dev/null

# wait_for_health polls /healthz on the admin listener for up to
# ready_timeout seconds, failing immediately once the container itself
# has exited rather than waiting out the full timeout for a process that
# is already gone.
wait_for_health() {
  local elapsed=0 state exit_code
  while ((elapsed < ready_timeout)); do
    state="$(docker inspect --format '{{.State.Running}}' "$container" 2>/dev/null || echo false)"
    if [[ $state != "true" ]]; then
      exit_code="$(docker inspect --format '{{.State.ExitCode}}' "$container" 2>/dev/null || echo unknown)"
      echo "${container} exited early with code ${exit_code}" >&2
      return 1
    fi
    if curl --fail --silent --show-error --max-time 2 "${admin_url}/healthz" >/dev/null 2>&1; then
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

# assert_status and assert_contains each record a failure and keep
# going, the same way internal/config aggregates every problem into one
# report instead of stopping at the first one. Both default to base_url,
# the public listener; a third argument points at a different one, the
# admin listener's admin_url.
assert_status() {
  local path="$1" want="$2" url="${3:-$base_url}" got
  got="$(curl --silent --output /dev/null --write-out '%{http_code}' "${url}${path}")" || got=""
  if [[ $got != "$want" ]]; then
    echo "GET ${url}${path}: want status ${want}, got \"${got}\"" >&2
    fail=1
  fi
}

assert_contains() {
  local path="$1" needle="$2" url="${3:-$base_url}" body
  body="$(curl --silent "${url}${path}")" || body=""
  if [[ $body != *"$needle"* ]]; then
    echo "GET ${url}${path}: missing \"${needle}\"" >&2
    fail=1
  fi
}

assert_status "/healthz" "200" "$admin_url"
assert_status "/readyz" "200" "$admin_url"
assert_status "/metrics" "200" "$admin_url"

# The public listener moved these three to the admin listener above: it
# must no longer answer /metrics itself. pageHandler's catch-all answers
# every unknown path with the locale's own 404 page (internal/webapp's
# own TestAppPublicHandlerServesNoOperationalEndpoints proves the same
# thing at the Go level), which is what a request for /metrics on the
# public listener now actually is.
assert_status "/metrics" "404"

# id="hero-title" is structural markup on the home page's hero section
# (web/src/views/HomeView.vue), not copy: it exists only in the real,
# server-rendered component tree internal/webapp/pages.go serves from
# cmd/website/prerendered, never in the empty "<div id=\"app\"></div>"
# SPA shell web/index.html ships. Asserting on it, rather than on a 200
# alone, is what actually proves the prerendered HTML survived into this
# image instead of an empty shell that would also answer 200.
assert_status "/de/" "200"
assert_contains "/de/" 'id="hero-title"'
assert_status "/en/" "200"
assert_contains "/en/" 'id="hero-title"'

assert_status "/robots.txt" "200"
assert_status "/sitemap.xml" "200"

if [[ $fail -ne 0 ]]; then
  echo "smoke test failed" >&2
  exit 1
fi

echo "smoke test passed"
