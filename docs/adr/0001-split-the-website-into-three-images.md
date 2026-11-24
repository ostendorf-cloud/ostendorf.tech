# 0001. Split the website into three images

Status: accepted

## Context

- The site was one Go binary that served pages, the API and the contact form, next to a second
  binary for the Instagram feed.
- The binary held the signing keys and the SMTP credentials, so a bug in a page handler ran next
  to them.
- Pages and API could not scale, restart or be given rights apart.
- govue offers the pieces for a static frontend server and for a stateless contact form.
- The browser must keep one origin, and every response keeps govue's security headers.
- Prerendered HTML and the build's legal page check must stay.

## Decision

The site is released as three images under one tag: `ostendorf-tech-frontend`,
`ostendorf-tech-api` and `ostendorf-tech-feed`.

- `cmd/frontend` is govue's `frontend` package over the embedded, prerendered `dist/`. It holds no
  secret and serves no API.
- `cmd/api` serves the contact form and `GET /api/v1/instagram`. It embeds no frontend.
- `cmd/feed` is unchanged and keeps the Instagram and OpenBao credentials.
- The Ingress or HTTPRoute sends `/api` to the API and `/` to the frontend, so the browser sees one
  origin and needs no CORS.
- One Dockerfile has three final stages. Only `frontend` runs the Node stage.
- The chart has a Deployment, Service, ServiceAccount and NetworkPolicy per part.

## Consequences

Better:

- The frontend pod holds no secret and may run several replicas with `RollingUpdate`.
- The API pod serves no HTML. Page requests survive an API rollout.
- The API and feed builds skip the Node stage and need no frontend build first.

Worse:

- Three images, three Deployments and a second route rule to run and release.
- The chart values break, see the chart README: the top level values are the API's, the object
  names carry `ostendorf-tech-api`, `ingress.hosts[].paths` is gone, the feed image is renamed from
  `ostendorf.tech-feed` and the API no longer reads `UMAMI_*`.
- An image serves one production origin, since the sitemap and robots.txt carry `VITE_SITE_URL`.
- The API stays at one replica with `Recreate`. govue keeps the contact limits and the proof of
  work replay set in process memory.
- The frontend NetworkPolicy needs an egress rule to Umami while `config.umamiUrl` is set.

Answers to open questions of the concept:

- What govue needs first: its `frontend` package and its stateless contact form now exist.
- Proxy or route by path: the Ingress routes by path. In compose the frontend forwards `/api/` to
  the API through `FRONTEND_DEV_API_PROXY`. The chart never sets it.
- Own NetworkPolicy for the frontend: yes. Ingress from the ingress controller, egress to DNS and
  to Umami only. The API admits the controller. The feed admits only the API.
- NetworkPolicy defaults: `false` for frontend and API, since no default names the ingress
  controller. `true` for the feed, which takes effect once `feed.enabled` is true.
- Session cost: gone. The contact form sets no cookie and takes no CSRF token. The single replica
  stays for the reason above.

Still open:

- The page rate limiter is per pod in memory, so several frontend replicas weaken it.
- The version rule during one rollout is not written down. The images share one tag.
- The other sites, and a shared chart for them, are not checked.

## Alternatives

- Stay at website plus feed: the page pod would still hold SMTP and signing keys and share one
  replica with the pages.
- A general static web server such as nginx: it replaces govue's tested headers, legal aliases
  and language redirect with hand written config. Page counting would have no home in that pod.
- Wait for govue before splitting, as the concept recommended: dropped because the govue
  `frontend` package and contact form were released before the split was built.

## Links

- Concept: `docs/concepts/workload-split.md`.
- Issue #104, issue #119, pull request #153.
- `AGENTS.md`, `README.md` and `charts/ostendorf.tech/README.md` describe the result.
