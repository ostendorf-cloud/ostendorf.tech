# Images, Dockerfile and compose

## Three images

The site runs as three images, released under one tag and deployed together:
`ostendorf-tech-frontend` (`cmd/frontend`), `ostendorf-tech-api` (`cmd/api`) and
`ostendorf-tech-feed` (`cmd/feed`). The Ingress routes `/api` to the API and every other
path to the frontend, so the browser sees one origin: no CORS, `connect-src 'self'`, and
the contact form's `SITE_URL` trusted origin as before.

## Dockerfile and publishing

- `Dockerfile` is the multi-stage build for all three images, as final stages `frontend`,
  `api` and `feed` (`docker build --target <name>`). The binaries share `go.mod`, `go.sum`
  and the pinned base image digests. Only `frontend` runs the Node stage.
- `.github/workflows/publish.yaml` builds each image on a native amd64 and arm64 runner
  and merges one manifest list per image under the release tag.

## docker-compose.yaml

`docker-compose.yaml` is a local development stack.

- `frontend` listens on the one published port (`LISTEN_PORT`) and forwards `/api/` to
  `api` through `FRONTEND_DEV_API_PROXY`, so the browser has one origin as in production.
- One feed service of the owner's choice runs, never both:
  - `fakefeed` (`cmd/fakefeed`, standard library only) under the `feed` profile: sample
    posts with generated pictures on the real feed service's routes.
  - `realfeed`, the real `cmd/feed` (token method, `OPENBAO_TOKEN` only a pass-through
    from `.env`) under the `realfeed` profile.
- `FEED_SERVICE_URL` in `.env` names the one the API reads.
- `fakefeed` is a fourth `Dockerfile` target that no published image depends on and no
  workflow builds.
- `internal/envsync` proves both, that each service's port agrees across the compose
  file, the program and `.env.example`, and that the dev proxy points at the API.
