{{/*
Expand the name of the chart.
*/}}
{{- define "ostendorf-tech.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to
this (by the DNS naming spec).
*/}}
{{- define "ostendorf-tech.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "ostendorf-tech.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
The site is three workloads deployed together, one release and one version tag:
the frontend (static pages), the API (contact form, feed proxy) and the feed
service. Each has its own fullname, selector labels, common labels and
ServiceAccount name. Every selector uses a different app.kubernetes.io/name:
sharing one would make each Service and each NetworkPolicy select the pods of
all of them instead of just their own.
*/}}

{{/*
Release labels of an object that belongs to no single part, the Ingress and
the HTTPRoute, which route to the frontend and the API alike: the release and
the site, no part's name and no component.
*/}}
{{- define "ostendorf-tech.release.labels" -}}
helm.sh/chart: {{ include "ostendorf-tech.chart" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/part-of: {{ include "ostendorf-tech.name" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
API fullname: the release's own fullname with "-api" appended. This is the
workload the chart's top level values (replicaCount, image, resources, probes)
configure, the former single website Deployment.
*/}}
{{- define "ostendorf-tech.api.fullname" -}}
{{- printf "%s-api" (include "ostendorf-tech.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
API selector labels. Deliberately excludes app.kubernetes.io/version: a selector
must stay stable across an image upgrade, matchLabels is immutable once a
Deployment exists.
*/}}
{{- define "ostendorf-tech.api.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ostendorf-tech.name" . }}-api
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
API common labels.
*/}}
{{- define "ostendorf-tech.api.labels" -}}
helm.sh/chart: {{ include "ostendorf-tech.chart" . }}
{{ include "ostendorf-tech.api.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/component: api
app.kubernetes.io/part-of: {{ include "ostendorf-tech.name" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
API ServiceAccount name.
*/}}
{{- define "ostendorf-tech.api.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "ostendorf-tech.api.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{/*
Frontend fullname: the release's own fullname with "-frontend" appended.
*/}}
{{- define "ostendorf-tech.frontend.fullname" -}}
{{- printf "%s-frontend" (include "ostendorf-tech.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Frontend selector labels; see the API's for why they exclude the version.
*/}}
{{- define "ostendorf-tech.frontend.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ostendorf-tech.name" . }}-frontend
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Frontend common labels.
*/}}
{{- define "ostendorf-tech.frontend.labels" -}}
helm.sh/chart: {{ include "ostendorf-tech.chart" . }}
{{ include "ostendorf-tech.frontend.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/component: frontend
app.kubernetes.io/part-of: {{ include "ostendorf-tech.name" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Frontend ServiceAccount name.
*/}}
{{- define "ostendorf-tech.frontend.serviceAccountName" -}}
{{- if .Values.frontend.serviceAccount.create -}}
{{- default (include "ostendorf-tech.frontend.fullname" .) .Values.frontend.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.frontend.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{/*
One NetworkPolicy peer from a dict with the optional keys namespaceSelector,
podSelector (each a map of labels, rendered as matchLabels) and cidr. An empty
dict renders "{}", which callers must not use: they test for a non empty
selector first. namespaceSelector and podSelector in one peer mean "pods with
these labels in namespaces with these labels" (both must match). A cidr
renders an ipBlock peer instead.
*/}}
{{- define "ostendorf-tech.networkPolicy.peer" -}}
{{- $peer := dict -}}
{{- with .cidr -}}
{{- $_ := set $peer "ipBlock" (dict "cidr" .) -}}
{{- end -}}
{{- with .namespaceSelector -}}
{{- $_ := set $peer "namespaceSelector" (dict "matchLabels" .) -}}
{{- end -}}
{{- with .podSelector -}}
{{- $_ := set $peer "podSelector" (dict "matchLabels" .) -}}
{{- end -}}
{{- toYaml $peer -}}
{{- end -}}

{{/*
The ingress controller peer of an http port rule. Called with the part's
networkPolicy values and the part's name for the error message. There is no
default: no chart can know which namespace and labels the ingress controller or
Gateway of a cluster runs with, and a guess would silently drop every visitor.
The namespaceSelector is required; the podSelector narrows it further and may
stay empty, which admits every pod of that namespace.
*/}}
{{- define "ostendorf-tech.networkPolicy.ingressControllerPeer" -}}
{{- $np := index . 0 -}}
{{- $part := index . 1 -}}
{{- $ic := $np.ingressController | default dict -}}
{{- if not $ic.namespaceSelector -}}
{{- fail (printf "%snetworkPolicy.ingressController.namespaceSelector is required when %snetworkPolicy.enabled is true: it names the namespace of the ingress controller or Gateway that may reach the http port. There is no default, since none could fit every cluster." $part $part) -}}
{{- end -}}
{{- include "ostendorf-tech.networkPolicy.peer" (dict "namespaceSelector" $ic.namespaceSelector "podSelector" $ic.podSelector) -}}
{{- end -}}

{{/*
The DNS egress rule of the API and the frontend: UDP and TCP 53, scoped to the
destination in networkPolicy.dns (default kube-dns in kube-system, the same
default and the same caveats as feed.networkPolicy.dns, see values.yaml). Null
both selectors for a port only rule. Called with the part's networkPolicy values.
*/}}
{{- define "ostendorf-tech.networkPolicy.dnsRule" -}}
- ports:
    - protocol: UDP
      port: 53
    - protocol: TCP
      port: 53
  {{- $dns := .dns | default dict }}
  {{- if or $dns.namespaceSelector $dns.podSelector }}
  to:
    - {{- include "ostendorf-tech.networkPolicy.peer" (dict "namespaceSelector" $dns.namespaceSelector "podSelector" $dns.podSelector) | nindent 6 }}
  {{- end }}
{{- end -}}

{{/*
Name of the existing Secret carrying CHALLENGE_KEY and the SMTP credentials
of the contact form. values.schema.json allows this to stay empty, because
no default Secret name could ever point at something that actually exists
on the cluster; this check turns that empty default into a clear
render-time failure instead of a site whose contact form answers 503.
*/}}
{{- define "ostendorf-tech.existingSecretName" -}}
{{- required "existingSecret.name is required: set it to the name of an existing Secret carrying CHALLENGE_KEY and the SMTP credentials, without which the contact form answers 503." .Values.existingSecret.name -}}
{{- end -}}

{{/*
Feed service fullname: the website's own fullname with "-feed" appended, so the two
workloads' names can never collide.
*/}}
{{- define "ostendorf-tech.feed.fullname" -}}
{{- printf "%s-feed" (include "ostendorf-tech.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Feed selector labels. Deliberately a different app.kubernetes.io/name than
ostendorf-tech.selectorLabels: reusing the website's own selector labels here would make
the website's Service and the feed's Service each select both workloads' pods instead of
just their own.
*/}}
{{- define "ostendorf-tech.feed.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ostendorf-tech.name" . }}-feed
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Feed common labels. component is "feed", distinct from the website's own "server".
*/}}
{{- define "ostendorf-tech.feed.labels" -}}
helm.sh/chart: {{ include "ostendorf-tech.chart" . }}
{{ include "ostendorf-tech.feed.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/component: feed
app.kubernetes.io/part-of: {{ include "ostendorf-tech.name" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Feed ServiceAccount name, same create/name override pattern as the website's own.
*/}}
{{- define "ostendorf-tech.feed.serviceAccountName" -}}
{{- if .Values.feed.serviceAccount.create -}}
{{- default (include "ostendorf-tech.feed.fullname" .) .Values.feed.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.feed.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{/*
OpenBao address the feed service authenticates against. Required: no default could ever
point at an OpenBao installation that actually exists.
*/}}
{{- define "ostendorf-tech.feed.openBaoAddr" -}}
{{- required "feed.config.openBaoAddr is required: set it to the OpenBao address the feed service authenticates against. There is no default, since none could ever point at an OpenBao installation that actually exists." .Values.feed.config.openBaoAddr -}}
{{- end -}}

{{/*
OpenBao auth method of the feed service, after the render time guards that
refuse a mixed configuration: the token method needs a token Secret
reference and no Kubernetes role, the kubernetes method takes no token
Secret reference. values.schema.json already limits the value to the two
methods. A token itself is never a value: it comes from an existing Secret.
*/}}
{{- define "ostendorf-tech.feed.openBaoAuthMethod" -}}
{{- $method := .Values.feed.config.openBaoAuthMethod | default "kubernetes" -}}
{{- $ref := .Values.feed.openBaoToken | default dict -}}
{{- if eq $method "token" -}}
{{- if not $ref.existingSecret -}}
{{- fail "feed.openBaoToken.existingSecret is required with feed.config.openBaoAuthMethod=token: set it to the name of an existing Secret holding the OpenBao token." -}}
{{- end -}}
{{- if not $ref.existingSecretKey -}}
{{- fail "feed.openBaoToken.existingSecretKey is required with feed.config.openBaoAuthMethod=token: set it to the key of the OpenBao token inside that Secret." -}}
{{- end -}}
{{- if .Values.feed.config.openBaoRole -}}
{{- fail "feed.config.openBaoRole must not be set with feed.config.openBaoAuthMethod=token: the role belongs to the kubernetes method." -}}
{{- end -}}
{{- else if $ref.existingSecret -}}
{{- fail "feed.openBaoToken.existingSecret must not be set with feed.config.openBaoAuthMethod=kubernetes: the token belongs to the token method." -}}
{{- end -}}
{{- $method -}}
{{- end -}}

{{/*
OpenBao Kubernetes auth role the feed service logs in as. Required for the same reason
as openBaoAddr: the role and its policy are created once, by hand, in OpenBao itself (see
issue #24's plan), and no default here could ever name one that actually exists.
*/}}
{{- define "ostendorf-tech.feed.openBaoRole" -}}
{{- required "feed.config.openBaoRole is required: set it to the OpenBao Kubernetes auth role the feed service logs in as. There is no default, since none could ever name a role that actually exists in your OpenBao installation." .Values.feed.config.openBaoRole -}}
{{- end -}}

{{/*
GOMEMLIMIT must stay below the container's memory limit, or the Go runtime never
collects early enough to keep the pod inside it. Called with a list: the value's
name, the value, the memory limit's name and the memory limit. values.schema.json
already checks the value's form (a whole number with an optional B, KiB, MiB, GiB or
TiB suffix). The limit is compared when it is a whole number with an optional Ki, Mi,
Gi, Ti, k, M, G or T suffix; another quantity form, a decimal for example, is not
compared. An empty value or an empty limit renders nothing to check.
*/}}
{{- define "ostendorf-tech.checkGoMemLimit" -}}
{{- $name := index . 0 -}}
{{- $value := index . 1 | default "" | toString -}}
{{- $limitName := index . 2 -}}
{{- $limit := index . 3 | default "" | toString -}}
{{- $goUnits := dict "" 1 "B" 1 "KiB" 1024 "MiB" 1048576 "GiB" 1073741824 "TiB" 1099511627776 -}}
{{- $kubeUnits := dict "" 1 "Ki" 1024 "Mi" 1048576 "Gi" 1073741824 "Ti" 1099511627776 "k" 1000 "M" 1000000 "G" 1000000000 "T" 1000000000000 -}}
{{- if and $value $limit (regexMatch "^[0-9]+(B|KiB|MiB|GiB|TiB)?$" $value) (regexMatch "^[0-9]+(Ki|Mi|Gi|Ti|k|M|G|T)?$" $limit) -}}
{{- $valueDigits := regexFind "^[0-9]+" $value -}}
{{- $valueBytes := mul ($valueDigits | int64) (get $goUnits (trimPrefix $valueDigits $value)) -}}
{{- $limitDigits := regexFind "^[0-9]+" $limit -}}
{{- $limitBytes := mul ($limitDigits | int64) (get $kubeUnits (trimPrefix $limitDigits $limit)) -}}
{{- if ge $valueBytes $limitBytes -}}
{{- fail (printf "%s must be below %s: GOMEMLIMIT is the heap size at which Go collects harder, so it has to leave room under the container's memory limit. About 90 percent of the limit is the default." $name $limitName) -}}
{{- end -}}
{{- end -}}
{{- end -}}
