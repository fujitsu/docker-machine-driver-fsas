{{/*
Expand the chart name.
*/}}
{{- define "fsas-cluster.name" -}}
{{- default .Chart.Name .Values.cluster.name | trunc 63 | trimSuffix "-" -}}
{{- end }}

{{/*
Create a full name.
*/}}
{{- define "fsas-cluster.fullname" -}}
{{- if .Values.cluster.name }}
{{- .Values.cluster.name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Pool FsasConfig name.
*/}}
{{- define "fsas-cluster.pool1ConfigName" -}}
{{ include "fsas-cluster.fullname" . }}-worker
{{- end }}