{{/* 资源名：稳定版 = appName，灰度版 = appName-<grayTag> */}}
{{- define "acme.name" -}}
{{- required "appName 必填" .Values.appName -}}
{{- end -}}

{{- define "acme.fullname" -}}
{{- if .Values.grayTag -}}
{{- printf "%s-%s" (include "acme.name" .) .Values.grayTag | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- include "acme.name" . -}}
{{- end -}}
{{- end -}}

{{- define "acme.image" -}}
{{- if .Values.image.digest -}}
{{ .Values.image.repository }}@{{ .Values.image.digest }}
{{- else -}}
{{ .Values.image.repository }}:{{ .Values.image.tag }}
{{- end -}}
{{- end -}}

{{- define "acme.labels" -}}
app.kubernetes.io/name: {{ include "acme.name" . }}
app.kubernetes.io/instance: {{ include "acme.fullname" . }}
app.kubernetes.io/version: {{ .Values.image.tag | quote }}
app.kubernetes.io/part-of: acme-cloud
app.kubernetes.io/managed-by: {{ .Release.Service }}
acme.com/env: {{ .Values.env }}
acme.com/gray-tag: {{ .Values.grayTag | default "stable" | quote }}
{{- end -}}

{{/* Service 只按 name 选择：稳定版与灰度版 Pod 都在同一个 Service 后面，流量分配由 Nacos 元数据 + 灰度 LoadBalancer 决定 */}}
{{- define "acme.serviceSelector" -}}
app.kubernetes.io/name: {{ include "acme.name" . }}
{{- end -}}

{{- define "acme.selectorLabels" -}}
app.kubernetes.io/name: {{ include "acme.name" . }}
app.kubernetes.io/instance: {{ include "acme.fullname" . }}
{{- end -}}
