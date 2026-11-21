# Splitting the website into frontend, backend and feed workloads

Status: proposal, not decided. Tracked in issue #104. The owner decides.

## Problem

- Today `cmd/website` is one Go binary. It serves the prerendered HTML, the SPA assets, the API and
  the contact form.
- The binary holds the CSRF key, the challenge key and the SMTP credentials. A bug in a page handler
  runs next to them.
- Pages and API cannot scale or restart apart. The session sits in `session.MemoryStore`, so the
  whole Deployment is pinned to one replica with `Recreate`.
- Every page request therefore shares one pod with the contact form, and every rollout takes the
  pages down briefly.
- The feed is already a second binary (`cmd/feed`) with its own Deployment, Service, ServiceAccount
  and NetworkPolicy. It is the only part that holds the Instagram and OpenBao credentials.
- The owner wants the same shape on every website: frontend, backend and Instagram agent as separate
  pods.
- Other sites (ostendorf.dev, simon-ostendorf.de, qarr.de, burnlink.de, wishlist, cv) were not
  checked. Whether they have an agent or a feed is unknown.
- Open owner question: must the frontend stay prerendered and static, or can it be a full Vue app
  next to a Go backend?

## Goals

- Each workload has its own image, Deployment, probes, resource limits and rights.
- The frontend pod holds no secret and no credential.
- The backend pod serves no HTML and has the smaller attack surface.
- Every response keeps govue's security headers and a CSP without `unsafe-inline`.
- `govue-check-legal` still runs over the prerendered output and fails the build on a missing legal
  page.
- The browser keeps one origin, so the `__Host-session` cookie works and no CORS is needed.
- The local compose stack starts all workloads, and one `LISTEN_ADDR` variable still sets the
  published port.
- Other sites can adopt the split without copying the server code seven times.

## Non goals

- No change to the page content, the routes or the legal texts.
- No move to server side rendering at request time.
- No shared session store. The backend stays at one replica with `Recreate` until that exists.
- No change to the feed service itself.
- No decision about other repositories. They are not checked here.
- No decision about moving the contact form to govue. That is a separate item of the same issue.

## Terms

- Frontend: serves `dist/` (SPA assets, favicons), the `prerendered/` HTML, `robots.txt`,
  `sitemap.xml`, the locale redirect and the legal aliases.
- Backend: serves `/api/v1/challenge`, `/api/v1/contact` and `/api/v1/instagram`. It owns the
  session, CSRF, proof of work and SMTP.
- Agent: today `cmd/feed`. It fetches from Meta and OpenBao. Only the website reads from it.
- Prerendered means static HTML per route and locale, written by `task prerender`. It is served byte
  for byte today.

## Facts verified in this repository

- `cmd/website/embed.go` embeds `dist` and `prerendered`. A build needs both first.
- `internal/webapp` serves pages, `/assets/`, public files, `robots.txt`, `sitemap.xml`, the API and
  the legal aliases from one mux.
- One public listener carries govue's `DefaultSecurity`, rate limit, access log and CSRF. A second
  admin listener carries health, readiness and metrics.
- The theme boot script is inline in every prerendered page. The CSP allows it through a `sha256`
  source computed from the embedded HTML at startup, not a nonce.
- govue's own docs state the default is a nonce injected at serve time through `themeboot`. A static
  `index.html` cannot carry one.
- Page views are counted server side by govue's `analytics` middleware, which reads the mux route
  pattern.
- The Dockerfile has two final stages, `website` and `feed`. A `feed` build skips the Node stage.
- The chart renders one Deployment per binary, one Ingress or HTTPRoute, and a NetworkPolicy for the
  feed only.
- The compose file runs the website alone and publishes a fixed `127.0.0.1:8080:8080`.

## Approach

- Not chosen yet. Three options follow, with a recommendation at the end.

### Option A: stay at website plus feed

- Keep `cmd/website` as is. Two images, two Deployments.
- Cost: none now. The contact backend and the page server stay in one pod.
- Buys: nothing new. The one real secret holder, the feed, is already apart.
- Real reason to drop: the website pod still holds SMTP and signing keys, and pages share its single
  replica.

### Option B: second small Go binary on govue, plus backend and feed

- `cmd/frontend` embeds `dist/` and `prerendered/` and serves them with govue's security headers. It
  holds no secret.
- `cmd/backend` embeds nothing, serves the API and contact form, and reads the feed. It builds
  without the Node stage, like `cmd/feed`.
- Ingress routes `/api` to the backend and `/` to the frontend. Same origin, so no CORS and the host
  only cookie keeps working.
- Prerendering stays. Search engines and first paint keep real markup, and `govue-check-legal` stays
  in the build.
- The CSP hash for the theme boot script stays on the frontend, computed from the HTML it serves.
  The backend sends JSON with the strict default headers.
- Image cost: three images, one Dockerfile with three final stages. Only the frontend stage needs
  Node.
- Chart cost: a third Deployment and Service, a second route rule, three sets of probes and resource
  values, a NetworkPolicy for backend and frontend.
- NetworkPolicy cost: the backend admits only the Ingress controller. The feed admits only the
  backend, not the frontend. Both rules change.
- Session cost: the backend keeps one replica and `Recreate`. The frontend may run two replicas with
  `RollingUpdate`, so page requests survive a backend rollout.
- Rate limit cost: the page limiter is per pod in memory. Two frontend replicas halve its effect.
- Analytics cost: the Umami middleware and `TRUSTED_PROXIES` move to the frontend, since it sees the
  page requests.
- Version coupling: both binaries ship from one repository and one tag. Skew appears only inside a
  rollout, so the API stays backward compatible for one release.
- Compose cost: three services and no Ingress. Local path routing needs a reverse proxy, or the
  frontend proxies `/api`. Either is a design question.
- `LISTEN_ADDR` rule: one variable per published port no longer maps to one service. Compose needs a
  single published port with a fixed internal split.
- Duplication: each of the seven repositories would carry a frontend server, a backend and a chart.
  The frontend server is the same code everywhere.
- Buys: separate scaling and rights, a secret free static pod, a backend with no HTML surface.
- Real reason to drop: it is a third binary and a second route per site, for a site with one small
  contact form and low traffic.

### Option C: a general static web server pod such as nginx

- The frontend pod runs nginx on the `dist/` and `prerendered/` files. The backend and feed are as
  in option B.
- Prerendering stays and is required, since nginx cannot render.
- Cost: security headers are written by hand and drift from govue's `DefaultSecurity` over time.
- Cost: the CSP hash for the theme boot script must be computed at build time and copied into the
  nginx config.
- Cost: the locale redirect, the legal aliases, exact paths without a trailing slash and the 404
  pages must be rebuilt in nginx config.
- Cost: `language`, `themeboot` and `analytics` are Go packages. Page view counting would have no
  home in the frontend pod.
- Cost: a third base image and CVE stream next to distroless, and a second way to configure rate
  limits.
- Buys: no new Go code for the frontend, and the smallest static pod.
- Real reason to drop: it replaces govue's tested header and legal behaviour with hand written
  config. The issue text already names this as the reason against a general web server.

## Effect of prerendering

- Options A and B keep it unchanged. Option C requires it.
- Dropping it for a pure client rendered Vue app would cost first paint, crawler markup and the
  legal page check. That is not proposed.
- A full Vue app next to a Go backend is possible, and B is exactly that. The prerendered HTML is
  the first response of that app, not a competing design.
- The Instagram section is already client rendered from `GET /api/v1/instagram`. It stays so.

## Recommendation

- Stay at option A until two things exist. These are the contact form move to govue and a shared
  static frontend server in govue.
- Then adopt option B across all sites, with the frontend server taken from govue, not copied.
- Trade off: A leaves SMTP and signing keys next to the page server. B removes that, and adds a
  binary, a Deployment and a route per site.
- With the contact form on govue, the backend shrinks and the secret surface changes. Deciding
  before that move may fix the wrong boundary.
- Option C is not recommended.
- The owner decides. No ADR is written, since nothing is settled.

## Open questions

- What must exist in govue first: a shared server package for embedded static frontends with
  `themeboot` hash handling? Next step: owner opens a govue issue.
- Does the frontend proxy `/api` to the backend, or does the Ingress route by path? Next step: owner
  decides, compose needs the same answer.
- Do the other six sites already run an agent or a backend worth splitting? Next step: not checked,
  owner or a later pass reads those repositories.
- Does the wishlist or cv site have an API at all, or only static pages? Next step: not checked,
  same pass.
- Can a shared chart (library chart or one chart with workload toggles) replace seven copies? Next
  step: owner decides after the first site.
- Does the frontend need an own NetworkPolicy, and which egress does it keep? Next step: owner, with
  the chart change.
- Should the page rate limiter move out of the pod for more than one frontend replica? Next step:
  decide with the shared session store question.
- Which version rule keeps frontend and backend compatible during one rollout? Next step: owner,
  before the first release of option B.

## Related

- Issue #104.
- govue `docs/concepts/govue.md`, sections on security headers, `themeboot` and the language layer.
- `AGENTS.md`, the `internal/webapp` and `cmd/feed` bullets.
