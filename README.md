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
locale, answers `/healthz`, `/readyz` and `/metrics`, and serves `/robots.txt` and
`/sitemap.xml` built from the same route table the prerenderer uses.

## Configuration

| Variable | Purpose | Default | Required |
| --- | --- | --- | --- |
| `LISTEN_ADDR` | Address the HTTP server listens on | `:8080` | no |
| `SITE_URL` | The site's own canonical origin, used for SEO tags and the sitemap | `https://ostendorf.tech` | no |
| `LOG_LEVEL` | `debug`, `info`, `warn` or `error` | `info` | no |
| `TRUSTED_PROXY_CIDRS` | Comma separated CIDRs or addresses trusted to set `X-Forwarded-For` | none | no |
| `ALLOW_PLAIN_HTTP` | Drops the session cookie's `Secure` attribute, for a plain HTTP dev run | `false` | no |
| `RATE_LIMIT_BURST` | Requests one client address may make before waiting | `120` | no |
| `RATE_LIMIT_PER` | The burst's refill period | `1m` | no |
| `CSRF_KEY` (+ `_FILE`) | Hex encoded, at least 32 bytes. Signs the contact form's CSRF token | none | **yes** |
| `CHALLENGE_KEY` (+ `_FILE`) | Hex encoded, at least 32 bytes. Signs the proof of work challenge | none | no\* |
| `SMTP_ADDR` | SMTP submission server the contact form sends through | `mail.ostendorf.cloud:587` | no |
| `SMTP_USERNAME` (+ `_FILE`) | SMTP username | none | no\* |
| `SMTP_PASSWORD` (+ `_FILE`) | SMTP password | none | no\* |
| `CONTACT_TO` | Recipient address of every contact mail | `simon@ostendorf.tech` | no |
| `CONTACT_FROM` | Envelope sender address of every contact mail | `ostendorf.tech <no-reply@ostendorf.cloud>` | no |
| `CHALLENGE_RATE_LIMIT_BURST` | Challenge requests one client address may make before waiting | `10` | no |
| `CHALLENGE_RATE_LIMIT_PER` | That burst's refill period | `20s` | no |
| `CONTACT_RATE_LIMIT_BURST` | Contact submissions one client address may make before waiting | `5` | no |
| `CONTACT_RATE_LIMIT_PER` | That burst's refill period | `10m` | no |
| `INSTAGRAM_TOKEN` (+ `_FILE`) | Instagram access token the Impressionen section is loaded with. Empty renders no section at all | none | no |
| `INSTAGRAM_API_VERSION` | Pinned Instagram Graph API version | `v23.0` | no |
| `INSTAGRAM_POST_COUNT` | Posts loaded per sync, 1 to 100 | `20` | no |
| `INSTAGRAM_REFRESH_INTERVAL` | How often the feed is reloaded, at least `1m` | `1h` | no |
| `INSTAGRAM_TOKEN_REFRESH_AGE` | How old the token may get before a refresh, `1d` to `50d` | `7d` | no |
| `UMAMI_URL` | Base URL of the Umami page views are reported to. Empty counts nothing | none | no\*\* |
| `UMAMI_WEBSITE_ID` | Lowercase UUID of the website in Umami | none | no\*\* |

\* Without `CHALLENGE_KEY`, or without both `SMTP_USERNAME` and `SMTP_PASSWORD`, the
contact form answers "unavailable" instead of failing to start; every other page keeps
working. `CSRF_KEY` has no such fallback: the server refuses to start without it.

\*\* `UMAMI_URL` and `UMAMI_WEBSITE_ID` are set together or not at all; exactly one of
them is refused at startup rather than silently counting nothing.

The Impressionen section and the visitor statistic are both switched off by default and
both run server side only. Without `INSTAGRAM_TOKEN` no feed is loaded and the section is
not rendered; the rest of the page is unaffected, and a failure reaching Meta never
changes what a visitor sees. The posts and their pictures are fetched by this server and
served from this domain, so a visitor's browser never connects to Instagram or Meta.
Without `UMAMI_URL` and `UMAMI_WEBSITE_ID` no page view is reported; with them, this
server reports each view to Umami itself, with no script, cookie or beacon in the
visitor's browser, and a failure reaching Umami never affects the response.

A refreshed Instagram token lives in memory only. There is no secret manager client here,
so a restart falls back to whatever `INSTAGRAM_TOKEN` carries: rotate that value before
Meta expires it, 60 days after it was issued.

Behind any reverse proxy or ingress, leaving `TRUSTED_PROXY_CIDRS` unset means every
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
with a throwaway `CSRF_KEY`, and asserts it answers `/healthz`, `/readyz`, the home
page in each locale with its prerendered content intact, `/robots.txt` and
`/sitemap.xml`. The `docker-build` job in `.github/workflows/checks.yaml` runs it
against the image it just built, so the container has started and served a real
request at least once before anyone ever runs it in the cluster.

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
