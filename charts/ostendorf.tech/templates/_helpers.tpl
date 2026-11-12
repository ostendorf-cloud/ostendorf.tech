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
Name of the existing Secret carrying CSRF_KEY and the other secret-backed
values. values.schema.json allows this to stay empty, because no default
Secret name could ever point at something that actually exists on the
cluster; this check turns that empty default into a clear render-time
failure instead of a Pod that crash-loops on a missing CSRF_KEY.
*/}}
{{- define "ostendorf-tech.existingSecretName" -}}
{{- required "existingSecret.name is required: set it to the name of an existing Secret carrying CSRF_KEY, which the application needs to start and has no default." .Values.existingSecret.name -}}
{{- end -}}
