# syntax=docker/dockerfile:1@sha256:4edf897a3ffa55b89f906fc8cc78afdb3f1834cc9c7083565e611a8a7d5fe99e

# This file builds two separate images, cmd/website and cmd/feed, as two
# named final stages ("website" and "feed") rather than two Dockerfiles.
# The two binaries share go.mod/go.sum and most of internal/, so one file
# keeps the pinned base image digests below, Node included, in exactly one
# place instead of two copies that could drift out of sync with each
# other. "docker build --target website" or "--target feed" (see
# .github/workflows/checks.yaml and publish.yaml) builds only the stages
# that final stage actually depends on: BuildKit never evaluates a stage
# that is not an ancestor of the requested target, so "--target feed"
# never runs the frontend stage below at all. That matters here more than
# it would for two ordinary Go binaries: cmd/feed embeds nothing and needs
# no frontend, so building it against a target that pulled in the Vue
# stage would silently cost minutes of pnpm/vite work and the resulting
# image size for a dependency the binary never uses. Building with no
# "--target" at all, as "docker compose build" and a bare "docker build ."
# both do, resolves to this file's last stage, "website", matching the
# only image this repository published before cmd/feed existed.

# Frontend stage: builds the Vue SPA bundle and the prerendered HTML
# straight into cmd/website/dist and cmd/website/prerendered, the same
# paths "task web:build" and "task prerender" write to locally; the Go
# build stage below only ever copies those two directories out, never
# runs Node itself. Only the "website" stage depends on this one; the
# "feed" stage never triggers it, see this file's own top comment.
#
# Digest for node:26.13.1-bookworm-slim (linux/amd64), resolved against
# Docker Hub's API; re-resolve with
# "docker buildx imagetools inspect node:26.13.1-bookworm-slim" if this
# patch version ever moves.
FROM node:26.13.1-bookworm-slim@sha256:18a166e2d50578afe933434857ed2d9059bd1aa398a223e9c89eb4302354f079 AS frontend

WORKDIR /src/web

# corepack no longer ships with Node: the Node.js TSC voted to stop
# bundling it by default starting with Node.js 25 (nodejs/corepack#722),
# and this image's Node 26 reflects that. npm still ships, so npm installs
# corepack first, pinned the same way every other version in this file
# is (0.36.0 was the latest published version when this was last
# checked); corepack then installs the pinned pnpm version below, the
# exact one mise.toml's "pnpm" tool resolves to, so the lockfile this
# stage reads was written by the same pnpm a local "task setup" installs.
RUN npm install -g corepack@0.36.0 && corepack enable && corepack prepare pnpm@12.19.0 --activate

COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

COPY web/ ./

# The frontend reads the one route table out of internal/routes, through
# web/src/lib/routeTable.ts, so this stage needs that file as well as web/
# itself. It lives outside web/ because Go embeds the same file; see
# AGENTS.md on why there is only one of it.
COPY internal/routes/routes.json /src/internal/routes/routes.json

RUN pnpm run build && pnpm run prerender

# Shared Go stage: downloads the module graph once, for both the
# "build-website" and "build-feed" stages below, so the cache layer this
# produces (and the base image digest it is pinned to) exists in one
# place for both binaries instead of two.
#
# Digest for golang:1.27.4-bookworm (linux/amd64), matching mise.toml's
# pinned Go patch exactly; resolved against Docker Hub's API.
FROM golang:1.27.4-bookworm@sha256:fb7cceb528b47141eccd9b60bf3cbd27d5ef5f8cf5b7d338bec866da495b50dc AS go-deps

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

# Build stage for cmd/website: compiles a fully static binary
# (CGO_ENABLED=0), so the runtime stage below needs no libc and can be a
# minimal, non-shell image.
#
# go:embed fails on an empty directory (see AGENTS.md's "A build needs the
# frontend first"), so the frontend's output has to exist before this
# COPY, same as it does before a local "go build".
FROM go-deps AS build-website

COPY cmd/ cmd/
COPY internal/ internal/
COPY --from=frontend /src/cmd/website/dist cmd/website/dist
COPY --from=frontend /src/cmd/website/prerendered cmd/website/prerendered

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/website ./cmd/website

# Build stage for cmd/feed: a fully static binary, same reasoning as
# build-website above. cmd/feed embeds nothing (no go:embed directive
# anywhere in internal/feed or internal/feedconfig) and needs no frontend
# at all, so this stage copies only cmd/feed/ rather than the whole cmd/
# tree, and never references the frontend stage above; a build with
# "--target feed" therefore never runs it.
FROM go-deps AS build-feed

COPY cmd/feed/ cmd/feed/
COPY internal/ internal/

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/feed ./cmd/feed

# Runtime stage for cmd/feed: distroless/static-debian12 carries only a
# minimal root filesystem, ca-certificates and tzdata, nothing else, no
# shell, no package manager. ca-certificates is what this image needs it
# for: internal/openbao's client and internal/instagram's Meta client both
# make outbound TLS connections (OpenBao's API and the Graph API),
# verified against the OS trust store. tzdata is unused by this binary:
# cmd/feed and the packages it imports have no need for time zones at
# all, let alone the "time/tzdata" import cmd/website's own runtime stage
# uses below. "scratch" was considered and rejected for the
# ca-certificates gap alone. The ":nonroot" tag already runs as uid/gid
# 65532, not root.
#
# Digest for gcr.io/distroless/static-debian12:nonroot (linux/amd64),
# resolved against gcr.io's registry API; re-resolve with
# "docker buildx imagetools inspect gcr.io/distroless/static-debian12:nonroot"
# if this needs refreshing.
FROM gcr.io/distroless/static-debian12:nonroot@sha256:52dcfbabb7457ea47c82f6e13af8c8a4a1d9f7b0145142b3ecab20f2b888411d AS feed

COPY --from=build-feed /out/feed /usr/local/bin/feed

# Matches feedconfig.DefaultListenAddr (":8081", see internal/feedconfig
# and charts/ostendorf.tech/README.md's "Health, readiness and metrics"
# section). Unlike the website, there is no second, administrative
# listener: internal/feed.App.buildHandler serves /healthz, /readyz and
# the API routes on this one port, so there is only one EXPOSE line here.
EXPOSE 8081

# Numeric, not "nonroot:nonroot": the same uid/gid the base image's own
# nonroot user already resolves to, but numeric form is what every
# container runtime can resolve without reading /etc/passwd, redundant
# here since the base image already defaults to it, kept explicit anyway.
USER 65532:65532

ENTRYPOINT ["/usr/local/bin/feed"]

# Runtime stage for cmd/website: same distroless base and the same
# ca-certificates/no-shell reasoning as cmd/feed's own runtime stage
# above. ca-certificates is what this image needs it for here: the
# contact form's SMTP sender verifies the mail server's certificate
# (internal/contact/smtp.go) against the OS trust store at connect time.
# Unlike cmd/feed, this binary does care about a time zone, but tzdata is
# still not needed from the image itself: internal/contact/message.go
# imports "time/tzdata" to embed Europe/Berlin into the binary directly,
# so the base image's own bundled tzdata ends up just as unused here as
# it is for cmd/feed, for a different reason.
#
# This is also why a bare "docker build ." or "docker compose build", with
# no "--target", builds this stage: it is deliberately the last one in the
# file, matching the only image this repository published before cmd/feed
# existed, see this file's own top comment.
FROM gcr.io/distroless/static-debian12:nonroot@sha256:52dcfbabb7457ea47c82f6e13af8c8a4a1d9f7b0145142b3ecab20f2b888411d AS website

COPY --from=build-website /out/website /usr/local/bin/website

# Matches LISTEN_ADDR's and ADMIN_ADDR's defaults (":8080" and ":8081", see
# README.md's Configuration table): the public listener and govue's own admin
# listener (health, readiness, metrics). Override the two env vars and these two
# EXPOSE lines together if either default port does not fit a deployment.
EXPOSE 8080
EXPOSE 8081

# Numeric, not "nonroot:nonroot"; see cmd/feed's own runtime stage comment
# above for why.
USER 65532:65532

ENTRYPOINT ["/usr/local/bin/website"]
