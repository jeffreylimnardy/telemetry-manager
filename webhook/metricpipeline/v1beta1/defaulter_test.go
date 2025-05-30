package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"

	telemetryv1beta1 "github.com/kyma-project/telemetry-manager/apis/telemetry/v1beta1"
)

func TestDefault(t *testing.T) {
	sut := defaulter{
		ExcludeNamespaces: []string{"kyma-system", "kube-system", "istio-system", "compass-system"},
		RuntimeInputResources: runtimeInputResourceDefaults{
			Pod:         true,
			Container:   true,
			Node:        true,
			Volume:      true,
			DaemonSet:   true,
			Deployment:  true,
			StatefulSet: true,
			Job:         true,
		},
		DefaultOTLPOutputProtocol: telemetryv1beta1.OTLPProtocolGRPC,
	}

	tests := []struct {
		name     string
		input    *telemetryv1beta1.MetricPipeline
		expected *telemetryv1beta1.MetricPipeline
	}{
		{
			name: "should set default OTLP protocol if not set",
			input: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Output: telemetryv1beta1.MetricPipelineOutput{
						OTLP: &telemetryv1beta1.OTLPOutput{},
					},
				},
			},
			expected: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Output: telemetryv1beta1.MetricPipelineOutput{
						OTLP: &telemetryv1beta1.OTLPOutput{
							Protocol: telemetryv1beta1.OTLPProtocolGRPC,
						},
					},
				},
			},
		},
		{
			name: "should not override existing OTLP protocol",
			input: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Output: telemetryv1beta1.MetricPipelineOutput{
						OTLP: &telemetryv1beta1.OTLPOutput{
							Protocol: telemetryv1beta1.OTLPProtocolHTTP,
						},
					},
				},
			},
			expected: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Output: telemetryv1beta1.MetricPipelineOutput{
						OTLP: &telemetryv1beta1.OTLPOutput{
							Protocol: telemetryv1beta1.OTLPProtocolHTTP,
						},
					},
				},
			},
		},
		{
			name: "should set default namespaces for Prometheus input",
			input: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Input: telemetryv1beta1.MetricPipelineInput{
						Prometheus: &telemetryv1beta1.MetricPipelinePrometheusInput{
							Enabled: ptr.To(true),
						},
					},
				},
			},
			expected: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Input: telemetryv1beta1.MetricPipelineInput{
						Prometheus: &telemetryv1beta1.MetricPipelinePrometheusInput{
							Enabled: ptr.To(true),
							Namespaces: &telemetryv1beta1.NamespaceSelector{
								Exclude: []string{"kyma-system", "kube-system", "istio-system", "compass-system"},
							},
							DiagnosticMetrics: &telemetryv1beta1.MetricPipelineIstioInputDiagnosticMetrics{
								Enabled: ptr.To(false),
							},
						},
					},
				},
			},
		},
		{
			name: "should set default namespaces for Istio input",
			input: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Input: telemetryv1beta1.MetricPipelineInput{
						Istio: &telemetryv1beta1.MetricPipelineIstioInput{
							Enabled: ptr.To(true),
						},
					},
				},
			},
			expected: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Input: telemetryv1beta1.MetricPipelineInput{
						Istio: &telemetryv1beta1.MetricPipelineIstioInput{
							Enabled: ptr.To(true),
							Namespaces: &telemetryv1beta1.NamespaceSelector{
								Exclude: []string{"kyma-system", "kube-system", "istio-system", "compass-system"},
							},
							EnvoyMetrics: &telemetryv1beta1.EnvoyMetrics{
								Enabled: ptr.To(false),
							},
							DiagnosticMetrics: &telemetryv1beta1.MetricPipelineIstioInputDiagnosticMetrics{
								Enabled: ptr.To(false),
							},
						},
					},
				},
			},
		},

		{
			name: "should set default for Runtime input",
			input: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Input: telemetryv1beta1.MetricPipelineInput{
						Runtime: &telemetryv1beta1.MetricPipelineRuntimeInput{
							Enabled: ptr.To(true),
						},
					},
				},
			},
			expected: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Input: telemetryv1beta1.MetricPipelineInput{
						Runtime: &telemetryv1beta1.MetricPipelineRuntimeInput{
							Enabled: ptr.To(true),
							Namespaces: &telemetryv1beta1.NamespaceSelector{
								Exclude: []string{"kyma-system", "kube-system", "istio-system", "compass-system"},
							},
							Resources: &telemetryv1beta1.MetricPipelineRuntimeInputResources{
								Pod: &telemetryv1beta1.MetricPipelineRuntimeInputResource{
									Enabled: ptr.To(true),
								},

								Container: &telemetryv1beta1.MetricPipelineRuntimeInputResource{
									Enabled: ptr.To(true),
								},

								Node: &telemetryv1beta1.MetricPipelineRuntimeInputResource{
									Enabled: ptr.To(true),
								},

								Volume: &telemetryv1beta1.MetricPipelineRuntimeInputResource{
									Enabled: ptr.To(true),
								},

								Deployment: &telemetryv1beta1.MetricPipelineRuntimeInputResource{
									Enabled: ptr.To(true),
								},

								Job: &telemetryv1beta1.MetricPipelineRuntimeInputResource{
									Enabled: ptr.To(true),
								},

								StatefulSet: &telemetryv1beta1.MetricPipelineRuntimeInputResource{
									Enabled: ptr.To(true),
								},

								DaemonSet: &telemetryv1beta1.MetricPipelineRuntimeInputResource{
									Enabled: ptr.To(true),
								},
							},
						},
					},
				},
			},
		},

		{
			name: "should set default for Runtime input except for Pod",
			input: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Input: telemetryv1beta1.MetricPipelineInput{
						Runtime: &telemetryv1beta1.MetricPipelineRuntimeInput{
							Enabled: ptr.To(true),
							Resources: &telemetryv1beta1.MetricPipelineRuntimeInputResources{
								Pod: &telemetryv1beta1.MetricPipelineRuntimeInputResource{
									Enabled: ptr.To(false),
								},
							},
						},
					},
				},
			},
			expected: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Input: telemetryv1beta1.MetricPipelineInput{
						Runtime: &telemetryv1beta1.MetricPipelineRuntimeInput{
							Enabled: ptr.To(true),
							Namespaces: &telemetryv1beta1.NamespaceSelector{
								Exclude: []string{"kyma-system", "kube-system", "istio-system", "compass-system"},
							},
							Resources: &telemetryv1beta1.MetricPipelineRuntimeInputResources{
								Pod: &telemetryv1beta1.MetricPipelineRuntimeInputResource{
									Enabled: ptr.To(false),
								},

								Container: &telemetryv1beta1.MetricPipelineRuntimeInputResource{
									Enabled: ptr.To(true),
								},

								Node: &telemetryv1beta1.MetricPipelineRuntimeInputResource{
									Enabled: ptr.To(true),
								},

								Volume: &telemetryv1beta1.MetricPipelineRuntimeInputResource{
									Enabled: ptr.To(true),
								},

								Deployment: &telemetryv1beta1.MetricPipelineRuntimeInputResource{
									Enabled: ptr.To(true),
								},

								Job: &telemetryv1beta1.MetricPipelineRuntimeInputResource{
									Enabled: ptr.To(true),
								},

								StatefulSet: &telemetryv1beta1.MetricPipelineRuntimeInputResource{
									Enabled: ptr.To(true),
								},

								DaemonSet: &telemetryv1beta1.MetricPipelineRuntimeInputResource{
									Enabled: ptr.To(true),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "should not set default for Prometheus input",
			input: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Input: telemetryv1beta1.MetricPipelineInput{
						Prometheus: &telemetryv1beta1.MetricPipelinePrometheusInput{
							Enabled: ptr.To(false),
						},
					},
				},
			},
			expected: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Input: telemetryv1beta1.MetricPipelineInput{
						Prometheus: &telemetryv1beta1.MetricPipelinePrometheusInput{
							Enabled: ptr.To(false),
						},
					},
				},
			},
		},
		{
			name: "should not set defaults for Istio input",
			input: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Input: telemetryv1beta1.MetricPipelineInput{
						Istio: &telemetryv1beta1.MetricPipelineIstioInput{
							Enabled: ptr.To(false),
						},
					},
				},
			},
			expected: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Input: telemetryv1beta1.MetricPipelineInput{
						Istio: &telemetryv1beta1.MetricPipelineIstioInput{
							Enabled: ptr.To(false),
						},
					},
				},
			},
		},
		{
			name: "should not set defaults for Runtime input",
			input: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Input: telemetryv1beta1.MetricPipelineInput{
						Runtime: &telemetryv1beta1.MetricPipelineRuntimeInput{
							Enabled: ptr.To(false),
						},
					},
				},
			},
			expected: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Input: telemetryv1beta1.MetricPipelineInput{
						Runtime: &telemetryv1beta1.MetricPipelineRuntimeInput{
							Enabled: ptr.To(false),
						},
					},
				},
			},
		},
		{
			name: "should not set default Istio Envoy metrics if set",
			input: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Input: telemetryv1beta1.MetricPipelineInput{
						Istio: &telemetryv1beta1.MetricPipelineIstioInput{
							Enabled: ptr.To(true),
							EnvoyMetrics: &telemetryv1beta1.EnvoyMetrics{
								Enabled: ptr.To(true),
							},
						},
					},
				},
			},
			expected: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Input: telemetryv1beta1.MetricPipelineInput{
						Istio: &telemetryv1beta1.MetricPipelineIstioInput{
							Enabled: ptr.To(true),
							Namespaces: &telemetryv1beta1.NamespaceSelector{
								Exclude: []string{"kyma-system", "kube-system", "istio-system", "compass-system"},
							},
							EnvoyMetrics: &telemetryv1beta1.EnvoyMetrics{
								Enabled: ptr.To(true),
							},
							DiagnosticMetrics: &telemetryv1beta1.MetricPipelineIstioInputDiagnosticMetrics{
								Enabled: ptr.To(false),
							},
						},
					},
				},
			},
		},
		{
			name: "should not set default Istio diagnostic metrics if set",
			input: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Input: telemetryv1beta1.MetricPipelineInput{
						Istio: &telemetryv1beta1.MetricPipelineIstioInput{
							Enabled:           ptr.To(true),
							DiagnosticMetrics: &telemetryv1beta1.MetricPipelineIstioInputDiagnosticMetrics{Enabled: ptr.To(true)},
						},
					},
				},
			},
			expected: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Input: telemetryv1beta1.MetricPipelineInput{
						Istio: &telemetryv1beta1.MetricPipelineIstioInput{
							Enabled: ptr.To(true),
							Namespaces: &telemetryv1beta1.NamespaceSelector{
								Exclude: []string{"kyma-system", "kube-system", "istio-system", "compass-system"},
							},
							EnvoyMetrics: &telemetryv1beta1.EnvoyMetrics{
								Enabled: ptr.To(false),
							},
							DiagnosticMetrics: &telemetryv1beta1.MetricPipelineIstioInputDiagnosticMetrics{
								Enabled: ptr.To(true),
							},
						},
					},
				},
			},
		},
		{
			name: "should not set default Prometheus diagnostic metrics if set",
			input: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Input: telemetryv1beta1.MetricPipelineInput{
						Prometheus: &telemetryv1beta1.MetricPipelinePrometheusInput{
							Enabled:           ptr.To(true),
							DiagnosticMetrics: &telemetryv1beta1.MetricPipelineIstioInputDiagnosticMetrics{Enabled: ptr.To(true)},
						},
					},
				},
			},
			expected: &telemetryv1beta1.MetricPipeline{
				Spec: telemetryv1beta1.MetricPipelineSpec{
					Input: telemetryv1beta1.MetricPipelineInput{
						Prometheus: &telemetryv1beta1.MetricPipelinePrometheusInput{
							Enabled: ptr.To(true),
							Namespaces: &telemetryv1beta1.NamespaceSelector{
								Exclude: []string{"kyma-system", "kube-system", "istio-system", "compass-system"},
							},
							DiagnosticMetrics: &telemetryv1beta1.MetricPipelineIstioInputDiagnosticMetrics{Enabled: ptr.To(true)},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sut.Default(t.Context(), tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}
