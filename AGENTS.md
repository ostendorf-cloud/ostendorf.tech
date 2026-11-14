# ostendorf.tech

The business website of Simon Ostendorf Veranstaltungstechnik.

The global baseline already applies here and is not repeated below.

## Layout

- `cmd/website` wires the service: config, construction, shutdown. `embed.go` embeds
  `dist/` (the built SPA, written by `task web:build`) and `prerendered/` (the static
  HTML per route and locale, written by `task prerender`). Both directories sit under
  `cmd/` rather than `internal/` because `go:embed` resolves relative to the file
  holding the directive, the same reason old_simon-ostendorf.de gave for its own
  `templates` and `static`.
- `internal/config` loads and validates the website's own environment once, aggregated
  into one error. Every secret-shaped variable also accepts a `_FILE` twin; see the root
  `README.md`'s Configuration table. The website itself has no secret manager client and
  no Instagram or Meta credential of its own any more: see `internal/instagram`'s own
  bullet below for where both moved to.
- `internal/routes` is the website's one route table: `routes.json`, checked in and
  embedded through `//go:embed`, lists every page as a name, a slug and the prerendered
  file that answers it. `internal/webapp`, `web/scripts/prerender.ts`, and the frontend's
  own `web/src/router/index.ts` and `web/src/lib/seo.ts`, through the shared
  `web/src/lib/routeTable.ts`, each read this one file to build their own list instead
  of restating it, so a page added to one side and not the other can no longer happen;
  see `internal/webapp`'s own bullet below for what proves that in practice. `routes.go`'s
  `Parse` rejects a table with a missing name or file, or a repeated name, slug or file,
  before any of the five consumers above ever sees it; `routeTable.ts`'s own
  `validateRouteTable` enforces the exact same three rules for the frontend, which reads
  `routes.json` through Vite's JSON import rather than through Go, so it has no Go
  validation to inherit. `internal/routes/testdata/{valid,invalid}` holds the fixtures
  both `routes_test.go` and `routeTable.test.ts` read, so a fixture either side adds is
  checked by both validators without touching the other test file.
- `internal/pow` issues and verifies the contact form's proof of work challenge, ported
  from old_simon-ostendorf.de's `internal/pow` unchanged: the algorithm is a contract
  with the browser worker that solves it.
- `internal/contact` validates a contact submission, builds the mail and sends it over
  SMTP, ported from old_simon-ostendorf.de's `internal/contact` with one change: a field
  error carries a stable code instead of a German message, so `internal/webapp` can
  answer in German or English depending on the visitor's own locale.
- `internal/instagram` is the domain logic behind the home page's Impressionen section:
  a `Client` against Meta's Graph API, a `Feed` cache of the posts and their picture
  bytes, a `Syncer` that keeps `Feed` current, and a `TokenStore` that reads and writes
  the access token through `internal/openbao`'s KV v2 API under optimistic concurrency
  (CAS), refreshing it against Meta once it passes `TokenRefreshAge` and treating a CAS
  mismatch as the other side's write winning rather than an error. None of this runs
  inside the website any more: `cmd/feed` is the only binary that imports this package,
  the only one with an OpenBao client, and the only one that ever holds the access
  token, exclusively in memory, never logged and never returned in an error. Because a
  refreshed token is written back to OpenBao rather than kept only in memory, it survives
  a restart: the manual rotation this needed before Meta's 60-day expiry is gone, not
  merely less frequent. Ported from old_simon-ostendorf.de's `internal/instagram`, with
  the same one change as before this split: where a refreshed token is stored, OpenBao
  rather than nowhere.
- `internal/openbao` is a small, standard library only HTTP client against OpenBao's
  Kubernetes auth method and its KV v2 API, ported unchanged from
  old_simon-ostendorf.de's own `internal/openbao`. It reads the projected service account
  token fresh on every login rather than caching it, because the kubelet rotates that
  file; it re-logs in at two thirds of the lease instead of waiting for `renew-self` to
  fail, and retries a login once on a 403. `Write` is a full-document overwrite guarded
  by the version `Read` last saw, so two replicas writing at once cannot silently clobber
  each other. `cmd/feed` is the only binary that constructs one of these: the website has
  no OpenBao client at all.
- `cmd/feed` wires the feed service: a second binary alongside `cmd/website`, with its
  own configuration (`internal/feedconfig`) and its own chart entry
  (`charts/ostendorf.tech/templates/deployment-feed.yaml` and friends). `internal/feed`
  builds it: the background loop that keeps `internal/instagram`'s `Feed` current from
  Meta using the OpenBao-backed token above, and the internal-only HTTP surface the
  website reads from, `GET /api/v1/posts` and `GET /api/v1/posts/{id}/image`, alongside
  `/healthz` and `/readyz` on the one listener. `/healthz` always answers 200 while the
  process is up; `/readyz` answers 503 until one full sync cycle has obtained a usable
  token and completed a call to Meta, and then stays 200 even through a later transient
  failure, since the last good feed keeps being served regardless. A cycle that reaches
  Meta and finds zero posts still counts: an Instagram account with nothing posted is a
  real, permanently empty feed, not a readiness failure. A feed pod that cannot reach
  OpenBao or Meta therefore sits at `0/1` ready, retrying, rather than crash-looping or
  serving stale or absent data as if nothing were wrong. Its HTTP surface authenticates
  no caller of its own; it relies on the chart's `NetworkPolicy` admitting only the
  website's pods instead, see `charts/ostendorf.tech/README.md`.
- `internal/feedclient` is the website's own client against the feed service above: a
  `Client` for its two routes, a `Cache` of the posts and picture bytes it last got back,
  and a `Syncer` that polls the feed service on an interval (`FEED_REFRESH_INTERVAL`) and
  keeps that `Cache` current, the same caching `internal/instagram` itself used to do
  before this split. Every call is bounded by a timeout, so an unreachable or slow feed
  service can never hang a page; an empty `Cache`, whether from an unreachable feed
  service, an unset `FEED_SERVICE_URL`, or a feed service with nothing to show, renders no
  Impressionen section at all, exactly as an absent token did before this split.
  `internal/webapp/feed.go` is the one place this package is read from.
- `internal/analytics` reports page views to Umami from the server, ported from
  old_simon-ostendorf.de's `internal/analytics` unchanged: no script, no cookie, nothing
  read from the visitor's device, which again is what the privacy policy states. It
  counts nothing unless `UMAMI_URL` and `UMAMI_WEBSITE_ID` are both set, leaves out a
  prefetch, Do Not Track, Global Privacy Control and any client that is not a public
  address, and drops a view rather than letting a slow Umami hold a response.
  `internal/webapp`'s `trackPageView` is the one place it is called from, after a page was
  actually served; the title it reports is read once at startup out of the prerendered
  HTML's own `<title>`, so Go never carries a second copy of the i18n catalogue.
- `internal/webapp` builds two HTTP handlers from govue's packages. `Handler` is the
  public one: the security and access log middleware, the client IP resolver, the rate
  limiter, a session manager and the CSRF protector, in front of pages, assets and the
  API routes. `AdminHandler` is the second, smaller one govue's admin listener serves:
  health, readiness and metrics, reached only from inside the cluster, never through the
  chart's own Ingress. `pages.go` is the catch-all document handler:
  the locale redirect at `/`, trailing slash canonicalisation, and serving the matching
  prerendered file or a locale's own 404 page. `pages.go`'s own `routeFiles` is built
  from `internal/routes`' embedded table, not hand written, so the page handler, the
  per-page mux registration and `sitemap.go`'s `/robots.txt`/`/sitemap.xml` (which reads
  that same `routeFiles`/`locales`) all answer from one list. That a page added to
  `routes.json` actually gets prerendered, rather than merely agreeing with Go's own
  copy of a list that no longer exists, is what `cmd/website`'s
  `TestPrerenderedOutputMatchesRouteTable` checks: it walks the real output `task
  prerender` wrote and compares it against `internal/routes`, catching both a route with
  no prerendered file and a prerendered file that answers no route. `task test:routes`
  runs it, with its own CI job. `contact.go` and `session.go` are the contact form's two
  endpoints and the anonymous, cookie-backed session they share with the CSRF
  protector. A session is minted in exactly one place: `GET /api/v1/challenge`, which
  already returns the resulting CSRF token in the same response body the frontend reads.
  No page route mints or reads a session any more. `POST /api/v1/contact` requires that
  session through `requireSession` instead, refusing a request with none outright.
  `session.go` keeps that session in govue's in-process `session.MemoryStore`, so the
  Deployment stays at one replica: a second replica, or two pods briefly coexisting
  during an ordinary rolling update, can answer a visitor's GET on one pod and their
  contact form POST on another pod that never saw that session. `requireSession` then
  refuses that POST with 403 — correctly from that pod's point of view, wrongly from the
  visitor's, since their session is real, just on the other pod. This is an availability
  problem for real visitors, not a security hole. `charts/ostendorf.tech` enforces the
  single replica at render time and runs the Deployment with `strategy.type: Recreate`,
  so a rollout never lets two pods coexist either. Lifts once `internal/webapp` moves to
  a shared session store instead of the in-memory one.
- `web/` is the Vue SPA, consuming govue's Vue package as a pnpm git dependency. `web/src`
  holds the app; `web/src/lib/contactChallenge.ts` and `web/src/workers/pow.worker.ts`
  are the contact form's proof of work client, a Web Worker kept cookie-free and
  third-party-free; `web/scripts/prerender.ts` is the static site generation step `task
  prerender` runs. `web/src/lib/routeTable.ts` is the one place the frontend reads
  `internal/routes/routes.json` from; `web/scripts/prerender.ts`, `web/src/router/index.ts`
  (the Vue Router table) and `web/src/lib/seo.ts` (the `PageRoute` list and its slugs) all
  import it instead of restating the page list themselves.
- `charts/ostendorf.tech/` is the Helm chart, for both the website's and the feed
  service's own workloads. `scripts/validate-chart.sh` renders it for its default values
  and every documented non-default combination, and for a must-fail set proving every
  render-time `fail` and `required` guard actually rejects the values it is meant to
  reject, shared by `task lint:helm` and `task validate:kubeconform` so CI and a local
  run check the exact same combinations; see `charts/ostendorf.tech/README.md`'s own
  "Verification" section. `Dockerfile` is the multi-stage build for both of their
  container images, as two separate final stages, `website` and `feed`
  (`docker build --target website` or `--target feed`) rather than two Dockerfiles, since
  the two binaries share `go.mod`/`go.sum` and the pinned base image digests; see the
  comment at the top of `Dockerfile` itself for why a build of one target never triggers
  the other's own stages. `docker-compose.yaml` is a local development stack for the
  website alone.

## A build needs the frontend first

`cmd/website` cannot compile against an empty `dist/` or `prerendered/` directory,
because `go:embed` fails when its pattern matches no file at all. `dist/` and
`prerendered/` are both gitignored, generated by `task web:build` and `task prerender`
respectively.

`task build` and `task test:go` both declare that dependency, so running either one
through Task always populates the frontend first. A bare `go build ./cmd/website` or
`go test ./cmd/website/...` outside Task fails on a fresh checkout for that reason; run
`task build` or `task test:go` instead, or `task web:build && task prerender` once by
hand before reaching for the Go toolchain directly.

`cmd/feed` has no `go:embed` directive anywhere in its own tree and needs none of this:
`task build:feed` declares no frontend dependency, on purpose, and `go build ./cmd/feed`
compiles on a fresh checkout with no prior step at all.

## govue's two halves move together

govue releases a Go module and a Vue package under one tag. `go.mod`'s
`require github.com/ostendorf-cloud/govue` and `web/package.json`'s
`@ostendorf-cloud/govue` pnpm git dependency must always name the same tag: a mismatch
means the repository takes half of a govue release and not the other half, silently.
`task check:govue-version` (`internal/depsync`) enforces this: it asks `go list -m` for
the version go.mod actually resolves to, the same resolution `go build` itself uses, and
pulls the tag out of package.json's `ostendorf-cloud/govue#<tag>&path:/web` specifier,
then fails naming both versions and both files when they differ. `web/pnpm-lock.yaml`'s
own specifier field already has to match package.json's for `pnpm install
--frozen-lockfile` to pass, and its resolved entry pins a commit hash, not a tag, so it
cannot be compared to a version directly either. That match is not enough on its own,
though: pnpm's own git resolver has written a resolution object for this dependency with
no `path` field at all, even from a fully fresh resolve against a cold store, which
makes `pnpm install --frozen-lockfile` install the whole govue repository instead of its
`web/` subdirectory while every specifier still agrees (pnpm/pnpm#12304, fixed by
pnpm/pnpm#12344, not backported to the pnpm 10 release line as far as this repository
could establish). `check:govue-version` therefore also reads `web/pnpm-lock.yaml`
directly and fails when that dependency's resolution carries no `path: /web`.

## Verify before committing

```sh
task check
```

`task --list` shows the individual targets, including the small ones `task check`
aggregates.

## Generated files

`web/src/assets/logo-wordmark-light.svg` and `logo-wordmark-dark.svg` are generated
from `Logo_Name.pdf` in the organisation's `old_logo` repository, never hand edited.
The source file bakes in two colours (white for "Simon Ostendorf", a mid blue for
"Veranstaltungstechnik"), meant for the dark stage background the old site always
rendered on; both are overridden to one flat colour per variant, navy
(`#1b1030`, this theme's own `--color-text`) for light mode and white for dark mode,
matching the two variants the design calls for. The command:

```sh
pdftocairo -svg Logo_Name.pdf logo-wordmark-light.svg
# then replace every fill="rgb(...)" with fill="#1b1030" (light) or fill="#ffffff" (dark)
svgo --multipass -i logo-wordmark-light.svg -o logo-wordmark-light.svg
```

repeated once per variant. The fader mark alone (the icon, `LogoMark.vue`) is not
generated: it is inline Vue markup ported from old_simon-ostendorf.de's `{{define
"logo"}}` template block, recoloured through a gradient whose stops read this theme's
own `--color-accent` and `--color-accent-text` tokens, so it never needs regenerating
when the theme changes.

## Scope of this milestone

The home page's hero, Leistungen, Impressionen, Bands, Über mich and Kontakt sections,
the imprint, privacy and AGB pages, crawling (`robots.txt`, `sitemap.xml`), the Instagram
feed (split across the website and the feed service, see `internal/instagram`'s own
bullet above), the Umami hookup, both container images and the Helm chart are
implemented. Left out, tracked as later milestones:

- The Impressionen section is rendered by the browser after the page loads, from
  `GET /api/v1/instagram`, not baked into the prerendered HTML: the feed changes while
  the binary does not, and `task prerender` runs at build time with no access token. A
  crawler that executes no JavaScript therefore does not see the section. That is
  deliberate, since the section carries no content worth indexing, but it does mean the
  nav has no Impressionen entry either, because a prerendered nav cannot know whether the
  section will be there.

## Deviations

- `LICENSE` is proprietary, all rights reserved, not the baseline's Apache-2.0: this is
  one business's own website, and reusing it would misidentify the source of a service.
  No per-file licence header and no reusable license-header workflow for the same
  reason; the single `LICENSE` carries it. `NOTICE` records what `LICENSE` does not
  cover: the dependencies and their own terms, the brand and imagery, the page texts
  including the band descriptions that belong to those bands, and why the legal pages
  must not be copied.
- `.github/workflows/dco.yaml` calls `ostendorf-cloud/workflows`'s reusable workflow at
  `v0.1.1`, the latest tag that exists, not a floating `v0` major tag:
  `ostendorf-cloud/workflows` has not cut one yet. Repoint at `v0` once it exists.
- No CodeQL workflow. Code scanning on a private repository needs GitHub Advanced
  Security, which this account does not have.
- The Helm chart lives at `charts/ostendorf.tech/`, matching the repository's own name,
  but its `Chart.yaml` `name` and every rendered object name use `ostendorf-tech`
  instead: a literal dot is invalid in a `Service`'s `metadata.name` (RFC 1035 label),
  which a chart named `ostendorf.tech` would otherwise produce by default.
- `helm-unittest` is pinned in `mise.toml` through mise's generic `github:` backend with
  a `postinstall` hook, not a plain registry entry: it is not in the mise registry, and
  `helm plugin install <url>` is forbidden by the baseline regardless of Helm version.
