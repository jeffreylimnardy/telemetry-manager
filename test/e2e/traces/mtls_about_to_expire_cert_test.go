//go:build e2e

package traces

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/kyma-project/telemetry-manager/apis/operator/v1alpha1"
	"github.com/kyma-project/telemetry-manager/internal/conditions"
	testutils "github.com/kyma-project/telemetry-manager/internal/utils/test"
	"github.com/kyma-project/telemetry-manager/test/testkit/assert"
	kitk8s "github.com/kyma-project/telemetry-manager/test/testkit/k8s"
	kitkyma "github.com/kyma-project/telemetry-manager/test/testkit/kyma"
	kitbackend "github.com/kyma-project/telemetry-manager/test/testkit/mocks/backend"
	"github.com/kyma-project/telemetry-manager/test/testkit/mocks/telemetrygen"
	"github.com/kyma-project/telemetry-manager/test/testkit/suite"
)

var _ = Describe(suite.ID(), Label(suite.LabelTraces), func() {
	var (
		mockNs           = suite.ID()
		pipelineName     = suite.ID()
		backendExportURL string
	)

	makeResources := func() []client.Object {
		var objs []client.Object
		objs = append(objs, kitk8s.NewNamespace(mockNs).K8sObject())

		serverCerts, clientCerts, err := testutils.NewCertBuilder(kitbackend.DefaultName, mockNs).
			WithAboutToExpireClientCert().
			Build()
		Expect(err).ToNot(HaveOccurred())

		backend := kitbackend.New(mockNs, kitbackend.SignalTypeTraces, kitbackend.WithTLS(*serverCerts))
		objs = append(objs, backend.K8sObjects()...)
		backendExportURL = backend.ExportURL(suite.ProxyClient)

		tracePipeline := testutils.NewTracePipelineBuilder().
			WithName(pipelineName).
			WithOTLPOutput(
				testutils.OTLPEndpoint(backend.Endpoint()),
				testutils.OTLPClientTLSFromString(
					clientCerts.CaCertPem.String(),
					clientCerts.ClientCertPem.String(),
					clientCerts.ClientKeyPem.String(),
				),
			).
			Build()

		objs = append(objs,
			telemetrygen.NewPod(mockNs, telemetrygen.SignalTypeTraces).K8sObject(),
			&tracePipeline,
		)

		return objs
	}

	Context("When a trace pipeline with TLS Cert expiring within 2 weeks is activated", Ordered, func() {
		BeforeAll(func() {
			k8sObjects := makeResources()

			DeferCleanup(func() {
				Expect(kitk8s.DeleteObjects(k8sObjects...)).Should(Succeed())
			})
			Expect(kitk8s.CreateObjects(GinkgoT(), k8sObjects...)).Should(Succeed())
		})

		It("Should have running pipelines", func() {
			assert.TracePipelineHealthy(GinkgoT(), pipelineName)
		})

		It("Should have a running trace gateway deployment", func() {
			assert.DeploymentReady(GinkgoT(), kitkyma.TraceGatewayName)
		})

		It("Should have a tlsCertificateAboutToExpire Condition set in pipeline conditions", func() {
			assert.TracePipelineHasCondition(GinkgoT(), pipelineName, metav1.Condition{
				Type:   conditions.TypeConfigurationGenerated,
				Status: metav1.ConditionTrue,
				Reason: conditions.ReasonTLSCertificateAboutToExpire,
			})
		})

		It("Should have telemetryCR showing correct condition in its status", func() {
			assert.TelemetryHasState(GinkgoT(), operatorv1alpha1.StateWarning)
			assert.TelemetryHasCondition(GinkgoT(), suite.K8sClient, metav1.Condition{
				Type:   conditions.TypeTraceComponentsHealthy,
				Status: metav1.ConditionTrue,
				Reason: conditions.ReasonTLSCertificateAboutToExpire,
			})
		})

		It("Should have a trace backend running", func() {
			assert.DeploymentReady(GinkgoT(), types.NamespacedName{Name: kitbackend.DefaultName, Namespace: mockNs})
		})

		It("Should deliver telemetrygen traces", func() {
			assert.TracesFromNamespaceDelivered(suite.ProxyClient, backendExportURL, mockNs)
		})
	})
})
