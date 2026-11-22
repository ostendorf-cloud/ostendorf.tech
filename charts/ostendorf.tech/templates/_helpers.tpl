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
Common labels.
*/}}
{{- define "ostendorf-tech.labels" -}}
helm.sh/chart: {{ include "ostendorf-tech.chart" . }}
{{ include "ostendorf-tech.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/component: server
app.kubernetes.io/part-of: {{ include "ostendorf-tech.name" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels. Deliberately excludes app.kubernetes.io/version: a selector
must stay stable across an image upgrade, matchLabels is immutable once a
Deployment exists.
*/}}
{{- define "ostendorf-tech.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ostendorf-tech.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Create the name of the service account to use.
*/}}
{{- define "ostendorf-tech.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "ostendorf-tech.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
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
OpenBao Kubernetes auth role the feed service logs in as. Required for the same reason
as openBaoAddr: the role and its policy are created once, by hand, in OpenBao itself (see
issue #24's plan), and no default here could ever name one that actually exists.
*/}}
{{- define "ostendorf-tech.feed.openBaoRole" -}}
{{- required "feed.config.openBaoRole is required: set it to the OpenBao Kubernetes auth role the feed service logs in as. There is no default, since none could ever name a role that actually exists in your OpenBao installation." .Values.feed.config.openBaoRole -}}
{{- end -}}
