//go:build e2e

package metrics

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/kyma-project/telemetry-manager/apis/operator/v1alpha1"
	telemetryv1alpha1 "github.com/kyma-project/telemetry-manager/apis/telemetry/v1alpha1"
	"github.com/kyma-project/telemetry-manager/internal/conditions"
	testutils "github.com/kyma-project/telemetry-manager/internal/utils/test"
	"github.com/kyma-project/telemetry-manager/test/testkit/assert"
	kitk8s "github.com/kyma-project/telemetry-manager/test/testkit/k8s"
	kitbackend "github.com/kyma-project/telemetry-manager/test/testkit/mocks/backend"
	"github.com/kyma-project/telemetry-manager/test/testkit/mocks/telemetrygen"
	"github.com/kyma-project/telemetry-manager/test/testkit/suite"
)

var _ = Describe(suite.ID(), Label(suite.LabelMetrics), Label(suite.LabelSetB), Ordered, func() {
	const (
		tlsCrdValidationError = "Can define either both 'cert' and 'key', or neither"
		notFoundError         = "not found"
	)

	var (
		mockNs                      = suite.ID()
		missingCaPipelineName       = suite.IDWithSuffix("-missing-ca")
		missingCertPipelineName     = suite.IDWithSuffix("-missing-cert")
		missingKeyPipelineName      = suite.IDWithSuffix("-missing-key")
		missingAllPipelineName      = suite.IDWithSuffix("-missing-all")
		missingAllButCaPipelineName = suite.IDWithSuffix("-missing-all-but-ca")
	)

	makeResources := func() ([]client.Object, []client.Object) {
		var succeedingObjs []client.Object
		var failingObjs []client.Object
		succeedingObjs = append(succeedingObjs, kitk8s.NewNamespace(mockNs).K8sObject())

		serverCerts, clientCerts, err := testutils.NewCertBuilder(kitbackend.DefaultName, mockNs).Build()
		Expect(err).ToNot(HaveOccurred())

		backend := kitbackend.New(mockNs, kitbackend.SignalTypeMetrics, kitbackend.WithTLS(*serverCerts))
		succeedingObjs = append(succeedingObjs, backend.K8sObjects()...)

		metricPipelineMissingCa := testutils.NewMetricPipelineBuilder().
			WithName(missingCaPipelineName).
			WithOTLPOutput(
				testutils.OTLPEndpoint(backend.Endpoint()),
				testutils.OTLPClientTLS(&telemetryv1alpha1.OTLPTLS{
					Cert: &telemetryv1alpha1.ValueType{Value: clientCerts.ClientCertPem.String()},
					Key:  &telemetryv1alpha1.ValueType{Value: clientCerts.ClientKeyPem.String()},
				}),
			).
			Build()

		metricPipelineMissingCert := testutils.NewMetricPipelineBuilder().
			WithName(missingCertPipelineName).
			WithOTLPOutput(
				testutils.OTLPEndpoint(backend.Endpoint()),
				testutils.OTLPClientTLS(&telemetryv1alpha1.OTLPTLS{
					CA:  &telemetryv1alpha1.ValueType{Value: clientCerts.CaCertPem.String()},
					Key: &telemetryv1alpha1.ValueType{Value: clientCerts.ClientKeyPem.String()},
				}),
			).
			Build()

		metricPipelineMissingKey := testutils.NewMetricPipelineBuilder().
			WithName(missingKeyPipelineName).
			WithOTLPOutput(
				testutils.OTLPEndpoint(backend.Endpoint()),
				testutils.OTLPClientTLS(&telemetryv1alpha1.OTLPTLS{
					CA:   &telemetryv1alpha1.ValueType{Value: clientCerts.CaCertPem.String()},
					Cert: &telemetryv1alpha1.ValueType{Value: clientCerts.ClientCertPem.String()},
				}),
			).
			Build()

		metricPipelineMissingAll := testutils.NewMetricPipelineBuilder().
			WithName(missingAllPipelineName).
			WithOTLPOutput(
				testutils.OTLPEndpoint(backend.Endpoint()),
				testutils.OTLPClientTLS(&telemetryv1alpha1.OTLPTLS{
					Insecure:           true,
					InsecureSkipVerify: true,
				}),
			).
			Build()

		metricPipelineMissingAllButCa := testutils.NewMetricPipelineBuilder().
			WithName(missingAllButCaPipelineName).
			WithOTLPOutput(
				testutils.OTLPEndpoint(backend.Endpoint()),
				testutils.OTLPClientTLS(&telemetryv1alpha1.OTLPTLS{
					CA: &telemetryv1alpha1.ValueType{Value: clientCerts.CaCertPem.String()},
				}),
			).
			Build()

		succeedingObjs = append(succeedingObjs,
			telemetrygen.NewPod(mockNs, telemetrygen.SignalTypeMetrics).K8sObject(),
			&metricPipelineMissingCa, &metricPipelineMissingAllButCa, &metricPipelineMissingAll,
		)

		failingObjs = append(failingObjs,
			&metricPipelineMissingKey, &metricPipelineMissingCert,
		)

		return succeedingObjs, failingObjs
	}

	Context("When a metric pipeline with missing TLS configuration parameters is created", Ordered, func() {
		BeforeAll(func() {
			k8sSucceedingObjects, k8sFailingObjects := makeResources()

			DeferCleanup(func() {
				Expect(kitk8s.DeleteObjects(suite.Ctx, k8sSucceedingObjects...)).Should(Succeed())
				Expect(kitk8s.DeleteObjects(suite.Ctx, k8sFailingObjects...)).
					Should(MatchError(ContainSubstring(notFoundError)))
			})
			Expect(kitk8s.CreateObjects(suite.Ctx, k8sSucceedingObjects...)).Should(Succeed())
			Expect(kitk8s.CreateObjects(suite.Ctx, k8sFailingObjects...)).
				Should(MatchError(ContainSubstring(tlsCrdValidationError)))
		})

		It("Should set ConfigurationGenerated condition to True in pipelines", func() {
			assert.MetricPipelineHasCondition(suite.Ctx, missingCaPipelineName, metav1.Condition{
				Type:   conditions.TypeConfigurationGenerated,
				Status: metav1.ConditionTrue,
				Reason: conditions.ReasonGatewayConfigured,
			})

			assert.MetricPipelineHasCondition(suite.Ctx, missingAllButCaPipelineName, metav1.Condition{
				Type:   conditions.TypeConfigurationGenerated,
				Status: metav1.ConditionTrue,
				Reason: conditions.ReasonGatewayConfigured,
			})

			assert.MetricPipelineHasCondition(suite.Ctx, missingAllPipelineName, metav1.Condition{
				Type:   conditions.TypeConfigurationGenerated,
				Status: metav1.ConditionTrue,
				Reason: conditions.ReasonGatewayConfigured,
			})
		})

		It("Should set TelemetryFlowHealthy condition to True in pipelines", func() {
			assert.MetricPipelineHasCondition(suite.Ctx, missingCaPipelineName, metav1.Condition{
				Type:   conditions.TypeFlowHealthy,
				Status: metav1.ConditionTrue,
				Reason: conditions.ReasonSelfMonFlowHealthy,
			})

			assert.MetricPipelineHasCondition(suite.Ctx, missingAllButCaPipelineName, metav1.Condition{
				Type:   conditions.TypeFlowHealthy,
				Status: metav1.ConditionTrue,
				Reason: conditions.ReasonSelfMonFlowHealthy,
			})

			assert.MetricPipelineHasCondition(suite.Ctx, missingAllPipelineName, metav1.Condition{
				Type:   conditions.TypeFlowHealthy,
				Status: metav1.ConditionTrue,
				Reason: conditions.ReasonSelfMonFlowHealthy,
			})
		})

		It("Should set MetricComponentsHealthy condition to True in Telemetry", func() {
			assert.TelemetryHasState(suite.Ctx, operatorv1alpha1.StateReady)
			assert.TelemetryHasCondition(suite.Ctx, suite.K8sClient, metav1.Condition{
				Type:   conditions.TypeMetricComponentsHealthy,
				Status: metav1.ConditionTrue,
				Reason: conditions.ReasonComponentsRunning,
			})
		})
	})
})
