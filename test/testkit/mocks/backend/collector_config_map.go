package backend

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testutils "github.com/kyma-project/telemetry-manager/internal/utils/test"
)

type collectorConfigMapBuilder struct {
	name             string
	namespace        string
	exportedFilePath string
	signalType       SignalType
	certs            *testutils.ServerCerts
}

func newCollectorConfigMap(name, namespace, path string, signalType SignalType, certs *testutils.ServerCerts) *collectorConfigMapBuilder {
	return &collectorConfigMapBuilder{
		name:             name,
		namespace:        namespace,
		exportedFilePath: path,
		signalType:       signalType,
		certs:            certs,
	}
}

const otelConfigTemplate = `receivers:
  otlp:
    protocols:
      grpc:
        endpoint: ${MY_POD_IP}:4317
      http:
        endpoint: ${MY_POD_IP}:4318
exporters:
  file:
    path: {{ FILEPATH }}
service:
  telemetry:
    logs:
      level: "info"
  pipelines:
    {{ SIGNAL_TYPE }}:
      receivers:
        - otlp
      exporters:
        - file`

const tlsConfigTemplate = `receivers:
  otlp:
    protocols:
      grpc:
        tls:
          cert_pem: "{{ CERT_PEM }}"
          key_pem: "{{ KEY_PEM }}"
          client_ca_file: {{ CA_FILE_PATH }}
        endpoint: ${MY_POD_IP}:4317
      http:
        endpoint: ${MY_POD_IP}:4318
exporters:
  file:
    path: {{ FILEPATH }}
service:
  telemetry:
    logs:
      level: "info"
  pipelines:
    {{ SIGNAL_TYPE }}:
      receivers:
        - otlp
      exporters:
        - file`

const logConfigTemplate = `receivers:
  fluentforward:
    endpoint: localhost:8006
  otlp:
    protocols:
      grpc:
        endpoint: ${MY_POD_IP}:4317
      http:
        endpoint: ${MY_POD_IP}:4318
exporters:
  file:
    path: {{ FILEPATH }}
service:
  telemetry:
    logs:
      level: "info"
  pipelines:
    {{ SIGNAL_TYPE }}:
      receivers:
        - otlp
        - fluentforward
      exporters:
        - file`

func (cm *collectorConfigMapBuilder) Name() string {
	return cm.name
}

func (cm *collectorConfigMapBuilder) K8sObject() *corev1.ConfigMap {
	var configTemplate string

	switch {
	case cm.signalType == SignalTypeLogsFluentBit:
		configTemplate = logConfigTemplate
	case cm.certs != nil:
		configTemplate = tlsConfigTemplate
	default:
		configTemplate = otelConfigTemplate
	}

	config := strings.Replace(configTemplate, "{{ FILEPATH }}", cm.exportedFilePath, 1)
	if cm.signalType == SignalTypeLogsOTel {
		config = strings.Replace(config, "{{ SIGNAL_TYPE }}", "logs", 1)
	} else {
		config = strings.Replace(config, "{{ SIGNAL_TYPE }}", string(cm.signalType), 1)
	}

	data := make(map[string]string)

	if cm.certs != nil && cm.signalType != SignalTypeLogsFluentBit {
		certPem := strings.ReplaceAll(cm.certs.ServerCertPem.String(), "\n", "\\n")
		keyPem := strings.ReplaceAll(cm.certs.ServerKeyPem.String(), "\n", "\\n")
		config = strings.Replace(config, "{{ CERT_PEM }}", certPem, 1)
		config = strings.Replace(config, "{{ KEY_PEM }}", keyPem, 1)
		config = strings.Replace(config, "{{ CA_FILE_PATH }}", "/etc/collector/ca.crt", 1)

		data["ca.crt"] = cm.certs.CaCertPem.String()
	}

	data["config.yaml"] = config

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cm.name,
			Namespace: cm.namespace,
		},
		Data: data,
	}
}
