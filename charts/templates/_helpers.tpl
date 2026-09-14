{{/*
Expand the chart name.
*/}}
{{- define "fsas-cluster-template.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a fullname.
*/}}
{{- define "fsas-cluster-template.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- .Values.cluster.name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "fsas-cluster-template.labels" -}}
app.kubernetes.io/name: {{ include "fsas-cluster-template.name" . }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Control-plane FsasConfig name.
*/}}
{{- define "fsas-cluster-template.cpConfigName" -}}
{{ printf "%s-cp" .Values.cluster.name }}
{{- end }}

{{/*
Worker FsasConfig name.
*/}}
{{- define "fsas-cluster-template.workerConfigName" -}}
{{ printf "%s-wk" .Values.cluster.name }}
{{- end }}