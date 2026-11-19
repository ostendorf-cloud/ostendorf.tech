# ostendorf.tech

The business website of Simon Ostendorf Veranstaltungstechnik: a German and English,
light and dark themed site for an event technology sole trader in Aachen, Germany.

## What it does

A Go service serves a Vue 3 single page application built on the organisation's own
`govue` framework. Every public route is also prerendered to static HTML at build
time, so a crawler or a browser's first paint gets real markup before the SPA takes
over.

The home page's hero, Leistungen, Bands, Über mich and Kontakt sections, and the
imprint, privacy and AGB pages, are implemented in German (the source of truth) and
English. The Kontakt section's contact form sends mail over SMTP, protected by a
proof of work challenge, a per-session CSRF token and a per-address rate limit. See
`AGENTS.md` for what is deliberately left out of this milestone.

## Install

```sh
task setup
```

Installs the pinned toolchain (Go, Node, pnpm, and the lint and release tooling)
through `mise`.

## Use

```sh
task web:build   # builds the SPA into cmd/website/dist
task prerender   # renders every route of every locale into cmd/website/prerendered
task build        # compiles the website binary into bin/website
bin/website
```

The server reads its configuration from the environment; every secret-shaped variable
also accepts a `_FILE` twin naming a path whose contents are the value, trimmed of a
trailing newline, and it is an error to set both or to point `_FILE` at a file that
resolves to nothing. There is no secret manager client in this binary: an OpenBao
agent, External Secrets, a Kubernetes secret mount or a plain `.env` file are all
interchangeable, as long as one of them lands the value in the right place before the
process starts. `CSRF_KEY` is the only variable with no usable default; every other one
does, so `bin/website` otherwise runs with none of them set.
It serves the site at `/de/...` and `/en/...`, redirects `/` to the visitor's preferred
locale, and serves `/robots.txt` and `/sitemap.xml` built from the same route table the
prerenderer uses. `/healthz`, `/readyz` and `/metrics` answer on a second, administrative
listener (`ADMIN_ADDR`), never on this public one; see the Configuration table below.

## Configuration

| Variable | Purpose | Default | Required |
| --- | --- | --- | --- |
| `LISTEN_ADDR` | Address the public HTTP server listens on | `:8080` | no |
| `ADMIN_ADDR` | Address govue's admin listener listens on: `/healthz`, `/readyz` and `/metrics` | `:8081` | no |
| `SITE_URL` | The site's own canonical origin, used for SEO tags and the sitemap | `https://ostendorf.tech` | no |
| `LOG_LEVEL` | `debug`, `info`, `warn` or `error` | `info` | no |
| `TRUSTED_PROXIES` | Comma separated CIDRs or addresses trusted to set `X-Forwarded-For` | none | no |
| `ALLOW_PLAIN_HTTP` | Drops the session cookie's `Secure` attribute, for a plain HTTP dev run | `false` | no |
| `RATE_LIMIT_BURST` | Requests one client address may make before waiting | `120` | no |
| `RATE_LIMIT_PER` | The burst's refill period | `1m` | no |
| `CSRF_KEY` (+ `_FILE`) | Hex encoded, at least 32 bytes. Signs the contact form's CSRF token | none | **yes** |
| `CHALLENGE_KEY` (+ `_FILE`) | Hex encoded, at least 32 bytes. Signs the proof of work challenge | none | no\* |
| `CHALLENGE_KEY_PREVIOUS` (+ `_FILE`) | Hex encoded, at least 32 bytes. The key `CHALLENGE_KEY` was just rotated out of; see "Rotating the challenge key" below | none | no |
| `SMTP_ADDR` | SMTP submission server the contact form sends through | `mail.ostendorf.cloud:587` | no |
| `SMTP_USERNAME` (+ `_FILE`) | SMTP username | none | no\* |
| `SMTP_PASSWORD` (+ `_FILE`) | SMTP password | none | no\* |
| `CONTACT_TO` | Recipient address of every contact mail | `simon@ostendorf.tech` | no |
| `CONTACT_FROM` | Envelope sender address of every contact mail | `ostendorf.tech <no-reply@ostendorf.cloud>` | no |
| `CHALLENGE_RATE_LIMIT_BURST` | Challenge requests one client address may make before waiting | `10` | no |
| `CHALLENGE_RATE_LIMIT_PER` | That burst's refill period | `20s` | no |
| `CONTACT_RATE_LIMIT_BURST` | Contact submissions one client address may make before waiting | `5` | no |
| `CONTACT_RATE_LIMIT_PER` | That burst's refill period | `10m` | no |
| `FEED_SERVICE_URL` | Origin of the feed service the Impressionen section is loaded from. Empty renders no section at all | none | no |
| `FEED_REFRESH_INTERVAL` | How often the website asks the feed service for the post list, at least `1m` | `5m` | no |
| `UMAMI_URL` | Base URL of the Umami page views are reported to. `https`, or `http` only for localhost or a name inside the cluster. Empty counts nothing | none | no\*\* |
| `UMAMI_WEBSITE_ID` | UUID of the website in Umami | none | no\*\* |
| `UMAMI_TIMEOUT` | Bound of one post to Umami, `1s` to `30s` | `5s` | no |
| `UMAMI_QUEUE_SIZE` | Page views that may wait for a worker, `1` to `10000`; a full queue drops the view | `256` | no |
| `MAIL_RATE_LIMIT_BURST` | Contact mails this service sends in total, across every client address, before waiting | `50` | no |
| `MAIL_RATE_LIMIT_PER` | That burst's refill period | `1h` | no |

\* Without `CHALLENGE_KEY`, or without both `SMTP_USERNAME` and `SMTP_PASSWORD`, the
contact form answers "unavailable" instead of failing to start; every other page keeps
working. `CSRF_KEY` has no such fallback: the server refuses to start without it.

\*\* `UMAMI_URL` and `UMAMI_WEBSITE_ID` are set together or not at all; exactly one of
them is refused at startup rather than silently counting nothing.

The Impressionen section and the visitor statistic are both off until configured and
both run server side only. The production site counts page views with Umami, which the
owner runs on their own hardware in Germany; the privacy policy states this plainly. Without `FEED_SERVICE_URL` no cache is loaded and the section
is not rendered; the rest of the page is unaffected, and a feed service that is
unreachable or empty never changes what a visitor sees, the same way an unconfigured URL
does. The posts and their pictures are fetched by this server, from the feed service, and
served from this domain, so a visitor's browser never connects to the feed service,
Instagram or Meta. Without `UMAMI_URL` and `UMAMI_WEBSITE_ID` no page view is reported;
with them, this server reports each view to Umami itself, with no script, cookie or
beacon in the visitor's browser, and a failure reaching Umami never affects the response.
The counting is govue's `analytics` package. It reports the route pattern of a served
page (never the path or a query string), the origin of the referrer, the browser
language, the user agent and the client address shortened to its /24 (IPv4) or /48
(IPv6). It leaves out Do Not Track, Global Privacy Control, prefetch, bots, private
client addresses, redirects, errors, assets and the API. Behind a proxy it needs
`TRUSTED_PROXIES`: without it every request comes from the proxy's private
address and nothing is counted (the startup log says so). The variable names did not
change when the site moved to govue's package, so an existing deployment keeps working.
Two things are stricter than before: plain `http` is accepted only for localhost or a
name inside the cluster, and an invalid value stops the start with a message naming
the variable.

This binary has no Instagram or Meta credential and no OpenBao client: both live in the
feed service instead, a second binary (`cmd/feed`) with its own configuration and its own
chart entry, reached over `FEED_SERVICE_URL` alone.

`CONTACT_RATE_LIMIT_BURST` bounds one client address; `MAIL_RATE_LIMIT_BURST` bounds the mail
account itself, across every address at once, which the per-client limit alone cannot do: a
client address is an IPv6 /64, and a single routed /48 is 65,536 of them. The default, 50 an
hour, is well above anything one event technology sole trader's contact form is ever expected to
receive legitimately. Exceeding it answers the same "too many requests" message the per-client
limit gives, and is logged at warning level, since it means either an attack or a genuine surge
worth the operator's attention, unlike an ordinary per-client throttle.

### Rotating the challenge key

Set `CHALLENGE_KEY_PREVIOUS` to the key `CHALLENGE_KEY` previously held, in the same change that
moves `CHALLENGE_KEY` to the new one. A challenge issued just before the rotation, and still
sitting unsolved in a visitor's browser, verifies against either key; only `CHALLENGE_KEY` is
ever used to issue a new one. Remove `CHALLENGE_KEY_PREVIOUS` again once the challenge TTL (10
minutes) has safely passed. Leaving it unset, as before, means rotating `CHALLENGE_KEY` invalidates
every challenge already issued the moment the new key takes effect.

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
cp .env.example .env   # fill in CSRF_KEY at least; see .env.example
docker compose up --build
```

Builds the multi-stage `Dockerfile` (frontend, prerender, static Go binary, then a
minimal non-root runtime stage with no shell) and serves the site at
`http://localhost:8080`. `docker-compose.yaml` is for local development only; see
`charts/ostendorf.tech/README.md` for a cluster installation.

`scripts/smoke-test-image.sh <image-ref>` starts an already built image the same way,
with a throwaway `CSRF_KEY`, and asserts its admin listener (`ADMIN_ADDR`, `:8081`)
answers `/healthz`, `/readyz` and `/metrics`, its public listener serves the home page in
each locale with its prerendered content intact, `/robots.txt` and `/sitemap.xml`, and
that the public listener itself no longer answers `/metrics`. The `docker-build` job in
`.github/workflows/checks.yaml` runs it against the image it just built, so the container
has started and served a real request at least once before anyone ever runs it in the
cluster.

The same `Dockerfile` also builds the feed service (`cmd/feed`) as a second, separate
image, `docker build --target feed`, rather than a second Dockerfile: the two binaries
share `go.mod`/`go.sum` and the pinned base image digests, and BuildKit never evaluates a
stage that the requested target does not depend on, so building `--target feed` never
runs the frontend stage above at all. `docker build` with no `--target` still resolves to
the website image, the last stage in the file.

`scripts/smoke-test-feed-image.sh <image-ref>` starts that image the same way, with
`OPENBAO_ADDR` pointed at a host that does not resolve, and asserts what is honestly
provable without a real OpenBao and a real Meta access token: the process stays up,
`/healthz` answers 200 throughout, `/readyz` stays 503 throughout (this service never
reports ready without a usable token and a synced feed, see "Configuration" above), and
the posts and image routes already answer correctly off an empty, unsynced feed. It does
not and cannot prove that this service ever reaches `/readyz` 200, or that OpenBao's
Kubernetes auth actually admits its role; both need a real cluster. The `docker-build-feed`
job in `.github/workflows/checks.yaml` runs it the same way `docker-build` runs the
website's own script.

## Development

```sh
task setup
task check
```

`task check` runs every lint, build and test target one after another; `task --list`
shows them individually. `task web:build` and `task prerender` have to run before
`task build` or `task test:go` can compile `cmd/website`, since it embeds both of
their outputs; `task build` and `task test:go` already declare that dependency, so
running either one through Task is enough.

Releases are driven by [release-me](https://github.com/ostendorf-cloud/release-me):
Conventional Commits decide the version, and merging the release pull request it
maintains cuts the release. `CHANGELOG.md` is generated, never hand edited.
