# ostendorf.tech

The business website of Simon Ostendorf Veranstaltungstechnik.

The global baseline already applies here and is not repeated below.

## Layout

- `cmd/frontend` wires the frontend image, `cmd/api` the API image, `cmd/feed` the feed
  service; see their own bullets below. `cmd/frontend/embed.go` embeds `dist/`, the
  built SPA with the prerender written into it (`task web:build`, then `task
  prerender`). It sits under `cmd/` rather than `internal/` because `go:embed` resolves
  relative to the file holding the directive.
- `internal/config` loads and validates the API's environment once, aggregated into
  one error, `internal/frontendconfig` the frontend's. Every secret-shaped variable of
  the API also accepts a `_FILE` twin; the frontend reads no secret. See the root
  `README.md`'s Configuration table. Neither has a secret manager client or an
  Instagram or Meta credential: see `internal/instagram`'s own bullet below for where
  both moved to.
- `internal/routes` is the site's one route table: `routes.json`, checked in and
  embedded through `//go:embed`, lists every page as a name, a slug and the prerendered
  file that answers it. `web/scripts/prerender.ts`, `web/src/router/index.ts` and
  `web/src/lib/seo.ts`, through the shared `web/src/lib/routeTable.ts`, each read this
  one file instead of restating it, and `cmd/frontend`'s
  `TestPrerenderedOutputMatchesRouteTable` checks the real build against it. `routes.go`'s
  `Parse` rejects a table with a missing name or file, or a repeated name, slug or file,
  before any consumer ever sees it; `routeTable.ts`'s own
  `validateRouteTable` enforces the exact same three rules for the frontend, which reads
  `routes.json` through Vite's JSON import rather than through Go, so it has no Go
  validation to inherit. `internal/routes/testdata/{valid,invalid}` holds the fixtures
  both `routes_test.go` and `routeTable.test.ts` read, so a fixture either side adds is
  checked by both validators without touching the other test file.
- The contact form is govue's: the Go handlers (`contact`, with `pow` and `mail`) and the
  Vue form (`GovueContactForm` with its proof of work worker). The site has no contact,
  proof of work or SMTP code of its own. `internal/webapp/contact.go` only builds govue's
  `contact.Handler` with the site's own envelope (`CONTACT_FROM`, `CONTACT_TO`), subject
  and body, the challenge keys (`CHALLENGE_KEY` first, `CHALLENGE_KEY_PREVIOUS` for a
  rotation), the audience (the host of `SITE_URL`), the site's limiters
  (`CHALLENGE_RATE_LIMIT_*`, `CONTACT_RATE_LIMIT_*`, `MAIL_RATE_LIMIT_*`) and the trusted
  origin of `SITE_URL`, and mounts `GET /api/v1/contact/challenge` and
  `POST /api/v1/contact` behind the general rate limit. `internal/config` reads SMTP
  through govue's `mail.LoadConfig` (`SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME` or
  `SMTP_USERNAME_FILE`, `SMTP_PASSWORD` or `SMTP_PASSWORD_FILE`; the chart mounts both
  credentials as files) and only when `SMTP_HOST` is set. Without a key
  or without `SMTP_HOST` both routes answer 503 `not_configured`. The handler runs with
  `RequireConsent`: the form sends the consent box (`requireConsent` in `HomeView.vue`),
  and a submission without a ticked box is a field error before the proof of work. The
  mail body is a German intro line, govue's `contact.DefaultBody` (name, address,
  message) and one line stating that the browser reported the box as ticked and the
  server's receive time (`Request.Received`), a claim of the client, not a proof. The
  privacy page says so in both languages; change it with the body.
  `internal/webapp/contact_test.go` proves the wiring through the real handler chain with
  a fake `mail.Sender`.
- `internal/instagram` is the domain logic behind the home page's Impressionen section:
  a `Client` against Meta's Graph API, a `Feed` cache of the posts and their picture
  bytes, a `Syncer` that keeps `Feed` current, and a `TokenStore` that reads and writes
  the access token and its refresh time under two configured key names (it has no built-in
  defaults; `internal/feedconfig` supplies them), keeping every other key of the secret,
  through `internal/openbao`'s KV v2 API under optimistic concurrency
  (CAS), refreshing it against Meta once it passes `TokenRefreshAge` and treating a CAS
  mismatch as the other side's write winning rather than an error. None of this runs
  inside the frontend or the API: `cmd/feed` is the only binary that imports this package,
  the only one with an OpenBao client, and the only one that ever holds the access
  token, exclusively in memory, never logged and never returned in an error. Because a
  refreshed token is written back to OpenBao rather than kept only in memory, it survives
  a restart: the manual rotation this needed before Meta's 60-day expiry is gone, not
  merely less frequent. Ported from old_simon-ostendorf.de's `internal/instagram`, with
  the same one change as before this split: where a refreshed token is stored, OpenBao
  rather than nowhere. `Client.Media` pages through every post of the account: it asks for
  `POSTS_PER_REQUEST` posts per request and sends the `after` cursor of each answer back
  as a query parameter on its own `BaseURL`, until Meta's answer has no `paging.next` or
  the `POST_COUNT` cap is reached. Only the presence of `paging.next` is read, never its
  URL, so an answer cannot send the client or the token to another host. Redirects are not
  followed either: every Graph API and picture request goes through `Client.do`, which
  returns a 3xx as `ErrRedirect`, so the picture host list holds for the one URL requested.
  `POST_COUNT` 0 means `MaxPosts`, 10000, and the page bound is also 10000, enough for
  10000 posts at one post per request. The cursor is checked for length and characters and
  never logged; a repeated cursor, a 429, any other 4xx and a cycle past the page bound end
  the cycle with an error kind (`ErrCursorLoop`, `ErrRateLimited`, `ErrRefused`,
  `ErrTooManyPages`). Pages are fetched one after the other
  with `DefaultPagePause` (500 ms) between them. `Syncer` replaces the `Feed` only once
  every page was read, and logs posts, pages, failed images and the duration per cycle.
  `paging_test.go` and `bounds_test.go` prove all of it against a fake media edge,
  including a second server in `paging.next` or behind a redirect that must receive no
  request.
- `internal/openbao` is a small, standard library only HTTP client against OpenBao's KV v2
  API, ported from old_simon-ostendorf.de's own `internal/openbao`. It logs in through a
  `Login` interface that returns the token and its lease. `Kubernetes` is the method for a
  cluster: it reads the projected service account token fresh on every login rather than
  caching it, because the kubelet rotates that file; the client re-logs in at two thirds of
  the lease instead of waiting for `renew-self` to fail, and retries a login once on a 403.
  `StaticToken` is the method for local tests: a token handed to the process, sent as
  `X-Vault-Token`, never renewed, and its 403 is the fixed `ErrForbidden` with no retry and
  no response body. Another method is one more `Login` implementation. Every token is an
  `openbao.Token`, which prints as `[redacted]` under every `fmt` verb, in `slog` and in
  JSON; the one plain string conversion is the request header. `Write` is a full-document
  overwrite guarded by the version `Read` last saw, so two replicas writing at once cannot
  silently clobber each other. `cmd/feed` is the only binary that constructs one of these:
  the frontend and the API have no OpenBao client at all. `internal/secrettest` holds the test helpers
  that prove a secret stays out of every error in an unwrap tree and every formatted value.
- `cmd/feed` wires the feed service: a third binary alongside `cmd/frontend` and
  `cmd/api`, with its own configuration (`internal/feedconfig`) and its own chart entry
  (`charts/ostendorf.tech/templates/deployment-feed.yaml` and friends).
  `OPENBAO_AUTH_METHOD` is `kubernetes` (default, in a cluster) or `token` (local tests,
  `OPENBAO_TOKEN` or its `OPENBAO_TOKEN_FILE` twin); the variables of the other method are
  refused at startup, and no error quotes a refused value. The service never renews a
  static token. `OPENBAO_SECRET_PATH` is the KV v2 data path of the secret, and
  `OPENBAO_SECRET_KEY_ACCESS_TOKEN` and `OPENBAO_SECRET_KEY_REFRESHED_AT` (defaults
  `access_token` and `refreshed_at`) name the two keys inside it; a refresh rewrites only
  those two and keeps every other key. `POST_COUNT` caps the number of posts, 0 means the
  maximum of 10000 (default 200), and `POSTS_PER_REQUEST` sets the page size (default 50,
  1 to 100); every post costs its picture in memory, see "Posts and memory" in
  `charts/ostendorf.tech/README.md`, where the chart's `goMemLimit` and `feed.goMemLimit`
  set `GOMEMLIMIT` below each memory limit. `internal/feed`
  builds it: the background loop that keeps `internal/instagram`'s `Feed` current from
  Meta using the OpenBao-backed token above, and the internal-only HTTP surface the
  API reads from, `GET /api/v1/posts` and `GET /api/v1/posts/{id}/image`, alongside
  `/healthz` and `/readyz` on the one listener. `/healthz` always answers 200 while the
  process is up; `/readyz` answers 503 until one full sync cycle has obtained a usable
  token and completed a call to Meta, and then stays 200 even through a later transient
  failure, since the last good feed keeps being served regardless. A cycle that reaches
  Meta and finds zero posts still counts: an Instagram account with nothing posted is a
  real, permanently empty feed, not a readiness failure. A feed pod that cannot reach
  OpenBao or Meta therefore sits at `0/1` ready, retrying, rather than crash-looping or
  serving stale or absent data as if nothing were wrong. Its HTTP surface authenticates
  no caller of its own; it relies on the chart's `NetworkPolicy` admitting only the
  API's pods instead, see `charts/ostendorf.tech/README.md`.
- `internal/feedclient` is the API's own client against the feed service above: a
  `Client` for its two routes, a `Cache` of the posts and picture bytes it last got back,
  and a `Syncer` that polls the feed service on an interval (`FEED_REFRESH_INTERVAL`) and
  keeps that `Cache` current, the same caching `internal/instagram` itself used to do
  before this split. A failed sync is retried after 5 s, doubling up to 5 minutes or the
  interval, until one succeeds (`Syncer.RetryDelay`). Every call is bounded by a timeout,
  so an unreachable or slow feed service can never hang a page; an empty `Cache`, whether
  from an unreachable feed service, an unset `FEED_SERVICE_URL`, or a feed service with
  nothing to show, renders no Impressionen section at all, exactly as an absent token did
  before this split. It takes at most 10000 posts and 16 MiB of post list per cycle, the
  same ceiling as `POST_COUNT`, whatever the feed service sends.
  `internal/webapp/feed.go` is the one place this package is read from. Its
  `GET /api/v1/instagram` answers every cached post at once, and `InstagramFeed.vue` shows
  them in batches of 12 per click without a further request.
- Page views are reported to Umami from the frontend server by govue's `analytics`
  package, through `frontend.Config.Analytics`; the site has no analytics code of its
  own. `internal/frontendconfig` reads it through `analytics.LoadConfig` (`UMAMI_URL`,
  `UMAMI_WEBSITE_ID`, `UMAMI_TIMEOUT`, `UMAMI_QUEUE_SIZE`), and `cmd/frontend/site.go`
  builds the client with the trusted proxy resolver (`TRUSTED_PROXIES`, chart value
  `config.trustedProxies`), `FullIP` off and the host of `SITE_URL`. govue places the
  counter around its mux, registers `analytics_views_dropped_total` and closes the
  client after the listeners stopped. Only a page answered with 200, or revalidated
  with 304, counts, by its pattern and without the query; redirects, the legal aliases,
  404s, assets, public files, robots.txt and the sitemap count nothing. The API counts
  nothing. `cmd/frontend/site_test.go` proves it through the real build. What is counted
  is stated in `web/src/views/PrivacyView.vue`; change that text whenever the payload
  changes, which means a govue upgrade that touches the analytics privacy checklist.
- The site runs as three images, released under one tag and deployed together:
  `ostendorf-tech-frontend` (`cmd/frontend`), `ostendorf-tech-api` (`cmd/api`) and
  `ostendorf-tech-feed` (`cmd/feed`). The Ingress routes `/api` to the API and every
  other path to the frontend, so the browser sees one origin: no CORS, `connect-src
  'self'`, and the contact form's `SITE_URL` trusted origin as before.
- `cmd/frontend` is govue's `frontend` package and nothing else of its own: it embeds
  `cmd/frontend/dist`, Vite's build with the prerendered pages, the 404 page per
  language and `sitemap.xml` written into it, and calls `frontend.New` and `Run`. govue
  serves every `loc` of the sitemap at its exact path, the slash forms with 301, the
  assets, the three favicon files and `og-image.png` (`PublicFiles`), robots.txt from the
  sitemap's origin, the language redirect at `/`, and `language.HandleLegal`'s aliases
  `/imprint`, `/privacy`, `/terms`, `/impressum`, `/datenschutz`,
  `/datenschutzerklaerung` and `/agb`. `/de/agb` and `/en/agb`, with and without the
  slash, answer 301 to the terms page through `Config.Redirects`. govue injects the
  favicon links, one `theme-color` tag per mode (`ThemeColor` `#f6f5fd`,
  `ThemeColorDark` `#07060f`, the two ends of `--color-bg`) and its theme boot script
  with `themeboot.WithDefaultMode(themeboot.ModeDark)`, and adds the script's hash to
  `script-src`; it refuses a page that carries any of them already. The page rate limit
  keeps an HTML answer in the page's language. The frontend holds no secret
  (`internal/frontendconfig` reads none) and serves no API: `/api/` answers 404, unless
  `FRONTEND_DEV_API_PROXY` is set, the development switch that turns on
  `Config.Development` and forwards `/api/` to the API container. Only
  `docker-compose.yaml` sets it, the chart never. The sitemap and robots.txt carry the
  origin the image was built with (`VITE_SITE_URL`, default `https://ostendorf.tech`),
  so an image serves one production origin. `cmd/frontend/build_test.go` runs the real
  prerender output: every route of `internal/routes` has a page and every page answers
  a route (`TestPrerenderedOutputMatchesRouteTable`, `task test:routes`, its own CI
  job), every page pins `data-mode="dark"`, the indexing tags, no head markup govue
  writes, and govue's `favicon.CheckPages` over the served pages. `site_test.go` feeds
  the real build to `frontend.New` and checks the headers, the redirects, the counting
  and the dev proxy.
- `cmd/api` and `internal/webapp` are the API: two HTTP handlers from govue's packages,
  no embedded frontend. `Handler` is the public one: the security and access log
  middleware, the client IP resolver and the rate limiter (govue's JSON answer), in front
  of govue's contact handlers, `GET /api/v1/instagram` from the feed client and a JSON
  404 for every other path. There is no session manager and no CSRF protector: every
  other route answers GET only and changes nothing, and the contact handlers check the
  origin themselves. `AdminHandler` serves health, readiness and metrics on the admin
  listener, reached only from inside the cluster, never through the chart's Ingress.
  `internal/headertest` holds the security headers the single image sent before the
  split; the frontend's and the API's tests compare their answers against it, with
  only the nonce and the boot script hash dropped from the policy. `contact.go` mounts
  govue's stateless contact form (see its own bullet above): no cookie, no session, no
  CSRF token. govue keeps the contact form's per client limits, its global mail ceiling
  and its proof of work replay set in the memory of one process, so the API Deployment
  stays at one replica: a second replica, or two pods briefly coexisting during an
  ordinary rolling update, has its own copy of each, which doubles every budget, the
  mail ceiling included, and lets a solved challenge be used once per pod.
  `charts/ostendorf.tech` enforces the single replica at render time and runs the API
  Deployment with `strategy.type: Recreate`, so a rollout never lets two pods coexist
  either. The frontend has no such state and may run several replicas. Lifts only once
  all of them move to a shared store: `pow.Config.Seen` is an interface that can take
  one, but `contact.Config`'s limiters are `*ratelimit.Limiter`, a concrete in-memory
  type, so govue offers no shared limiter in v0.9.0.
- `web/` is the Vue SPA, consuming govue's Vue package as a pnpm git dependency. `web/src`
  holds the app. The contact section in `web/src/views/HomeView.vue` renders govue's
  `GovueContactForm` with a cookie free client (`createClient({ credentials: 'omit' })`),
  the site's own consent text (`home.contact.consent`), the form heading and the privacy
  link of the page's language; its fields, errors and success text come from govue's
  catalogue. The site says "Sie": `main.ts` and the prerender script both install
  `createGovueI18n` with `formality: 'formal'`, govue's formal German catalogue, and
  `web/src/i18n/formality.test.ts` fails when either stops doing so. Its proof of work
  Worker is govue's, emitted by Vite as a same-origin file under `/assets/` that govue's
  default `worker-src 'self'` allows. `web/src/views/contactSection.test.ts` proves the
  section in both languages. The header (`web/src/components/AppHeader.vue`) is govue's
  `GovueHeader` with the navigation at the end, the contact link as the `cta` nav item,
  `GovueLanguageSwitch` in link mode (`linkFor`, `navigate` to the router) and the
  compact `GovueThemeSwitch` in the `#theme` slot; its translucent ground is govue's
  `--color-header-bg` and `--header-backdrop` tokens in `web/src/theme/vtech.css`. On the
  home page it is govue's `overlay` header, transparent over the hero until the page
  scrolls. That is an exception to govue's documented hero bound: the brightest hero
  ground is about 0.0638 in dark mode, above govue's 0.05 and above the bound with
  `text-muted`, which no header part paints. `web/src/theme/contrast.test.ts` guards the
  site's own bound over the roles the header paints. In light mode the hero is pale, so
  `web/src/styles.css` gives the transparent header the light roles and the navy focus
  ring. The hero itself is unchanged apart from filling the window under the header.
  `web/scripts/prerender.ts` is the static site generation step `task prerender` runs.
  `web/src/lib/routeTable.ts` is the one place the frontend reads
  `internal/routes/routes.json` from; `web/scripts/prerender.ts`, `web/src/router/index.ts`
  (the Vue Router table) and `web/src/lib/seo.ts` (the `PageRoute` list and its slugs) all
  import it instead of restating the page list themselves.
- `charts/ostendorf.tech/` is the Helm chart for all three parts: a Deployment, Service
  and ServiceAccount each for the frontend, the API and the feed, the Ingress or
  HTTPRoute (`/api` to the API, `/` to the frontend) and one NetworkPolicy per part; see
  `charts/ostendorf.tech/README.md`. `scripts/validate-chart.sh` renders it for its
  default values and every documented non-default combination, and for a must-fail set
  proving every render-time `fail` and `required` guard actually rejects the values it
  is meant to reject, shared by `task lint:helm` and `task validate:kubeconform`.
  `Dockerfile` is the multi-stage build for all three images, as final stages
  `frontend`, `api` and `feed` (`docker build --target <name>`), since the binaries share
  `go.mod`/`go.sum` and the pinned base image digests; only `frontend` runs the Node
  stage. `.github/workflows/publish.yaml` builds each image on a native amd64 and arm64
  runner and merges one manifest list per image under the release tag.
  `docker-compose.yaml` is a local development stack: `frontend` on the one published
  port (`LISTEN_PORT`), forwarding `/api/` to `api` through `FRONTEND_DEV_API_PROXY`, so
  the browser has one origin as in production, plus one feed service of the owner's
  choice, never both: `fakefeed` (`cmd/fakefeed`, standard library only) under the
  `feed` profile, sample posts with generated pictures on the real feed service's routes,
  or `realfeed`, the real `cmd/feed` (token method, `OPENBAO_TOKEN` only a pass-through
  from `.env`) under the `realfeed` profile. `FEED_SERVICE_URL` in `.env` names the one
  the API reads. `fakefeed` is a fourth `Dockerfile` target that no published image
  depends on and no workflow builds; `internal/envsync` proves both, that each service's
  port agrees across the compose file, the program and `.env.example`, and that the dev
  proxy points at the API.

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

## pnpm is pinned twice and moves once

`mise.toml` pins pnpm for a local run and the Dockerfile's Node stage installs the same
patch through corepack from its `ARG PNPM_VERSION`, so the image reads the lockfile with
the pnpm that wrote it. Renovate's mise manager reads the first pin; a regex custom
manager in `renovate.json` reads the argument under its `# renovate: datasource=npm
depName=pnpm` comment, and the package rule with `groupName` `pnpm` puts both into one
pull request with no manual step. `internal/envsync`'s `TestDockerfilePnpmMatchesMise`
fails when the two differ or when the install line names a literal version, and
`internal/depsync`'s `renovate_test.go` fails when the manager no longer extracts the
argument or the group is gone.

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
(`#0f1a2e`, this theme's own `--color-text`) for light mode and white for dark mode,
matching the two variants the design calls for. Moving that token means regenerating
the light variant: it carries the value, not a reference to it. The command:

```sh
pdftocairo -svg Logo_Name.pdf logo-wordmark-light.svg
# then replace every fill="rgb(...)" with fill="#0f1a2e" (light) or fill="#ffffff" (dark)
svgo --multipass -i logo-wordmark-light.svg -o logo-wordmark-light.svg
```

repeated once per variant.

`web/src/assets/hero-*.{avif,webp,jpg}` and `portrait-*.{avif,webp,jpg}` are generated
from `hero.jpg` (2400x1800) and `portrait.jpg` (1200x1200) in old_simon-ostendorf.de's
`cmd/website/static/img`, never hand edited and never re-cropped. Both are the owner's
own photographs. The build pipeline does not resize or re-encode anything, on purpose:
these two pictures change about as often as the logo does, so paying for an image
pipeline on every `task web:build` would buy nothing. `web/src/lib/heroImage.ts` is the
one place the variants are listed, with the width of each next to its import, and Vite
content hashes them from there. The commands:

```sh
magick hero.jpg -resize 1280x -strip -quality 50 web/src/assets/hero-1280.avif
magick hero.jpg -resize 1280x -strip -quality 72 -define webp:method=6 web/src/assets/hero-1280.webp
```

repeated for 640, 960, 1280, 1600, 1920 and 2400. The hero's JPEG fallback is the 1280
width at `-quality 78 -sampling-factor 4:2:0 -interlace JPEG`.

The portrait is the exception: it is shown large and close, so it keeps higher quality
and the original's own width. Its source is `portrait.jpg` of the old site
(`https://www.simon-ostendorf.de/static/img/portrait.jpg`, 1200x1200, 133 KB, the
owner's own photograph). It is not committed: put it at `web/src/assets/portrait-source.jpg`,
where `.gitignore` keeps it out. Every variant is made from that file and none is scaled
up. The commands, run in `web/src/assets/`, for W in 460, 920 and 1200:

```sh
magick portrait-source.jpg -resize Wx -strip -quality 80 portrait-W.avif
magick portrait-source.jpg -resize Wx -strip -quality 90 -define webp:method=6 portrait-W.webp
magick portrait-source.jpg -resize Wx -strip -quality 85 -sampling-factor 4:2:0 \
  -interlace JPEG portrait-W.jpg
```

The JPEG commands run for 460 and 920 only: `portrait-1200.jpg` is the source file
copied unchanged, so the widest fallback loses nothing. Each file stays under 150 KB.
`web/src/lib/heroImage.ts` lists all nine with their widths, and
`web/src/lib/portraitImage.test.ts` reads every file's header and fails when a declared
width differs from the file's pixels.

`web/public/favicon.svg` is the one hand kept source of the icon: the earlier site's 32 by
32 fader mark on a dark rounded square. `web/public/favicon-32x32.png`,
`apple-touch-icon.png` (180x180, opaque, the square's own dark colour fills the corners)
and `og-image.png` (1200x630, the mark and the white wordmark on that same colour) are
generated from it and from `web/src/assets/logo-wordmark-dark.svg`, never hand edited. The
file names are govue's favicon contract (its Go package `favicon`), plus `og-image.png`.
Vite copies `web/public` to the root of `dist/`. govue's frontend serves the three icon
files and `og-image.png` (a `PublicFiles` entry) from there, refuses to start without
one, and injects the icon links into every page itself; `web/src/lib/icons.ts` writes
only the social preview image's tags. `cmd/frontend/build_test.go` runs govue's
`favicon.CheckPages` over the pages as the server sends them. The command, needing ImageMagick from
`mise.toml`, run again whenever the SVG or the wordmark changes:

```sh
task favicon:generate
```

The icon is a dark square with a bright gradient mark, so it reads on a light and on a dark
tab bar without a colour scheme switch. The SVG carries literal colours because a favicon
cannot read the page's design tokens.

The fader mark alone (the icon, `LogoMark.vue`) is not
generated: it is inline Vue markup ported from old_simon-ostendorf.de's `{{define
"logo"}}` template block, with a gradient whose stops read the theme's `--logo-from` and
`--logo-to` tokens: the favicon's `#0ff0ff` and `#803cff`, the same in light and dark mode
and independent of the palette's accent roles, so it never needs regenerating when the
theme changes.

`web/src/assets/bands/herrenjedeck-logo.webp` is generated from the band's own logo,
`https://www.herrenjedeck.de/assets/logo-dark.png` (1182x945, transparent background),
never hand edited. The source is not committed: download it to
`web/src/assets/bands/herrenjedeck-logo-dark.png`, which `.gitignore` keeps out. The
command, needing ImageMagick from `mise.toml`, trims the transparent margin and scales to
480 px wide (480x489, about 28 KB):

```sh
magick herrenjedeck-logo-dark.png -trim +repage -resize 480x -strip \
  -define webp:method=6 -define webp:alpha-quality=90 -quality 80 herrenjedeck-logo.webp
```

The mark is a dark badge on a transparent ground, so `BandLogo.vue` sets it on a plate of
`--color-accent-on`, which is near white in both modes. Further band logos follow the same
pattern; see the comment in `web/src/data/bands.ts`.

`web/src/assets/bands/bademeister.webp` and `rubik.webp` are generated from the bands' own
pictures, never hand edited: `bademeister-source.jpg` (the band's profile picture,
1080x1080) and `rubik-source.jpg` (the band's photo, 315x315). The sources are not
committed: put them in `web/src/assets/bands/`, where `.gitignore` keeps `*-source.jpg`
out. Rubik's file is not scaled up, so it stays 315x315 and looks soft on high density
screens. Both stay under 60 KB. The commands, needing ImageMagick from `mise.toml`:

```sh
magick bademeister-source.jpg -resize 640x640 -strip -define webp:method=6 -quality 80 bademeister.webp
magick rubik-source.jpg -strip -define webp:method=6 -quality 85 rubik.webp
```

They are square pictures without transparency, so `BandPhoto.vue` shows them as rounded
squares with no plate, through the `photo` field in `web/src/data/bands.ts`.

## Scope of this milestone

The home page's hero, Leistungen, Impressionen, Bands, Über mich and Kontakt sections,
the imprint, privacy and AGB pages, crawling (`robots.txt`, `sitemap.xml`), the Instagram
feed (split across the API and the feed service, see `internal/instagram`'s own
bullet above), the Umami hookup, the three container images and the Helm chart are
implemented. Left out, tracked as later milestones:

- The Impressionen section is rendered by the browser after the page loads, from
  `GET /api/v1/instagram`, not baked into the prerendered HTML: the feed changes while
  the binary does not, and `task prerender` runs at build time with no access token. A
  crawler that executes no JavaScript therefore does not see the section. That is
  deliberate, since the section carries no content worth indexing. The nav's
  Impressionen entry follows the section the same way: the prerendered nav has none,
  because it cannot know whether the section will be there, and `InstagramFeed.vue` sets
  `impressionsOnPage` (`web/src/lib/pageSections.ts`) once its section is rendered, which
  makes `AppHeader.vue` add the entry between Leistungen and Bands. The entry shifts the
  links after it sideways when it arrives; the header's height does not change. It is
  left out of the row between 640 and 1023 px (`IMPRESSIONS_HIDDEN_QUERY`, the `sm` to
  below `lg` range), where the one row has no room for a fifth German link; it is in
  govue's menu below 640 px and in the row from 1024 px.

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
