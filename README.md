# ostendorf.tech

The business website of Simon Ostendorf Veranstaltungstechnik: a German and English,
light and dark themed site for an event technology sole trader in Aachen, Germany.

## What it does

A Go service serves a Vue 3 single page application built on the organisation's own
`govue` framework. Every public route is also prerendered to static HTML at build
time, so a crawler or a browser's first paint gets real markup before the SPA takes
over.

This milestone covers the home page's hero, Leistungen, Bands and Über mich sections,
and the imprint, privacy and AGB pages, each in German (the source of truth) and
English. See `AGENTS.md` for what is deliberately left out of this milestone.

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

The server reads its configuration from the environment; every variable below has a
working default, so `bin/website` runs with none of them set. It serves the site at
`/de/...` and `/en/...`, redirects `/` to the visitor's preferred locale, and answers
`/healthz`, `/readyz` and `/metrics`.

## Configuration

| Variable | Purpose | Default | Required |
| --- | --- | --- | --- |
| `LISTEN_ADDR` | Address the HTTP server listens on | `:8080` | no |
| `SITE_URL` | The site's own canonical origin, used for SEO tags | `https://ostendorf.tech` | no |
| `LOG_LEVEL` | `debug`, `info`, `warn` or `error` | `info` | no |
| `TRUSTED_PROXY_CIDRS` | Comma separated CIDRs or addresses trusted to set `X-Forwarded-For` | none | no |
| `RATE_LIMIT_BURST` | Requests one client address may make before waiting | `120` | no |
| `RATE_LIMIT_PER` | The burst's refill period | `1m` | no |

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
