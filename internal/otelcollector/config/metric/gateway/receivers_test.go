package gateway

import (
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	telemetryv1alpha1 "github.com/kyma-project/telemetry-manager/apis/telemetry/v1alpha1"
	testutils "github.com/kyma-project/telemetry-manager/internal/utils/test"
)

func TestReceivers(t *testing.T) {
	ctx := t.Context()
	fakeClient := fake.NewClientBuilder().Build()
	sut := Builder{Reader: fakeClient}

	t.Run("OTLP receiver", func(t *testing.T) {
		collectorConfig, _, err := sut.Build(
			ctx,
			[]telemetryv1alpha1.MetricPipeline{
				testutils.NewMetricPipelineBuilder().WithName("test").Build(),
			},
			BuildOptions{},
		)
		require.NoError(t, err)

		otlpReceiver := collectorConfig.Receivers.OTLP
		require.NotNil(t, otlpReceiver)
		require.Equal(t, "${MY_POD_IP}:4318", otlpReceiver.Protocols.HTTP.Endpoint)
		require.Equal(t, "${MY_POD_IP}:4317", otlpReceiver.Protocols.GRPC.Endpoint)
	})

	t.Run("kyma stats receiver", func(t *testing.T) {
		gatewayNamespace := "test-namespace"

		collectorConfig, _, err := sut.Build(
			ctx,
			[]telemetryv1alpha1.MetricPipeline{
				testutils.NewMetricPipelineBuilder().WithName("test").Build(),
			},
			BuildOptions{
				GatewayNamespace: gatewayNamespace,
			},
		)
		require.NoError(t, err)

		kymaStatsReceiver := collectorConfig.Receivers.KymaStatsReceiver
		require.Equal(t, "serviceAccount", kymaStatsReceiver.AuthType)
		require.Equal(t, "30s", kymaStatsReceiver.CollectionInterval)
		require.Len(t, kymaStatsReceiver.Resources, 4)

		expectedResources := []ModuleGVR{
			{
				Group:    "operator.kyma-project.io",
				Version:  "v1alpha1",
				Resource: "telemetries",
			},
			{
				Group:    "telemetry.kyma-project.io",
				Version:  "v1alpha1",
				Resource: "logpipelines",
			},
			{
				Group:    "telemetry.kyma-project.io",
				Version:  "v1alpha1",
				Resource: "metricpipelines",
			},
			{
				Group:    "telemetry.kyma-project.io",
				Version:  "v1alpha1",
				Resource: "tracepipelines",
			},
		}
		for i, expectedResource := range expectedResources {
			require.Equal(t, expectedResource.Group, kymaStatsReceiver.Resources[i].Group)
			require.Equal(t, expectedResource.Version, kymaStatsReceiver.Resources[i].Version)
			require.Equal(t, expectedResource.Resource, kymaStatsReceiver.Resources[i].Resource)
		}
	})
}
