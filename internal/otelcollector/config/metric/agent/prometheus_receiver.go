package agent

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/kyma-project/telemetry-manager/internal/otelcollector/config"
)

type NodeAffiliatedResource string

const (
	NodeAffiliatedPod      NodeAffiliatedResource = "pod"
	NodeAffiliatedEndpoint NodeAffiliatedResource = "endpoint"
)

type AnnotatedResource string

const (
	AnnotatedPod                   AnnotatedResource = "pod"
	AnnotatedService               AnnotatedResource = "service"
	PodNodeSelectorFieldExpression string            = "spec.nodeName=${MY_NODE_NAME}"
)

const (
	scrapeInterval           = 30 * time.Second
	sampleLimit              = 50000
	appPodsJobName           = "app-pods"
	appServicesJobName       = "app-services"
	appServicesSecureJobName = "app-services-secure"
)

// makePrometheusConfigForPods creates a Prometheus configuration for scraping Pods that are annotated with prometheus.io annotations.
func makePrometheusConfigForPods() *PrometheusReceiver {
	var config PrometheusReceiver

	scrapeConfig := ScrapeConfig{
		ScrapeInterval:             scrapeInterval,
		SampleLimit:                sampleLimit,
		KubernetesDiscoveryConfigs: makeDiscoveryConfigWithNodeSelector(RolePod),
		JobName:                    appPodsJobName,
		RelabelConfigs:             makePrometheusPodsRelabelConfigs(),
	}

	config.Config.ScrapeConfigs = append(config.Config.ScrapeConfigs, scrapeConfig)

	return &config
}

// makePrometheusConfigForServices creates a Prometheus configuration for scraping Services that are annotated with prometheus.io annotations.
// If Istio is enabled, an additional scrape job config is generated (suffixed with -secure) to scrape annotated Services over HTTPS using Istio certificate.
// Istio certificate is expected to be mounted at the provided path using the proxy.istio.io/config annotation.
// See more: https://istio.io/latest/docs/ops/integrations/prometheus/#tls-settings
func makePrometheusConfigForServices(opts BuildOptions) *PrometheusReceiver {
	var config PrometheusReceiver

	baseScrapeConfig := ScrapeConfig{
		ScrapeInterval:             scrapeInterval,
		SampleLimit:                sampleLimit,
		KubernetesDiscoveryConfigs: makeDiscoveryConfigWithNodeSelector(RoleEndpoints),
	}

	httpScrapeConfig := baseScrapeConfig
	httpScrapeConfig.JobName = appServicesJobName
	httpScrapeConfig.RelabelConfigs = makePrometheusEndpointsRelabelConfigs(false)
	config.Config.ScrapeConfigs = append(config.Config.ScrapeConfigs, httpScrapeConfig)

	// If Istio is enabled, generate an additional scrape config for scraping annotated Services over HTTPS
	if opts.IstioEnabled {
		httpsScrapeConfig := baseScrapeConfig
		httpsScrapeConfig.JobName = appServicesSecureJobName
		httpsScrapeConfig.RelabelConfigs = makePrometheusEndpointsRelabelConfigs(true)
		httpsScrapeConfig.TLSConfig = makeTLSConfig(opts.IstioCertPath)
		config.Config.ScrapeConfigs = append(config.Config.ScrapeConfigs, httpsScrapeConfig)
	}

	return &config
}

// makePrometheusPodsRelabelConfigs generates a set of relabel configs for the Pod role type.
// They restrict Pods that are selected for scraping and set internal labels (__address__, __scheme__, etc.).
// See more: https://prometheus.io/docs/prometheus/latest/configuration/configuration/#pod.
//
// Only Pods without Istio sidecars are selected.
func makePrometheusPodsRelabelConfigs() []RelabelConfig {
	return []RelabelConfig{
		keepIfRunningOnSameNode(NodeAffiliatedPod),
		keepIfScrapingEnabled(AnnotatedPod),
		dropIfPodNotRunning(),
		dropIfInitContainer(),
		inferSchemeFromIstioInjectedLabel(),
		dropIfSchemeHTTPS(),
		inferMetricsPathFromAnnotation(AnnotatedPod),
		inferAddressFromAnnotation(AnnotatedPod),
		inferURLParamFromAnnotation(AnnotatedPod),
	}
}

// makePrometheusEndpointsRelabelConfigs generates a set of relabel configs for the Endpoint role type.
// They restrict Service Endpoints that are selected for scraping and set internal labels (__address__, __scheme__, etc.).
// See more: https://prometheus.io/docs/prometheus/latest/configuration/configuration/#endpoint.
//
// If requireHTTPS is true, only Endpoints backed by Pods with Istio sidecars or those explicitly marked with prometheus.io/scheme=http annotations are selected.
// If requireHTTPS is false, only Endpoints backed by Pods wuthout Istio sidecars or those marked with prometheus.io/scheme=https annotation are selcted.
func makePrometheusEndpointsRelabelConfigs(requireHTTPS bool) []RelabelConfig {
	relabelConfigs := []RelabelConfig{
		keepIfRunningOnSameNode(NodeAffiliatedEndpoint),
		keepIfScrapingEnabled(AnnotatedService),
		dropIfPodNotRunning(),
		dropIfInitContainer(),
		dropIfIstioProxy(),
		inferSchemeFromIstioInjectedLabel(),
		inferSchemeFromAnnotation(AnnotatedService),
		inferURLParamFromAnnotation(AnnotatedService),
	}

	if requireHTTPS {
		relabelConfigs = append(relabelConfigs, dropIfSchemeHTTP())
	} else {
		relabelConfigs = append(relabelConfigs, dropIfSchemeHTTPS())
	}

	return append(relabelConfigs,
		inferMetricsPathFromAnnotation(AnnotatedService),
		inferAddressFromAnnotation(AnnotatedService),
		inferServiceFromMetaLabel())
}

func makeTLSConfig(istioCertPath string) *TLSConfig {
	istioCAFile := filepath.Join(istioCertPath, "root-cert.pem")
	istioCertFile := filepath.Join(istioCertPath, "cert-chain.pem")
	istioKeyFile := filepath.Join(istioCertPath, "key.pem")

	return &TLSConfig{
		CAFile:             istioCAFile,
		CertFile:           istioCertFile,
		KeyFile:            istioKeyFile,
		InsecureSkipVerify: true,
	}
}

func makePrometheusIstioConfig(envoyMetricsEnabled bool) *PrometheusReceiver {
	metricNames := "istio_.*"
	if envoyMetricsEnabled {
		metricNames = strings.Join([]string{"envoy_.*", metricNames}, "|")
	}

	return &PrometheusReceiver{
		Config: PrometheusConfig{
			ScrapeConfigs: []ScrapeConfig{
				{
					JobName:                    "istio-proxy",
					SampleLimit:                sampleLimit,
					MetricsPath:                "/stats/prometheus",
					ScrapeInterval:             scrapeInterval,
					KubernetesDiscoveryConfigs: makeDiscoveryConfigWithNodeSelector(RolePod),
					RelabelConfigs: []RelabelConfig{
						keepIfRunningOnSameNode(NodeAffiliatedPod),
						keepIfIstioProxy(),
						keepIfContainerWithEnvoyPort(),
						dropIfPodNotRunning(),
					},
					MetricRelabelConfigs: []RelabelConfig{
						{
							SourceLabels: []string{"__name__"},
							Regex:        metricNames,
							Action:       Keep,
						},
					},
				},
			},
		},
	}
}

func keepIfRunningOnSameNode(nodeAffiliated NodeAffiliatedResource) RelabelConfig {
	return RelabelConfig{
		SourceLabels: []string{fmt.Sprintf("__meta_kubernetes_%s_node_name", nodeAffiliated)},
		Regex:        fmt.Sprintf("${%s}", config.EnvVarCurrentNodeName),
		Action:       Keep,
	}
}

func keepIfScrapingEnabled(annotated AnnotatedResource) RelabelConfig {
	return RelabelConfig{
		SourceLabels: []string{fmt.Sprintf("__meta_kubernetes_%s_annotation_prometheus_io_scrape", annotated)},
		Regex:        "true",
		Action:       Keep,
	}
}

func keepIfIstioProxy() RelabelConfig {
	return RelabelConfig{
		SourceLabels: []string{"__meta_kubernetes_pod_container_name"},
		Action:       Keep,
		Regex:        "istio-proxy",
	}
}

func keepIfContainerWithEnvoyPort() RelabelConfig {
	return RelabelConfig{
		SourceLabels: []string{"__meta_kubernetes_pod_container_port_name"},
		Action:       Keep,
		Regex:        "http-envoy-prom",
	}
}

// InferSchemeFromIstioInjectedLabel configures the default scraping scheme to HTTPS
// based on the presence of the security.istio.io/tlsMode label in a Pod. This label
// is automatically added by Istio's MutatingWebhook when a sidecar is injected.
//
// When a sidecar is detected (i.e., the label is present), this function sets the scraping scheme to HTTPS.
//
// Note: The HTTPS scheme can be manually overridden by setting the "prometheus.io/scheme"
// annotation on the Pod or the Service.
func inferSchemeFromIstioInjectedLabel() RelabelConfig {
	return RelabelConfig{
		SourceLabels: []string{"__meta_kubernetes_pod_label_security_istio_io_tlsMode"},
		Action:       Replace,
		TargetLabel:  "__scheme__",
		Regex:        "(istio)",
		Replacement:  "https",
	}
}

func inferSchemeFromAnnotation(annotated AnnotatedResource) RelabelConfig {
	return RelabelConfig{
		SourceLabels: []string{fmt.Sprintf("__meta_kubernetes_%s_annotation_prometheus_io_scheme", annotated)},
		Action:       Replace,
		Regex:        "(https?)",
		TargetLabel:  "__scheme__",
	}
}

func inferMetricsPathFromAnnotation(annotated AnnotatedResource) RelabelConfig {
	return RelabelConfig{
		SourceLabels: []string{fmt.Sprintf("__meta_kubernetes_%s_annotation_prometheus_io_path", annotated)},
		Action:       Replace,
		Regex:        "(.+)",
		TargetLabel:  "__metrics_path__",
	}
}

func inferAddressFromAnnotation(annotated AnnotatedResource) RelabelConfig {
	return RelabelConfig{
		SourceLabels: []string{"__address__", fmt.Sprintf("__meta_kubernetes_%s_annotation_prometheus_io_port", annotated)},
		Action:       Replace,
		Regex:        "([^:]+)(?::\\d+)?;(\\d+)",
		Replacement:  "$$1:$$2",
		TargetLabel:  "__address__",
	}
}

func inferServiceFromMetaLabel() RelabelConfig {
	return RelabelConfig{
		SourceLabels: []string{"__meta_kubernetes_service_name"},
		Action:       Replace,
		TargetLabel:  "service",
	}
}

func dropIfPodNotRunning() RelabelConfig {
	return RelabelConfig{
		SourceLabels: []string{"__meta_kubernetes_pod_phase"},
		Action:       Drop,
		Regex:        "Pending|Succeeded|Failed",
	}
}

func dropIfInitContainer() RelabelConfig {
	return RelabelConfig{
		SourceLabels: []string{"__meta_kubernetes_pod_container_init"},
		Action:       Drop,
		Regex:        "(true)",
	}
}

func dropIfIstioProxy() RelabelConfig {
	return RelabelConfig{
		SourceLabels: []string{"__meta_kubernetes_pod_container_name"},
		Action:       Drop,
		Regex:        "(istio-proxy)",
	}
}

func dropIfSchemeHTTP() RelabelConfig {
	return RelabelConfig{
		SourceLabels: []string{"__scheme__"},
		Action:       Drop,
		Regex:        "(http)",
	}
}

func dropIfSchemeHTTPS() RelabelConfig {
	return RelabelConfig{
		SourceLabels: []string{"__scheme__"},
		Action:       Drop,
		Regex:        "(https)",
	}
}

// inferURLParamFromAnnotation extracts and configures the URL parameter
// for scraping based on annotations of the form prometheus.io/param_{name}: {value}.
func inferURLParamFromAnnotation(annotated AnnotatedResource) RelabelConfig {
	return RelabelConfig{
		Regex:       fmt.Sprintf("__meta_kubernetes_%s_annotation_prometheus_io_param_(.+)", annotated),
		Action:      LabelMap,
		Replacement: "__param_$1",
	}
}

func makeDiscoveryConfigWithNodeSelector(role Role) []KubernetesDiscoveryConfig {
	return []KubernetesDiscoveryConfig{
		{
			Role: role,
			Selectors: []K8SDiscoverySelector{
				{
					Role:  RolePod,
					Field: PodNodeSelectorFieldExpression,
				},
			},
		},
	}
}
