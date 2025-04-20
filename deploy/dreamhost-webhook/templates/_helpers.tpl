{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "dreamhost-webhook.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "dreamhost-webhook.fullname" -}}
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
{{- define "dreamhost-webhook.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "dreamhost-webhook.selfSignedIssuer" -}}
{{ printf "%s-selfsign" (include "dreamhost-webhook.fullname" .) }}
{{- end -}}

{{- define "dreamhost-webhook.rootCAIssuer" -}}
{{ printf "%s-ca" (include "dreamhost-webhook.fullname" .) }}
{{- end -}}

{{- define "dreamhost-webhook.rootCACertificate" -}}
{{ printf "%s-ca" (include "dreamhost-webhook.fullname" .) }}
{{- end -}}

{{- define "dreamhost-webhook.servingCertificate" -}}
{{ printf "%s-webhook-tls" (include "dreamhost-webhook.fullname" .) }}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "dreamhost-webhook.labels" -}}
helm.sh/chart: {{ include "dreamhost-webhook.chart" . }}
{{ include "dreamhost-webhook.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "dreamhost-webhook.selectorLabels" -}}
app.kubernetes.io/name: {{ include "dreamhost-webhook.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
