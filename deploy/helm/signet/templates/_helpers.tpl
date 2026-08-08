{{/*
Expand the name of the chart.
*/}}
{{- define "signet.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "signet.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Chart label value.
*/}}
{{- define "signet.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels applied to every resource.
*/}}
{{- define "signet.labels" -}}
helm.sh/chart: {{ include "signet.chart" . }}
{{ include "signet.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels — used in Deployment.spec.selector and Service.spec.selector.
These must remain stable; do not add mutable labels here.
*/}}
{{- define "signet.selectorLabels" -}}
app.kubernetes.io/name: {{ include "signet.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
ServiceAccount name for signetd (needs TokenReview permission).
*/}}
{{- define "signet.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "signet.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
ServiceAccount name for admin operators.
*/}}
{{- define "signet.adminServiceAccountName" -}}
{{- default "signet-admin" .Values.adminServiceAccount.name }}
{{- end }}

{{/*
Name of the Secret containing SIGNET_DB_CONN_STRING and SIGNET_AUDIT_CHAIN_KEY.
*/}}
{{- define "signet.secretName" -}}
{{- if .Values.signet.existingSecret }}
{{- .Values.signet.existingSecret }}
{{- else }}
{{- include "signet.fullname" . }}
{{- end }}
{{- end }}

{{/*
CockroachDB fully-qualified name within this release.
*/}}
{{- define "signet.cockroachdbName" -}}
{{- printf "%s-cockroachdb" (include "signet.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
CockroachDB headless service DNS name.
*/}}
{{- define "signet.cockroachdbHost" -}}
{{- printf "%s.%s.svc.cluster.local" (include "signet.cockroachdbName" .) .Release.Namespace }}
{{- end }}

{{/*
Effective admin gRPC bind address. signet.adminAddr defaults to
127.0.0.1:8444 (loopback-only). When admin.clusterAccess is enabled, the
loopback host is rewritten to 0.0.0.0 so the admin listener is actually
reachable via the Service/NetworkPolicy this chart creates for it -- a
Service alone cannot make a loopback-bound listener reachable from outside
the pod's network namespace. A no-op if the operator has already pointed
signet.adminAddr somewhere other than 127.0.0.1.
*/}}
{{- define "signet.adminAddr" -}}
{{- if .Values.admin.clusterAccess -}}
{{- .Values.signet.adminAddr | replace "127.0.0.1:" "0.0.0.0:" -}}
{{- else -}}
{{- .Values.signet.adminAddr -}}
{{- end -}}
{{- end }}

{{/*
Full image reference for signetd.
global.image.registry (if set) replaces image.registry, enabling air-gapped
installs without requiring changes to image.repository.
*/}}
{{- define "signet.image" -}}
{{- $registry := .Values.global.image.registry | default .Values.image.registry -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion -}}
{{- if $registry -}}
{{- printf "%s/%s:%s" $registry .Values.image.repository $tag -}}
{{- else -}}
{{- printf "%s:%s" .Values.image.repository $tag -}}
{{- end -}}
{{- end }}
