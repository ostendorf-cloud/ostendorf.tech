# ostendorf.tech

The business website of Simon Ostendorf Veranstaltungstechnik: a German and English,
light and dark themed site for an event technology sole trader in Aachen, Germany.

## What it does

A Vue 3 single page application built on the organisation's own `govue` framework,
served by three Go images released under one tag and deployed together:

- `ostendorf-tech-frontend` (`cmd/frontend`): the pages, through govue's `frontend`
  package. Every public route is prerendered to static HTML at build time, so a crawler
  or a browser's first paint gets real markup before the SPA takes over. No secret, no
  API.
- `ostendorf-tech-api` (`cmd/api`): the contact form and `GET /api/v1/instagram`. No
  page.
- `ostendorf-tech-feed` (`cmd/feed`): reads the Instagram feed from Meta, reached by the
  API only.

The Ingress routes `/api` to the API and everything else to the frontend, so the browser
sees one origin. The pages, `sitemap.xml` and `robots.txt` carry the origin the frontend
image was built with (`VITE_SITE_URL`, default `https://ostendorf.tech`): an image serves
one production origin.

The home page's hero, Leistungen, Impressionen, Bands, Über mich and Kontakt sections, and the
imprint, privacy and AGB pages, are implemented in German (the source of truth) and
English. The Kontakt section's contact form is govue's stateless one: it sends mail over
SMTP, protected by an origin check, a proof of work challenge, a per-address rate limit and
a global mail ceiling, with no cookie and no session. See
`AGENTS.md` for what is deliberately left out of this milestone.

## Install

```sh
task setup
```

Installs the pinned toolchain (Go, Node, pnpm, and the lint and release tooling)
through `mise`.

## Use

```sh
task web:build   # builds the SPA into cmd/frontend/dist
task prerender   # writes every route of every locale into cmd/frontend/dist
task build        # compiles the frontend binary into bin/frontend
task build:api    # compiles the API binary into bin/api
bin/frontend
```

Both servers read their configuration from the environment. Every secret-shaped
variable of the API also accepts a `_FILE` twin naming a path whose contents are the
value, trimmed of a trailing newline, and it is an error to set both or to point `_FILE`
at a file that resolves to nothing. There is no secret manager client in either binary:
an OpenBao agent, External Secrets, a Kubernetes secret mount or a plain `.env` file are
all interchangeable. Every variable has a usable default or may stay unset, so both
binaries run with none of them set; the contact form then answers 503 `not_configured`.
The frontend serves the site at `/de/...` and `/en/...`, redirects `/` to the visitor's
language, and serves `/robots.txt` and `/sitemap.xml` from the prerendered sitemap.
`/healthz`, `/readyz` and `/metrics` answer on a second, administrative listener
(`ADMIN_ADDR`) of each, never on the public one; see the Configuration table below.

## Configuration

The frontend and the API read the variables of the first rows. The column "Image" names
the one that reads a variable alone. The feed service's variables are in `.env.example`.

| Variable | Image | Purpose | Default | Required |
| --- | --- | --- | --- | --- |
| `LISTEN_ADDR` | both | Address the public HTTP server listens on | `:8080` | no |
| `ADMIN_ADDR` | both | Address govue's admin listener listens on: `/healthz`, `/readyz` and `/metrics` | `:8081` | no |
| `SITE_URL` | both | The site's public origin: the contact form trusts it, the proof of work audience is its host, page views are reported under its host. The pages and the sitemap carry the build's `VITE_SITE_URL` instead | `https://ostendorf.tech` | no |
| `FRONTEND_DEV_API_PROXY` | frontend | Development only: forwards `/api/` to this API URL, so compose has one origin. Never set in production | none | no |
| `LOG_LEVEL` | both | `debug`, `info`, `warn` or `error` | `info` | no |
| `TRUSTED_PROXIES` | both | Comma separated CIDRs or addresses trusted to set `X-Forwarded-For` | none | no |
| `ALLOW_PLAIN_HTTP` | both | Drops the `Strict-Transport-Security` header, for a plain HTTP dev run | `false` | no |
| `RATE_LIMIT_BURST` | both | Requests one client address may make before waiting, on every route | `120` | no |
| `RATE_LIMIT_PER` | both | The burst's refill period | `1m` | no |
| `CHALLENGE_KEY` (+ `_FILE`) | api | Hex encoded, at least 32 bytes. Signs the proof of work challenge | none | no\* |
| `CHALLENGE_KEY_PREVIOUS` (+ `_FILE`) | api | Hex encoded, at least 32 bytes. The key `CHALLENGE_KEY` was just rotated out of; see "Rotating the challenge key" below | none | no |
| `SMTP_HOST` | api | SMTP submission server the contact form sends through, a bare host name | none | no\* |
| `SMTP_PORT` | api | Its port: `587` requires STARTTLS, `465` means TLS from the first byte | `587` | no |
| `SMTP_USERNAME` (+ `_FILE`) | api | SMTP username, set together with `SMTP_PASSWORD` | none | no |
| `SMTP_PASSWORD` (+ `_FILE`) | api | SMTP password | none | no |
| `CONTACT_TO` | api | Recipient address of every contact mail | `simon@ostendorf.tech` | no |
| `CONTACT_FROM` | api | Envelope sender address of every contact mail | `ostendorf.tech <no-reply@ostendorf.cloud>` | no |
| `CHALLENGE_RATE_LIMIT_BURST` | api | Challenge requests one client address may make before waiting | `10` | no |
| `CHALLENGE_RATE_LIMIT_PER` | api | That burst's refill period | `20s` | no |
| `CONTACT_RATE_LIMIT_BURST` | api | Contact submissions one client address may make before waiting | `5` | no |
| `CONTACT_RATE_LIMIT_PER` | api | That burst's refill period | `10m` | no |
| `FEED_SERVICE_URL` | api | Origin of the feed service the Impressionen section is loaded from. Empty renders no section at all | none | no |
| `FEED_REFRESH_INTERVAL` | api | How often the API asks the feed service for the post list, at least `1m`. A failed sync is retried after 5s, 10s, 20s, 40s and so on (at most `5m` or this interval) until one succeeds | `5m` | no |
| `UMAMI_URL` | frontend | Base URL of the Umami page views are reported to. `https`, or `http` only for localhost or a name inside the cluster. Empty counts nothing | none | no\*\* |
| `UMAMI_WEBSITE_ID` | frontend | UUID of the website in Umami | none | no\*\* |
| `UMAMI_TIMEOUT` | frontend | Bound of one post to Umami, `1s` to `30s` | `5s` | no |
| `UMAMI_QUEUE_SIZE` | frontend | Page views that may wait for a worker, `1` to `10000`; a full queue drops the view | `256` | no |
| `MAIL_RATE_LIMIT_BURST` | api | Contact mails this service sends in total, across every client address, before waiting | `50` | no |
| `MAIL_RATE_LIMIT_PER` | api | That burst's refill period | `1h` | no |

\* Without `CHALLENGE_KEY` or without `SMTP_HOST`, both contact form routes answer 503
`not_configured` instead of the server failing to start; every other page keeps working.
The SMTP variables are read by govue's `mail.LoadConfig`. A username without a password, or
the other way round, stops the start. With `SMTP_HOST` and neither credential, the form
sends without SMTP AUTH.

The contact form is govue's stateless one (`contact.Handler`): `GET /api/v1/contact/challenge`
and `POST /api/v1/contact`. It sets no cookie, keeps no session and takes no CSRF token. It
checks the origin itself, trusting the origin of `SITE_URL`, and keeps its limits and its
proof of work replay set in the memory of the one process.

\*\* `UMAMI_URL` and `UMAMI_WEBSITE_ID` are set together or not at all; exactly one of
them is refused at startup rather than silently counting nothing.

The Impressionen section and the visitor statistic are both off until configured and
both run server side only. The production site counts page views with Umami, which the
owner runs on their own hardware in Germany; the privacy policy states this plainly.
Without `FEED_SERVICE_URL` no cache is loaded and the section
is not rendered; the rest of the page is unaffected, and a feed service that is
unreachable or empty never changes what a visitor sees, the same way an unconfigured URL
does. The posts and their pictures are fetched by the API, from the feed service, and
served from this domain, so a visitor's browser never connects to the feed service,
Instagram or Meta. Without `UMAMI_URL` and `UMAMI_WEBSITE_ID` no page view is reported;
with them, the frontend reports each view to Umami itself, with no script, cookie or
beacon in the visitor's browser, and a failure reaching Umami never affects the response.
The counting is govue's `analytics` package, placed by govue's `frontend` server. It
reports the route pattern of a page answered with 200 or revalidated with 304 (never
the path or a query string), the origin of the referrer, the browser language, the user
agent and the client address shortened to its /24 (IPv4) or /48 (IPv6). It leaves out
Do Not Track, Global Privacy Control, prefetch, bots, private client addresses,
redirects, errors, assets, public files, the sitemap, robots.txt and the API. Behind a
proxy it needs `TRUSTED_PROXIES`: without it every request comes from the proxy's private
address and nothing is counted (the startup log says so). The variable names did not
change when the site moved to govue's package, so an existing deployment keeps working.
Two things are stricter than before: plain `http` is accepted only for localhost or a
name inside the cluster, and an invalid value stops the start with a message naming
the variable.

Neither the frontend nor the API has an Instagram or Meta credential or an OpenBao
client: both live in the feed service instead, a third binary (`cmd/feed`) with its own
configuration and its own chart entry, reached by the API over `FEED_SERVICE_URL` alone.
Its variables are listed in `.env.example`; see "Running the real feed service locally"
below for a local run.

`CONTACT_RATE_LIMIT_BURST` bounds one client address; `MAIL_RATE_LIMIT_BURST` bounds the mail
account itself, across every address at once, which the per-client limit alone cannot do: a
client address is an IPv6 /64, and a single routed /48 is 65,536 of them. The default, 50 an
hour, is well above anything one event technology sole trader's contact form is ever expected to
receive legitimately. Exceeding it answers 503 `mail_limit` with a `Retry-After`, and govue logs
`global mail ceiling reached` at warning level at most once a minute, since it means either an
attack or a genuine surge worth the operator's attention, unlike an ordinary per-client
throttle, which answers 429.

### Rotating the challenge key

Set `CHALLENGE_KEY_PREVIOUS` to the key `CHALLENGE_KEY` previously held, in the same change that
moves `CHALLENGE_KEY` to the new one. A challenge issued just before the rotation, and still
sitting unsolved in a visitor's browser, verifies against either key; govue's `pow` signs new
challenges with `CHALLENGE_KEY` only. Remove `CHALLENGE_KEY_PREVIOUS` again once the challenge
TTL (10 minutes) has safely passed. Leaving it unset, as before, means rotating
`CHALLENGE_KEY` invalidates every challenge already issued the moment the new key takes
effect.

`CHALLENGE_KEY_PREVIOUS` is for a planned rotation only. Never set it to a leaked key to keep that
key verifying a little longer: a leaked `CHALLENGE_KEY` is dropped at once, with nothing put in
`CHALLENGE_KEY_PREVIOUS` for it, accepting up to ten minutes of broken challenges rather than
keeping the leaked key alive a moment longer than replacing it takes.

Behind any reverse proxy or ingress, leaving `TRUSTED_PROXIES` unset means every
request is attributed to the proxy's own address, not the visitor's. Every rate limit
then collapses into one shared bucket for the whole site: about 20 page loads a minute
site-wide at the documented default, and the contact form blocked for every visitor for
10 minutes after 5 submissions from anyone. Set it to the proxy's or ingress
controller's own address range.

## Container

```sh
cp .env.example .env   # see .env.example; no variable is required
docker compose up --build
```

An empty `.env` starts the site. The contact form needs `CHALLENGE_KEY` and `SMTP_HOST`, and
`SMTP_USERNAME` and `SMTP_PASSWORD` for a server with SMTP AUTH, and answers 503 without the
first two. A plain `docker compose up` runs the frontend and the API, no feed service, so
the Impressionen section stays off unless `FEED_SERVICE_URL` points at a feed service
started some other way. Only the frontend publishes a port. It forwards `/api/` to the
API container through `FRONTEND_DEV_API_PROXY`, the development switch of govue's
frontend server, so the browser talks to one origin, as the Ingress gives production.

To see the Impressionen section locally, start one feed service next to the API. It is
one choice: a profile and the matching `FEED_SERVICE_URL` in `.env`. Start only one of the
two. Neither publishes a host port.

| Feed | `.env` | Command |
| --- | --- | --- |
| Fake, generated sample posts | `FEED_SERVICE_URL=http://fakefeed:9081` | `docker compose --profile feed up --build` |
| Real, against your OpenBao | `FEED_SERVICE_URL=http://realfeed:9081` | `docker compose --profile realfeed up --build` |

With Podman, write `podman compose` instead of `docker compose`. See "Real feed service with
compose" below for the real one.

The `fakefeed` service (`cmd/fakefeed`) serves generated sample posts and pictures on the
same routes as the real feed service, never real Instagram content. It publishes no host
port: the API reaches it on the compose network. `POST_COUNT` in `.env` sets how many
sample posts there are, 0 to 200, and empty or 0 means its whole sample account of 40. The
real feed service reads the same variable as a cap, see below. The fake is its own `fakefeed`
target in the
`Dockerfile`; no production target depends on it and no workflow builds or publishes it.
The fake needs no OpenBao.

Builds the multi-stage `Dockerfile` (the Vue build and prerender, static Go binaries,
then minimal non-root runtime stages with no shell) and serves the site at
`http://localhost:8080`. `docker-compose.yaml` is for local development only; see
`charts/ostendorf.tech/README.md` for a cluster installation.

To use another port, set `LISTEN_PORT=8123` in `.env`. Compose passes `LISTEN_ADDR=:8123`
to the container and publishes `127.0.0.1:8123:8123` from that one value. Compose cannot
cut a port out of an address, so `LISTEN_ADDR` itself is not read from `.env` here.
`internal/envsync` checks that the two stay in step. The admin listener (`:8081`) is not
published.

`scripts/smoke-test-image.sh <image-ref> frontend|api` starts an already built image the
same way, with no secret, and asserts its admin listener (`ADMIN_ADDR`, `:8081`) answers
`/healthz`, `/readyz` and `/metrics` and the public listener does not. For the frontend
it asserts the home page in each locale with its prerendered content intact,
`/robots.txt`, `/sitemap.xml` and no API; for the API the feed route and no page. The
`docker-build` job in `.github/workflows/checks.yaml` runs it against both images on
both platforms, right after building them.

The same `Dockerfile` builds all three images as final stages, `docker build --target
frontend`, `api` or `feed`: the binaries share `go.mod`/`go.sum` and the pinned base image
digests, and BuildKit never evaluates a stage the requested target does not depend on, so
only `--target frontend` runs the Vue stage. `docker build` with no `--target` resolves to
the frontend image, the last stage in the file. `.github/workflows/publish.yaml` builds
each image for `linux/amd64` and `linux/arm64` and publishes `ostendorf-tech-frontend`,
`ostendorf-tech-api` and `ostendorf-tech-feed` under the release tag. The three are one
version: deploy them together, as the chart does.

`scripts/smoke-test-feed-image.sh <image-ref>` starts that image the same way, with
`OPENBAO_ADDR` pointed at a host that does not resolve, and asserts what is honestly
provable without a real OpenBao and a real Meta access token: the process stays up,
`/healthz` answers 200 throughout, `/readyz` stays 503 throughout (this service never
reports ready without a usable token and a synced feed, see "Configuration" above), and
the posts and image routes already answer correctly off an empty, unsynced feed. It does
not and cannot prove that this service ever reaches `/readyz` 200, or that OpenBao's
Kubernetes auth actually admits its role; both need a real cluster. The `docker-build-feed`
job in `.github/workflows/checks.yaml` runs it the same way `docker-build` runs the
frontend's and the API's script.

### Real feed service with compose

The `realfeed` service builds the `feed` target and runs `cmd/feed` with the token method
against an OpenBao you run yourself. Put these into `.env` (git ignored, never commit it):

```sh
FEED_SERVICE_URL=http://realfeed:9081
OPENBAO_ADDR=http://host.containers.internal:8200
OPENBAO_TOKEN=<a token allowed to read and write the secret below>
OPENBAO_SECRET_PATH=<KV v2 data path of the secret>
```

```sh
podman compose --profile realfeed up --build
```

- `OPENBAO_AUTH_METHOD` defaults to `token` in this stack.
- An OpenBao on the host is not `127.0.0.1` inside a container. Use
  `http://host.containers.internal:8200` with Podman and `http://host.docker.internal:8200`
  with Docker.
- Compose also forwards `OPENBAO_SECRET_KEY_ACCESS_TOKEN`, `OPENBAO_SECRET_KEY_REFRESHED_AT`,
  `SYNC_INTERVAL`, `TOKEN_REFRESH_AGE`, `API_VERSION`, `POST_COUNT`, `POSTS_PER_REQUEST` and
  `LOG_LEVEL`. Empty means the service's own default.
- `POST_COUNT` caps the number of posts, 0 means the maximum of 10000. The default is 200.
  `POSTS_PER_REQUEST` is how many posts one request to Meta asks for, default 50, `1` to
  `100`. Each post costs its picture in memory, in the feed service and again in the
  API: see "Posts and memory" in `charts/ostendorf.tech/README.md`, which also covers
  the chart's `GOMEMLIMIT`.
- The feed service follows no redirect from Meta or from a picture host.
- `OPENBAO_TOKEN_FILE` is not forwarded: a bind mount of a file that may be absent cannot be
  optional in compose.
- The token never goes into `docker-compose.yaml`. It is only read from `.env` or the shell.
- The service never renews the token. Once it expires or is revoked, OpenBao answers 403 and
  `/readyz` stays 503. Put a new token into `.env` and restart: `podman compose --profile
  realfeed up -d --force-recreate realfeed`.

### Running the real feed service locally

The real feed service needs OpenBao and a Meta access token already stored in it. In a
cluster it logs in with the Kubernetes auth method. For a local run against a development
OpenBao it uses the token method instead. Set these in the shell, or put them into `.env`
and export them, then run `go run ./cmd/feed`:

```sh
OPENBAO_AUTH_METHOD=token
OPENBAO_ADDR=http://127.0.0.1:8200
OPENBAO_TOKEN=<a token allowed to read and write the secret below>
OPENBAO_SECRET_PATH=<KV v2 data path of the secret>
LISTEN_ADDR=:9081
```

- `OPENBAO_TOKEN_FILE` may name a file holding the token instead. Setting both is a startup
  error.
- `OPENBAO_SECRET_KEY_ACCESS_TOKEN` and `OPENBAO_SECRET_KEY_REFRESHED_AT` name the keys
  inside the secret. They default to `access_token` and `refreshed_at`.
- `OPENBAO_ROLE`, `OPENBAO_JWT_FILE` and `OPENBAO_AUTH_PATH` belong to the Kubernetes method.
  Setting any of them with the token method is a startup error.
- A token in `.env` is a secret. `.env` stays git ignored and must never be committed.
- The service never renews the token. Once it expires or is revoked, OpenBao answers 403,
  `/readyz` stays 503 and no request is retried. Give it a new token and restart it.
- The token never appears in a log line, an error or a formatted value.

## Development

```sh
task setup
task check
```

`task check` runs every lint, build and test target one after another; `task --list`
shows them individually. `task web:build` and `task prerender` have to run before
`task build` or `task test:go` can compile `cmd/frontend`, since it embeds their
output; `task build` and `task test:go` already declare that dependency, so running
either one through Task is enough. `cmd/api` and `cmd/feed` embed nothing.

Releases are driven by [release-me](https://github.com/ostendorf-cloud/release-me):
Conventional Commits decide the version, and merging the release pull request it
maintains cuts the release. `CHANGELOG.md` is generated, never hand edited.
