//go:build e2e

package traces

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

var _ = Describe(suite.ID(), Label(suite.LabelTraces), func() {
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

		backend := kitbackend.New(mockNs, kitbackend.SignalTypeTraces, kitbackend.WithTLS(*serverCerts))
		succeedingObjs = append(succeedingObjs, backend.K8sObjects()...)

		tracePipelineMissingCa := testutils.NewTracePipelineBuilder().
			WithName(missingCaPipelineName).
			WithOTLPOutput(
				testutils.OTLPEndpoint(backend.Endpoint()),
				testutils.OTLPClientTLS(&telemetryv1alpha1.OTLPTLS{
					Cert: &telemetryv1alpha1.ValueType{Value: clientCerts.ClientCertPem.String()},
					Key:  &telemetryv1alpha1.ValueType{Value: clientCerts.ClientKeyPem.String()},
				}),
			).
			Build()

		tracePipelineMissingCert := testutils.NewTracePipelineBuilder().
			WithName(missingCertPipelineName).
			WithOTLPOutput(
				testutils.OTLPEndpoint(backend.Endpoint()),
				testutils.OTLPClientTLS(&telemetryv1alpha1.OTLPTLS{
					CA:  &telemetryv1alpha1.ValueType{Value: clientCerts.CaCertPem.String()},
					Key: &telemetryv1alpha1.ValueType{Value: clientCerts.ClientKeyPem.String()},
				}),
			).
			Build()

		tracePipelineMissingKey := testutils.NewTracePipelineBuilder().
			WithName(missingKeyPipelineName).
			WithOTLPOutput(
				testutils.OTLPEndpoint(backend.Endpoint()),
				testutils.OTLPClientTLS(&telemetryv1alpha1.OTLPTLS{
					CA:   &telemetryv1alpha1.ValueType{Value: clientCerts.CaCertPem.String()},
					Cert: &telemetryv1alpha1.ValueType{Value: clientCerts.ClientCertPem.String()},
				}),
			).
			Build()

		tracePipelineMissingAll := testutils.NewTracePipelineBuilder().
			WithName(missingAllPipelineName).
			WithOTLPOutput(
				testutils.OTLPEndpoint(backend.Endpoint()),
				testutils.OTLPClientTLS(&telemetryv1alpha1.OTLPTLS{
					Insecure:           true,
					InsecureSkipVerify: true,
				}),
			).
			Build()

		tracePipelineMissingAllButCa := testutils.NewTracePipelineBuilder().
			WithName(missingAllButCaPipelineName).
			WithOTLPOutput(
				testutils.OTLPEndpoint(backend.Endpoint()),
				testutils.OTLPClientTLS(&telemetryv1alpha1.OTLPTLS{
					CA: &telemetryv1alpha1.ValueType{Value: clientCerts.CaCertPem.String()},
				}),
			).
			Build()

		succeedingObjs = append(succeedingObjs,
			telemetrygen.NewPod(mockNs, telemetrygen.SignalTypeTraces).K8sObject(),
			&tracePipelineMissingCa, &tracePipelineMissingAllButCa, &tracePipelineMissingAll,
		)

		failingObjs = append(failingObjs,
			&tracePipelineMissingKey, &tracePipelineMissingCert,
		)

		return succeedingObjs, failingObjs
	}

	Context("When a trace pipeline with missing TLS configuration parameters is created", Ordered, func() {
		BeforeAll(func() {
			k8sSucceedingObjects, k8sFailingObjects := makeResources()

			DeferCleanup(func() {
				Expect(kitk8s.DeleteObjects(k8sSucceedingObjects...)).Should(Succeed())
				Expect(kitk8s.DeleteObjects(k8sFailingObjects...)).
					Should(MatchError(ContainSubstring(notFoundError)))
			})
			Expect(kitk8s.CreateObjects(GinkgoT(), k8sSucceedingObjects...)).Should(Succeed())
			Expect(kitk8s.CreateObjects(GinkgoT(), k8sFailingObjects...)).
				Should(MatchError(ContainSubstring(tlsCrdValidationError)))
		})

		It("Should set ConfigurationGenerated condition to True in pipelines", func() {
			assert.TracePipelineHasCondition(GinkgoT(), missingCaPipelineName, metav1.Condition{
				Type:   conditions.TypeConfigurationGenerated,
				Status: metav1.ConditionTrue,
				Reason: conditions.ReasonGatewayConfigured,
			})

			assert.TracePipelineHasCondition(GinkgoT(), missingAllButCaPipelineName, metav1.Condition{
				Type:   conditions.TypeConfigurationGenerated,
				Status: metav1.ConditionTrue,
				Reason: conditions.ReasonGatewayConfigured,
			})

			assert.TracePipelineHasCondition(GinkgoT(), missingAllPipelineName, metav1.Condition{
				Type:   conditions.TypeConfigurationGenerated,
				Status: metav1.ConditionTrue,
				Reason: conditions.ReasonGatewayConfigured,
			})
		})

		It("Should set TelemetryFlowHealthy condition to True in pipelines", func() {
			assert.TracePipelineHasCondition(GinkgoT(), missingCaPipelineName, metav1.Condition{
				Type:   conditions.TypeFlowHealthy,
				Status: metav1.ConditionTrue,
				Reason: conditions.ReasonSelfMonFlowHealthy,
			})

			assert.TracePipelineHasCondition(GinkgoT(), missingAllButCaPipelineName, metav1.Condition{
				Type:   conditions.TypeFlowHealthy,
				Status: metav1.ConditionTrue,
				Reason: conditions.ReasonSelfMonFlowHealthy,
			})

			assert.TracePipelineHasCondition(GinkgoT(), missingAllPipelineName, metav1.Condition{
				Type:   conditions.TypeFlowHealthy,
				Status: metav1.ConditionTrue,
				Reason: conditions.ReasonSelfMonFlowHealthy,
			})
		})

		It("Should set TraceComponentsHealthy condition to True in Telemetry", func() {
			assert.TelemetryHasState(GinkgoT(), operatorv1alpha1.StateReady)
			assert.TelemetryHasCondition(GinkgoT(), suite.K8sClient, metav1.Condition{
				Type:   conditions.TypeTraceComponentsHealthy,
				Status: metav1.ConditionTrue,
				Reason: conditions.ReasonComponentsRunning,
			})
		})
	})
})
