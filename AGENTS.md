# ostendorf.tech

The business website of Simon Ostendorf Veranstaltungstechnik.

The global baseline already applies here and is not repeated below.

## Layout

The tree and the root `README.md` show the directories. Path specific knowledge lives in
nested `AGENTS.md` files that load when a file below their directory is read.

- The site runs as three images, released under one tag and deployed together:
  frontend (`cmd/frontend`), API (`cmd/api`) and feed (`cmd/feed`). The Ingress routes
  `/api` to the API and every other path to the frontend. See `docs/images-and-compose.md`.
- `internal/config` loads and validates the API's environment once, aggregated into one
  error, `internal/frontendconfig` the frontend's. Every secret-shaped variable of the
  API also accepts a `_FILE` twin. The frontend reads no secret. See the root `README.md`'s
  Configuration table.
- `cmd/frontend/embed.go` embeds `dist/`, the built SPA with the prerender written into
  it (`task web:build`, then `task prerender`).

## Nested files

- `cmd/frontend/AGENTS.md`: govue's frontend, legal aliases, dev proxy, Umami counting.
- `cmd/api/AGENTS.md`: the API handlers and why the API stays at one replica.
- `cmd/feed/AGENTS.md`: the feed service, its OpenBao variables and readiness.
- `internal/feedclient/AGENTS.md`: the API's cache of the feed service.
- `internal/instagram/AGENTS.md`: Graph API client, token store, paging bounds.
- `internal/openbao/AGENTS.md`: the OpenBao KV v2 client and its login methods.
- `internal/routes/AGENTS.md`: the one route table and its two validators.
- `internal/webapp/AGENTS.md`: the contact form wiring and the feed route.
- `web/AGENTS.md`: the Vue SPA, header, contact section and generated files.
- `charts/AGENTS.md`: the Helm chart and its render checks.

## Docs

- `docs/images-and-compose.md`: three images, Dockerfile targets, publishing, compose.
- `docs/govue-and-pnpm.md`: govue's two halves, Renovate setup, the three pnpm pins.
- `docs/generated-assets.md`: sources and commands for logos, photos, icons and bands.
- `docs/milestone-scope.md`: what the first milestone leaves out, the Impressionen nav.

## A build needs the frontend first

`cmd/frontend` cannot compile against an empty `dist/` directory, because `go:embed`
fails when its pattern matches no file at all. `cmd/frontend/dist/` is gitignored,
written by `task web:build` and completed by `task prerender`.

`task build`, `task test:go`, `lint:vet` and `lint:golangci` declare that dependency, so
running one through Task always populates the frontend first. A bare `go build
./cmd/frontend` or `go test ./...` outside Task fails on a fresh checkout for that
reason; run `task web:build && task prerender` once by hand before reaching for the Go
toolchain directly.

`web:build` and `prerender` carry `run: once`: several targets reach them by more than one
path, and two vite builds on `cmd/frontend/dist` at once fail with `ENOTEMPTY`. Keep that
when adding a target that needs the frontend; `task check:build-once` asserts one start of
each per target through `task --dry`, and its target list is in `scripts/check-build-once.sh`.

`cmd/api` and `cmd/feed` embed nothing and need none of this: `task build:api` and `task
build:feed` declare no frontend dependency, on purpose, and `cmd/api`'s own test fails
when it gains a `go:embed` directive.

## Hard constraints

- govue's Go module and Vue package must name the same tag. `task check:govue-version`
  enforces it. See `docs/govue-and-pnpm.md`.
- pnpm is pinned in `mise.toml`, the Dockerfile and `renovate.json`, and moves once.
  See `docs/govue-and-pnpm.md`.
- Generated assets are never hand edited. See `docs/generated-assets.md`.

## Verify before committing

```sh
task check
```

`task --list` shows the individual targets, including the small ones `task check`
aggregates.

## Deviations

- `LICENSE` is proprietary, all rights reserved, not the baseline's Apache-2.0: this is
  one business's own website, and reusing it would misidentify the source of a service.
  No per-file licence header and no reusable license-header workflow for the same
  reason; the single `LICENSE` carries it. `NOTICE` records what `LICENSE` does not
  cover: the dependencies and their own terms, the brand and imagery, the page texts
  including the band descriptions that belong to those bands, and why the legal pages
  must not be copied.
- The Helm chart lives at `charts/ostendorf.tech/`, matching the repository's own name,
  but its `Chart.yaml` `name` and every rendered object name use `ostendorf-tech`
  instead: a literal dot is invalid in a `Service`'s `metadata.name` (RFC 1035 label),
  which a chart named `ostendorf.tech` would otherwise produce by default.
- `helm-unittest` is pinned in `mise.toml` through mise's generic `github:` backend with
  a `postinstall` hook, not a plain registry entry: it is not in the mise registry, and
  `helm plugin install <url>` is forbidden by the baseline regardless of Helm version.
