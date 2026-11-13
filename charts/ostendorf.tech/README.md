# ostendorf-tech

Installs [ostendorf.tech](https://github.com/ostendorf-cloud/ostendorf.tech) on a Kubernetes
cluster.

## Prerequisites

- Kubernetes 1.27 or newer.
- A `Secret` already in the target namespace, created by your own secret manager operator,
  carrying at least the CSRF signing key (see "Secrets" below). This chart never creates that
  `Secret` itself.

## Installing

```sh
helm install ostendorf-tech charts/ostendorf.tech \
  --set existingSecret.name=ostendorf-tech-secrets
```

The chart has no usable default for `existingSecret.name`: no name could ever point at a
`Secret` that actually exists on your cluster. Leaving it empty fails `helm install`,
`helm upgrade` and `helm template` at render time, with a message naming `existingSecret.name`
and why it is needed, rather than installing a `Deployment` that crash-loops on a missing
`CSRF_KEY`.

`config.trustedProxyCidrs` must be set once `ingress.enabled` is true. Left empty behind this
chart's own `Ingress`, every request is attributed to the ingress controller's pod IP, which
collapses every rate limiter into one shared bucket for the whole site instead of one per
visitor. `templates/deployment.yaml` rejects an install or template render that leaves this
empty while `ingress.enabled` is true, with a message naming the consequence.

This chart is also published as an OCI artifact, at
`oci://ghcr.io/ostendorf-cloud/charts/ostendorf-tech`, versioned in lock-step with every release:

```sh
helm install ostendorf-tech oci://ghcr.io/ostendorf-cloud/charts/ostendorf-tech --version <version> \
  --set existingSecret.name=ostendorf-tech-secrets
```

or pull it without installing:

```sh
helm pull oci://ghcr.io/ostendorf-cloud/charts/ostendorf-tech --version <version>
```

## Secrets

Every secret-shaped setting is read from one existing `Secret`, named by
`existingSecret.name`, key by key. This chart never offers a values-driven way to create its
own `Secret` from plaintext values: the `Secret` itself comes from your own secret manager
operator (OpenBao, External Secrets, SOPS, or similar), and this chart only ever references it
by name.

| Env var | `values.yaml` key | Default key in the Secret | Required |
| --- | --- | --- | --- |
| `CSRF_KEY` | `existingSecret.csrfKeyKey` | `CSRF_KEY` | Yes |
| `CHALLENGE_KEY` | `existingSecret.challengeKeyKey` | unset (omitted) | No |
| `CHALLENGE_KEY_PREVIOUS` | `existingSecret.challengeKeyPreviousKey` | unset (omitted) | No |
| `SMTP_USERNAME` | `existingSecret.smtpUsernameKey` | unset (omitted) | No |
| `SMTP_PASSWORD` | `existingSecret.smtpPasswordKey` | unset (omitted) | No |
| `INSTAGRAM_TOKEN` | `existingSecret.instagramTokenKey` | unset (omitted) | No |

`CSRF_KEY` has no fallback: a `Secret` missing it fails the pod at the Kubernetes level
(`CreateContainerConfigError`), and the server would refuse to start without one anyway.
`CHALLENGE_KEY`, `CHALLENGE_KEY_PREVIOUS`, `SMTP_USERNAME` and `SMTP_PASSWORD` left unset omit
the corresponding environment variable entirely, rather than referencing a key that may not
exist in your `Secret`; the contact form answers "unavailable" while `CHALLENGE_KEY` or either
SMTP credential is missing, and every other page keeps working.

## Rotating the challenge key

Set `existingSecret.challengeKeyPreviousKey` to the key `CHALLENGE_KEY` previously pointed at,
in the same render that moves `existingSecret.challengeKeyKey` to the new one. A proof of work
challenge issued just before the rotation is still accepted for as long as both keys are
present; the server only ever issues new challenges under the current key. Remove
`challengeKeyPreviousKey` again once the challenge TTL (10 minutes) has safely passed.

`INSTAGRAM_TOKEN` left unset omits the variable the same way, and the home page's Impressionen
section is then not rendered at all; every other section keeps working. A refreshed token lives
in the pod's memory only, since this application has no secret manager client: rotate the value
in the `Secret` before Meta expires it after 60 days.

Every key is at least 32 random bytes, hex encoded, for example:

```sh
openssl rand -hex 32
```

## Replicas

`replicaCount` stays at 1. `internal/webapp/session.go` keeps the anonymous session the CSRF
protector relies on in govue's in-process `session.MemoryStore`. A second replica, or two pods
briefly coexisting during an ordinary rolling update, can answer a visitor's GET on one pod and
their contact form POST on another pod that never saw that session: `requireSession` then
refuses that POST with 403, correctly from that pod's point of view but wrongly from the
visitor's, since their session is real, just on the other pod. This breaks the contact form for
real visitors; it is an availability problem, not a security hole. `templates/deployment.yaml`
rejects a higher value at render time and runs the Deployment with `strategy.type: Recreate`, so
a rollout never lets two pods coexist either. Raising `replicaCount` above 1 is possible only
once the webapp moves to a shared session store instead of the in-memory one.

`replicaCount: 0` is deliberately still allowed. Stopping the site entirely, for a maintenance
window or to pause the deployment, is a legitimate and intentional use of `replicaCount` that
the single-pod guard above was never meant to prevent: it guards against more than one pod
existing, not against zero, and scaling back up from 0 is a normal start, not a rolling update,
so it raises no second-pod risk either.

## Health, readiness and metrics

`/healthz`, `/readyz` and `/metrics` are served on a second, administrative listener
(`service.adminPort`/`config.adminListenAddr`, default `8081`), never on the public
`service.port`: `templates/deployment.yaml`'s `startupProbe`, `livenessProbe` and
`readinessProbe` all point at the `admin` container port, and `templates/ingress.yaml`
only ever routes to `service.port`. A `ServiceMonitor` or `PodMonitor` (not part of this
chart) can scrape `service.adminPort` directly, in-cluster.

## Values

See `values.yaml` for the full, commented list. The ones most deployments touch:

| Key | Description | Default |
| --- | --- | --- |
| `image.repository` | Container image | `ghcr.io/ostendorf-cloud/ostendorf.tech` |
| `image.tag` | Image tag | `.Chart.AppVersion` |
| `replicaCount` | Pod replicas: 0 (stopped) or 1, never above (see "Replicas" above) | `1` |
| `service.port` | Public Service and container port, the only one the `Ingress` ever routes to | `8080` |
| `service.adminPort` | govue's admin listener: health, readiness and metrics, in-cluster only | `8081` |
| `ingress.enabled` | Create an `Ingress` | `false` |
| `resources` | Pod resource requests/limits | see `values.yaml` |
| `config.siteUrl` | The site's own canonical origin | `https://ostendorf.tech` |
| `config.smtpAddr` | SMTP submission server | `mail.ostendorf.cloud:587` |
| `config.adminListenAddr` | govue's admin listener address (`ADMIN_ADDR`); must match `service.adminPort` | `:8081` |
| `config.trustedProxyCidrs` | Trusted proxy CIDRs for client IP resolution, required when `ingress.enabled` is true (see the note under "Installing" above) | `""` |
| `config.instagramApiVersion` | Pinned Instagram Graph API version | `v23.0` |
| `config.instagramPostCount` | Posts loaded per sync, 1 to 100 | `20` |
| `config.instagramRefreshInterval` | How often the feed is reloaded | `1h` |
| `config.instagramTokenRefreshAge` | How old the access token may get before a refresh, `1d` to `50d` | `7d` |
| `config.umamiUrl` | Base URL of the Umami that page views are reported to, empty counts nothing | `""` |
| `config.umamiWebsiteId` | Lowercase UUID of the website in Umami, set together with `config.umamiUrl` | `""` |
| `existingSecret.name` | Name of the existing `Secret` | `""` (required) |

## Upgrading

No breaking change has shipped yet; `Chart.yaml`'s `version` tracks that going forward.

## Uninstalling

```sh
helm uninstall ostendorf-tech
```

Removes the `Deployment`, `Service`, `ServiceAccount` and `Ingress` (when enabled) this chart
created. The `Secret` `existingSecret.name` points at is never touched: it was never created by
this chart, so it is never removed by it either.
