# cmd/api

`cmd/api` and `internal/webapp` are the API: two HTTP handlers from govue's packages, no
embedded frontend.

- `Handler` is the public one: the security and access log middleware, the client IP
  resolver and the rate limiter (govue's JSON answer), in front of govue's contact
  handlers, `GET /api/v1/instagram` from the feed client and a JSON 404 for every other
  path.
- There is no session manager and no CSRF protector. Every other route answers GET only
  and changes nothing, and the contact handlers check the origin themselves.
- `AdminHandler` serves health, readiness and metrics on the admin listener, reached only
  from inside the cluster, never through the chart's Ingress.
- `internal/headertest` holds the security headers the single image sent before the
  split. The API's tests compare against it, with only the nonce and the boot script
  hash dropped from the policy.
- `contact.go` mounts govue's stateless contact form (see `internal/webapp/AGENTS.md`):
  no cookie, no session, no CSRF token.

## One replica

- govue keeps the contact form's per client limits, its global mail ceiling and its proof
  of work replay set in the memory of one process, so the API Deployment stays at one
  replica.
- A second replica, or two pods briefly coexisting during an ordinary rolling update, has
  its own copy of each. That doubles every budget, the mail ceiling included, and lets a
  solved challenge be used once per pod.
- `charts/ostendorf.tech` enforces the single replica at render time and runs the API
  Deployment with `strategy.type: Recreate`, so a rollout never lets two pods coexist.
- The frontend has no such state and may run several replicas.
- This lifts only once all of them move to a shared store. `pow.Config.Seen` is an
  interface that can take one, but `contact.Config`'s limiters are `*ratelimit.Limiter`,
  a concrete in-memory type, so govue offers no shared limiter. Checked against govue
  v0.11.0, the version in `go.mod`: its `contact` package still documents the limits and
  the replay set as per process.

## No frontend dependency

`cmd/api` embeds nothing. `task build:api` declares no frontend dependency, on purpose,
and `cmd/api`'s own test fails when it gains a `go:embed` directive.
