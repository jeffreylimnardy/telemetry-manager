package prober

import (
	"testing"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kyma-project/telemetry-manager/internal/selfmonitor/prober/mocks"
)

func TestFluentBitLogPipelineProber(t *testing.T) {
	testCases := []struct {
		name         string
		alerts       promv1.AlertsResult
		alertsErr    error
		pipelineName string
		expected     FluentBitProbeResult
		expectErr    bool
	}{
		{
			name:         "alert getter fails",
			pipelineName: "cls",
			alertsErr:    assert.AnError,
			expectErr:    true,
		},
		{
			name:         "no alerts firing",
			pipelineName: "cls",
			alerts: promv1.AlertsResult{
				Alerts: []promv1.Alert{},
			},
			expected: FluentBitProbeResult{
				PipelineProbeResult: PipelineProbeResult{
					Healthy: true,
				},
			},
		},
		{
			name:         "unknown alert firing",
			pipelineName: "cls",
			alerts: promv1.AlertsResult{
				Alerts: []promv1.Alert{
					{
						Labels: model.LabelSet{
							"alertname":     "UnknownAlert",
							"pipeline_name": "cls",
						},
						State: promv1.AlertStateFiring,
					},
				},
			},
			expected: FluentBitProbeResult{
				PipelineProbeResult: PipelineProbeResult{
					Healthy: true,
				},
			},
		},
		{
			name:         "pending alert should be ignored",
			pipelineName: "cls",
			alerts: promv1.AlertsResult{
				Alerts: []promv1.Alert{
					{
						Labels: model.LabelSet{
							"alertname": "LogFluentBitAllDataDropped",
						},
						State: promv1.AlertStatePending,
					},
				},
			},
			expected: FluentBitProbeResult{
				PipelineProbeResult: PipelineProbeResult{
					Healthy: true,
				},
			},
		},
		{
			name:         "alert missing pipeline_name label should be mapped to any pipeline",
			pipelineName: "cls",
			alerts: promv1.AlertsResult{
				Alerts: []promv1.Alert{
					{
						Labels: model.LabelSet{
							"alertname": "LogFluentBitAllDataDropped",
						},
						State: promv1.AlertStateFiring,
					},
				},
			},
			expected: FluentBitProbeResult{
				PipelineProbeResult: PipelineProbeResult{
					AllDataDropped: true,
					Healthy:        false,
				},
			},
		},
		{
			name:         "pipeline_name label does not match pipeline name",
			pipelineName: "cls",
			alerts: promv1.AlertsResult{
				Alerts: []promv1.Alert{
					{
						Labels: model.LabelSet{
							"alertname":     "LogFluentBitBufferFull",
							"pipeline_name": "dynatrace",
						},
						State: promv1.AlertStateFiring,
					},
				},
			},
			expected: FluentBitProbeResult{
				PipelineProbeResult: PipelineProbeResult{
					Healthy: true,
				},
			},
		},
		{
			name:         "flow type mismatch",
			pipelineName: "cls",
			alerts: promv1.AlertsResult{
				Alerts: []promv1.Alert{
					{
						Labels: model.LabelSet{
							"alertname":     "MetricGatewayAllDataDropped",
							"pipeline_name": "cls",
						},
						State: promv1.AlertStateFiring,
					},
					{
						Labels: model.LabelSet{
							"alertname":     "TraceGatewayAllDataDropped",
							"pipeline_name": "cls",
						},
						State: promv1.AlertStateFiring,
					},
				},
			},
			expected: FluentBitProbeResult{
				PipelineProbeResult: PipelineProbeResult{
					Healthy: true,
				},
			},
		},
		{
			name:         "all data dropped alert firing",
			pipelineName: "cls",
			alerts: promv1.AlertsResult{
				Alerts: []promv1.Alert{
					{
						Labels: model.LabelSet{
							"alertname":     "LogFluentBitAllDataDropped",
							"pipeline_name": "cls",
						},
						State: promv1.AlertStateFiring,
					},
				},
			},
			expected: FluentBitProbeResult{
				PipelineProbeResult: PipelineProbeResult{
					AllDataDropped: true,
				},
			},
		},
		{
			name:         "some data dropped alert firing",
			pipelineName: "cls",
			alerts: promv1.AlertsResult{
				Alerts: []promv1.Alert{
					{
						Labels: model.LabelSet{
							"alertname":     "LogFluentBitSomeDataDropped",
							"pipeline_name": "cls",
						},
						State: promv1.AlertStateFiring,
					},
				},
			},
			expected: FluentBitProbeResult{
				PipelineProbeResult: PipelineProbeResult{
					SomeDataDropped: true,
				},
			},
		},
		{
			name:         "no logs delivered firing",
			pipelineName: "cls",
			alerts: promv1.AlertsResult{
				Alerts: []promv1.Alert{
					{
						Labels: model.LabelSet{
							"alertname":     "LogFluentBitNoLogsDelivered",
							"pipeline_name": "cls",
						},
						State: promv1.AlertStateFiring,
					},
				},
			},
			expected: FluentBitProbeResult{
				NoLogsDelivered: true,
			},
		},
		{
			name:         "buffer in use firing",
			pipelineName: "cls",
			alerts: promv1.AlertsResult{
				Alerts: []promv1.Alert{
					{
						Labels: model.LabelSet{
							"alertname":     "LogFluentBitBufferInUse",
							"pipeline_name": "cls",
						},
						State: promv1.AlertStateFiring,
					},
				},
			},
			expected: FluentBitProbeResult{
				BufferFillingUp: true,
			},
		},
		{
			name:         "no alerts firing",
			pipelineName: "cls",
			alerts: promv1.AlertsResult{
				Alerts: []promv1.Alert{},
			},
			expected: FluentBitProbeResult{
				PipelineProbeResult: PipelineProbeResult{
					Healthy: true,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sut, err := NewFluentBitProber(types.NamespacedName{})
			require.NoError(t, err)

			alertGetterMock := &mocks.AlertGetter{}
			if tc.alertsErr != nil {
				alertGetterMock.On("Alerts", mock.Anything).Return(promv1.AlertsResult{}, tc.alertsErr)
			} else {
				alertGetterMock.On("Alerts", mock.Anything).Return(tc.alerts, nil)
			}

			sut.getter = alertGetterMock

			result, err := sut.Probe(t.Context(), tc.pipelineName)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}
