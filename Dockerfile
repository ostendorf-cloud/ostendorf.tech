# syntax=docker/dockerfile:1

# Frontend stage: builds the Vue SPA bundle and the prerendered HTML
# straight into cmd/website/dist and cmd/website/prerendered, the same
# paths "task web:build" and "task prerender" write to locally; the Go
# build stage below only ever copies those two directories out, never
# runs Node itself.
#
# Digest for node:26.9.0-bookworm-slim (linux/amd64), resolved against
# Docker Hub's API; re-resolve with
# "docker buildx imagetools inspect node:26.9.0-bookworm-slim" if this
# patch version ever moves.
FROM node:26.9.0-bookworm-slim@sha256:1778ed6290208f56fdb5203b01d7975d0a16cc1549d806c2e44cb99bc96aa377 AS frontend

WORKDIR /src/web

# corepack no longer ships with Node: the Node.js TSC voted to stop
# bundling it by default starting with Node.js 25 (nodejs/corepack#722),
# and this image's Node 26 reflects that; npm still ships, so npm installs
# the pinned pnpm directly instead. The version below is the exact one
# mise.toml's "pnpm" tool resolves to, so the lockfile this stage reads
# was written by the same pnpm a local "task setup" installs.
RUN npm install --global pnpm@10.34.7

COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY web/ ./

# The frontend reads the one route table out of internal/routes, through
# web/src/lib/routeTable.ts, so this stage needs that file as well as web/
# itself. It lives outside web/ because Go embeds the same file; see
# AGENTS.md on why there is only one of it.
COPY internal/routes/routes.json /src/internal/routes/routes.json

RUN pnpm run build && pnpm run prerender

# Build stage: compiles a fully static binary (CGO_ENABLED=0), so the
# runtime stage below needs no libc and can be a minimal, non-shell image.
#
# Digest for golang:1.27.3-bookworm (linux/amd64), matching mise.toml's
# pinned Go patch exactly; resolved against Docker Hub's API.
FROM golang:1.27.3-bookworm@sha256:025eefc1e8dbe63201734a72aaa0a55d7a077f2d8410432f8ae2f3e939f32871 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
# go:embed fails on an empty directory (see AGENTS.md's "A build needs the
# frontend first"), so the frontend's output has to exist before this
# COPY, same as it does before a local "go build".
COPY --from=frontend /src/cmd/website/dist cmd/website/dist
COPY --from=frontend /src/cmd/website/prerendered cmd/website/prerendered

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/website ./cmd/website

# Runtime stage: distroless/static-debian12 carries only a minimal root
# filesystem, ca-certificates and tzdata, nothing else, no shell, no
# package manager. ca-certificates is what this image needs it for: the
# contact form's SMTP sender verifies the mail server's certificate
# (internal/contact/smtp.go) against the OS trust store at connect time.
# tzdata is not needed from the image itself; internal/contact/message.go
# imports "time/tzdata" to embed Europe/Berlin into the binary directly.
# "scratch" was considered and rejected for exactly that ca-certificates
# gap. The ":nonroot" tag already runs as uid/gid 65532, not root.
#
# Digest for gcr.io/distroless/static-debian12:nonroot (linux/amd64),
# resolved against gcr.io's registry API; re-resolve with
# "docker buildx imagetools inspect gcr.io/distroless/static-debian12:nonroot"
# if this needs refreshing.
FROM gcr.io/distroless/static-debian12:nonroot@sha256:52dcfbabb7457ea47c82f6e13af8c8a4a1d9f7b0145142b3ecab20f2b888411d

COPY --from=build /out/website /usr/local/bin/website

# Matches LISTEN_ADDR's default (":8080", see README.md's Configuration
# table). Override LISTEN_ADDR and this EXPOSE together if this default
# port does not fit a deployment.
EXPOSE 8080

# Numeric, not "nonroot:nonroot": the same uid/gid the base image's own
# nonroot user already resolves to, but numeric form is what every
# container runtime can resolve without reading /etc/passwd, redundant
# here since the base image already defaults to it, kept explicit anyway.
USER 65532:65532

ENTRYPOINT ["/usr/local/bin/website"]
