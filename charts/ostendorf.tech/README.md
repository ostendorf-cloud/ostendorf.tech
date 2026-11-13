# ostendorf-tech

Installs [ostendorf.tech](https://github.com/ostendorf-cloud/ostendorf.tech) on a Kubernetes
cluster.

## Prerequisites

- Kubernetes 1.27 or newer.
- A `Secret` already in the target namespace, created by your own secret manager operator,
  carrying at least the CSRF signing key (see "Secrets" below). This chart never creates that
  `Secret` itself.
- Only if `httpRoute.enabled` is set: the Gateway API CRDs and a running `Gateway` with a
  listener that allows this release's namespace. See "External routing" below; this chart
  ships neither.

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

`CSRF_KEY` has no fallback: a `Secret` missing it fails the pod at the Kubernetes level
(`CreateContainerConfigError`), and the server would refuse to start without one anyway.
`CHALLENGE_KEY`, `CHALLENGE_KEY_PREVIOUS`, `SMTP_USERNAME` and `SMTP_PASSWORD` left unset omit
the corresponding environment variable entirely, rather than referencing a key that may not
exist in your `Secret`; the contact form answers "unavailable" while `CHALLENGE_KEY` or either
SMTP credential is missing, and every other page keeps working.

This chart never handles the Meta access token the feed service needs at all any more: it
lives in OpenBao, read by the feed service itself through the Kubernetes auth method, never
through a Kubernetes `Secret` or a value in this chart. See "Feed service" below.

## Rotating the challenge key

Set `existingSecret.challengeKeyPreviousKey` to the key `CHALLENGE_KEY` previously pointed at,
in the same render that moves `existingSecret.challengeKeyKey` to the new one. A proof of work
challenge issued just before the rotation is still accepted for as long as both keys are
present; the server only ever issues new challenges under the current key. Remove
`challengeKeyPreviousKey` again once the challenge TTL (10 minutes) has safely passed.

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

The feed service has a different, deliberate probe contract, served on its one port
(`feed.service.port`): `/healthz` always passes while the process is up, and `/readyz` fails
until the feed service has actually obtained a usable OpenBao token and a non-empty feed from
Meta. A feed pod that cannot reach OpenBao or Meta therefore sits at `0/1` ready, never
answering readiness traffic, rather than crash-looping: `templates/deployment-feed.yaml`'s
`livenessProbe` points at `/healthz` and its `readinessProbe` at `/readyz`, never the reverse.
This is a deliberate design choice, not a bug: putting the readiness check on the liveness
probe would restart a pod for merely being unready, and putting the liveness check on the
readiness probe would report ready immediately and defeat the whole point of the gate.

## External routing

This chart reaches external traffic to its own `Service` two ways, chosen by
`ingress.enabled` and `httpRoute.enabled`. Both default to `false`: a default install creates
neither, reachable only from inside the cluster, until one is deliberately turned on. They are
mutually exclusive: turning both on fails the install or upgrade with a readable message
naming which one to turn off, enforced once in `templates/deployment.yaml`, the one template
that always renders regardless of which routing flag is set.

### Ingress

Set `ingress.enabled: true`. `ingress.hosts` names the host and path rules, and `ingress.tls`
carries the certificate configuration directly on the `Ingress` object: this chart owns that
configuration. `ingress.enabled: true` also requires `config.trustedProxyCidrs`; see
"Installing" above for why.

### Gateway API HTTPRoute

Set `httpRoute.enabled: true`, `httpRoute.parentRefs.name` and `httpRoute.parentRefs.namespace`
to the `Gateway` this release attaches to. The namespace is required, not optional the way
`ingress.className` is: a `Gateway` is almost always platform owned and lives in a different
namespace than this release, and the Gateway API's own default for an unset
`parentRefs.namespace` is the `HTTPRoute`'s own namespace, normally wrong here.

This chart never creates the `Gateway` and never ships the Gateway API CRDs (`Gateway`,
`HTTPRoute` and the rest): both are cluster-level prerequisites, installed and operated once
per cluster by the cluster operator, same as any other CRD (see `kubernetes-conventions`'s
`install-crds` convention), outside this chart. The `Gateway`'s own `allowedRoutes` decides
whether it actually accepts this `HTTPRoute`; a `httpRoute.enabled: true` release that the
`Gateway` does not allow installs without error but never actually receives traffic.

Unlike `Ingress`, an `HTTPRoute` carries no `tls` field of its own. TLS terminates on the
`Gateway`'s listener instead, which the cluster operator, not this chart, owns and configures;
`httpRoute.hostnames` only has to match a hostname the listener itself already serves for
traffic to reach this `HTTPRoute` at all.

Routing itself is fixed to this chart's own `Service`, on `service.port`, never
`service.adminPort`: no values-driven way to add a second backend or a per-path rule, the same
single-backend assumption `ingress.hosts` already makes. An `HTTPRoute` rule with no `matches`
defaults to a `/` prefix match per the Gateway API spec, the same default an `Ingress` with one
`/` path rule gives, so this chart's `HTTPRoute` sets none explicitly.

`httpRoute.hostnames` is a list because `HTTPRoute` itself accepts more than one hostname for
the one set of rules above; this chart's own site is a single public domain, so it is set up
front to one hostname in practice, not resolved per request.

## Feed service

The home page's Impressionen section is loaded from a second, independent workload: the feed
service (`cmd/feed`), with its own `Deployment` (`templates/deployment-feed.yaml`), `Service`
(`templates/service-feed.yaml`) and `ServiceAccount` (`templates/serviceaccount-feed.yaml`),
separate from the website's own. It needs its own `ServiceAccount` because it authenticates to
OpenBao through the Kubernetes auth method, using that `ServiceAccount`'s own projected token as
the login credential; the website's `ServiceAccount` has no use for that identity and keeps
`automountServiceAccountToken: false` either way.

`feed.config.openBaoAddr` and `feed.config.openBaoRole` are required, with no default: no value
could ever point at an OpenBao installation, or name a role in it, that actually exists on your
cluster. Leaving either empty fails `helm install`, `helm upgrade` and `helm template` at render
time, with a message naming the missing value and why it is needed, the same pattern
`existingSecret.name` already uses above. The role and its OpenBao policy, granting read access
to `feed.config.tokenPath`, are created once, by hand, in OpenBao itself; this chart only
references the role by name.

The feed service's container image, `feed.image.repository` (default
`ghcr.io/ostendorf-cloud/ostendorf.tech-feed`), is built from the same root `Dockerfile` as the
website's own image, as a separate final stage (`docker build --target feed`) rather than a
second Dockerfile, and published by the `feed-image` job in `.github/workflows/publish.yaml`
alongside the website's own `image` job, both tagged from the same release so the two images and
this chart's `Chart.yaml` `appVersion` stay in lock-step.

The website reaches the feed service over `FEED_SERVICE_URL`, computed by
`templates/deployment.yaml` from the feed `Service`'s own name and `feed.service.port`
(`http://<release>-ostendorf-tech-feed:<feed.service.port>` by default). There is no values key
for it: this value must always resolve to this exact release's own feed `Service`, in the same
namespace, so there is no legitimate reason for an installer to point it elsewhere.

## NetworkPolicy

`feed.networkPolicy.enabled` defaults to `true`: this organisation's clusters enforce
`NetworkPolicy`, and the goal is enforcement by the chart itself rather than by convention alone.

This is a correctness warning, not a hedge: a `NetworkPolicy` object does nothing at all on a
cluster whose CNI does not enforce `NetworkPolicy`. Such a cluster accepts and stores the object
without complaint, with no error and no warning either. Confirm your CNI actually enforces
`NetworkPolicy` before relying on this for anything.

The one ingress rule admits traffic only from the website's own pods (matched by
`ostendorf-tech.selectorLabels`, in the same namespace), on `feed.service.port`. It admits no
rule for kubelet probe traffic, and needs none: per the
[Kubernetes documentation](https://kubernetes.io/docs/concepts/services-networking/network-policies/),
"traffic to and from the node where a Pod is running is always allowed, regardless of the IP
address of the Pod or the node". Every `NetworkPolicy`-conformant CNI must honour this, so a
kubelet liveness or readiness probe, which originates from the node, is never subject to this
policy at all.

Three egress rules allow what the feed service actually needs to reach, none scoped by
destination:

- DNS, UDP and TCP port 53. The cluster's DNS `Service`'s own namespace and pod labels are not
  something this chart can know in advance across every cluster it might be installed on, and
  guessing wrong would fail silently at runtime, indistinguishable from a Meta outage, rather
  than at render time. Restricting only the port, not the destination, is the standard, widely
  used exception for exactly this reason.
- OpenBao, on `feed.networkPolicy.openBaoPort` (default `8200`, OpenBao's conventional API port).
  Kept as its own explicit value rather than parsed out of `feed.config.openBaoAddr`'s URL,
  because Helm's templating has no clean way to read a URL's port. Keep this in sync by hand with
  whatever port `feed.config.openBaoAddr` actually names, if it ever differs from the default.
- Meta's Graph API, TCP port 443. Not scoped by destination because Meta serves this from a CDN
  with no fixed, documentable IP range.

## Values

See `values.yaml` for the full, commented list. The ones most deployments touch:

| Key | Description | Default |
| --- | --- | --- |
| `image.repository` | Container image | `ghcr.io/ostendorf-cloud/ostendorf.tech` |
| `image.tag` | Image tag | `.Chart.AppVersion` |
| `replicaCount` | Pod replicas: 0 (stopped) or 1, never above (see "Replicas" above) | `1` |
| `service.port` | Public Service and container port, the only one the `Ingress` ever routes to | `8080` |
| `service.adminPort` | govue's admin listener: health, readiness and metrics, in-cluster only | `8081` |
| `ingress.enabled` | Create an `Ingress`. Requires `config.trustedProxyCidrs` to be set; see that row. Mutually exclusive with `httpRoute.enabled`; see "External routing" above. | `false` |
| `ingress.annotations` | Annotations for the `Ingress` | `{}` |
| `ingress.tls` | TLS configuration for the `Ingress` | `[]` |
| `httpRoute.enabled` | Create a Gateway API `HTTPRoute`. Mutually exclusive with `ingress.enabled`; see "External routing" above. | `false` |
| `httpRoute.hostnames` | Hostnames the `HTTPRoute` matches | `[]` |
| `httpRoute.annotations` | Annotations for the `HTTPRoute` | `{}` |
| `httpRoute.parentRefs.name` | Name of the `Gateway` to attach to. Required when `httpRoute.enabled` is true | `""` |
| `httpRoute.parentRefs.namespace` | Namespace the `Gateway` lives in. Required when `httpRoute.enabled` is true; see "External routing" above for why this has no same-namespace fallback | `""` |
| `httpRoute.parentRefs.sectionName` | Specific listener on the `Gateway` to attach to, when it exposes more than one | `""` |
| `resources` | Pod resource requests/limits | see `values.yaml` |
| `config.siteUrl` | The site's own canonical origin | `https://ostendorf.tech` |
| `config.smtpAddr` | SMTP submission server | `mail.ostendorf.cloud:587` |
| `config.adminListenAddr` | govue's admin listener address (`ADMIN_ADDR`); must match `service.adminPort` | `:8081` |
| `config.trustedProxyCidrs` | Trusted proxy CIDRs for client IP resolution, required when `ingress.enabled` is true (see the note under "Installing" above) | `""` |
| `config.feedRefreshInterval` | How often the website asks the feed service for the current post list | `5m` |
| `config.umamiUrl` | Base URL of the Umami that page views are reported to, empty counts nothing | `""` |
| `config.umamiWebsiteId` | Lowercase UUID of the website in Umami, set together with `config.umamiUrl` | `""` |
| `existingSecret.name` | Name of the existing `Secret` | `""` (required) |
| `feed.image.repository` | Feed service container image; not published yet, see "Feed service" above | `ghcr.io/ostendorf-cloud/ostendorf.tech-feed` |
| `feed.service.port` | Feed service's one container and Service port | `8081` |
| `feed.config.openBaoAddr` | OpenBao address the feed service authenticates against. Required; see "Feed service" above | `""` (required) |
| `feed.config.openBaoRole` | OpenBao Kubernetes auth role the feed service logs in as. Required; see "Feed service" above | `""` (required) |
| `feed.networkPolicy.enabled` | Create a `NetworkPolicy` restricting the feed service's traffic; see "NetworkPolicy" above | `true` |
| `feed.networkPolicy.openBaoPort` | Port OpenBao egress is allowed on; keep in sync with `feed.config.openBaoAddr` | `8200` |

## Upgrading

No breaking change has shipped yet; `Chart.yaml`'s `version` tracks that going forward.

## Uninstalling

```sh
helm uninstall ostendorf-tech
```

Removes the website's `Deployment`, `Service`, `ServiceAccount`, and the `Ingress` or
`HTTPRoute`, whichever was enabled; the feed service's own `Deployment`, `Service`,
`ServiceAccount` and `NetworkPolicy`, all created by this same chart, are removed the same way.
Neither OpenBao nor anything stored in it is touched: this chart never created it, so it is never
removed by it either, the same reasoning as the existing `Secret` `existingSecret.name` points
at.

## Verification

This chart is verified with:

- `helm lint`
- `helm template`, for the default values and for the documented ingress-enabled and
  httpRoute-enabled combinations.
- `kubeconform`, against the rendered manifests. The default schema set carries no Gateway API
  schemas, so an httpRoute-enabled render's `HTTPRoute` is validated with a second
  `-schema-location` added after the default one, pointing at the community
  [CRDs-catalog](https://github.com/datreeio/CRDs-catalog):

  ```sh
  kubeconform -strict -summary \
    -schema-location default \
    -schema-location 'https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json'
  ```

- [`helm-unittest`](https://github.com/helm-unittest/helm-unittest), run with
  `helm unittest charts/ostendorf.tech`; see `mise.toml` for how the plugin is pinned.

`task lint:helm`, `task validate:kubeconform` and `task test:helm` run each of these, one job
per check in CI. A live end-to-end install into a real cluster is a separate, tracked follow-up,
not part of this chart's static verification.
