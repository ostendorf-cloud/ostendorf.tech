# cmd/feed

`cmd/feed` wires the feed service: a third binary alongside `cmd/frontend` and
`cmd/api`, with its own configuration (`internal/feedconfig`) and its own chart entry
(`charts/ostendorf.tech/templates/deployment-feed.yaml` and friends).

## Configuration

- `OPENBAO_AUTH_METHOD` is `kubernetes` (default, in a cluster) or `token` (local tests,
  `OPENBAO_TOKEN` or its `OPENBAO_TOKEN_FILE` twin).
- The variables of the other method are refused at startup, and no error quotes a
  refused value. The service never renews a static token.
- `OPENBAO_SECRET_PATH` is the KV v2 data path of the secret.
- `OPENBAO_SECRET_KEY_ACCESS_TOKEN` and `OPENBAO_SECRET_KEY_REFRESHED_AT` (defaults
  `access_token` and `refreshed_at`) name the two keys inside it. A refresh rewrites only
  those two and keeps every other key.
- `POST_COUNT` caps the number of posts. 0 means the maximum of 10000. Default 200.
- `POSTS_PER_REQUEST` sets the page size. Default 50, range 1 to 100.
- Every post costs its picture in memory. See "Posts and memory" in
  `charts/ostendorf.tech/README.md`, where the chart's `goMemLimit` and
  `feed.goMemLimit` set `GOMEMLIMIT` below each memory limit.

## Service

`internal/feed` builds it: the background loop that keeps `internal/instagram`'s `Feed`
current from Meta using the OpenBao-backed token, and the internal-only HTTP surface the
API reads from.

- Routes: `GET /api/v1/posts` and `GET /api/v1/posts/{id}/image`, alongside `/healthz`
  and `/readyz` on the one listener.
- `/healthz` always answers 200 while the process is up.
- `/readyz` answers 503 until one full sync cycle has obtained a usable token and
  completed a call to Meta. It then stays 200 even through a later transient failure,
  since the last good feed keeps being served regardless.
- A cycle that reaches Meta and finds zero posts still counts. An Instagram account with
  nothing posted is a real, permanently empty feed, not a readiness failure.
- A feed pod that cannot reach OpenBao or Meta sits at `0/1` ready, retrying, rather
  than crash-looping or serving stale or absent data as if nothing were wrong.
- Its HTTP surface authenticates no caller of its own. It relies on the chart's
  `NetworkPolicy` admitting only the API's pods, see `charts/ostendorf.tech/README.md`.
- It needs no frontend build: `cmd/feed` embeds nothing, see the root `AGENTS.md`.
