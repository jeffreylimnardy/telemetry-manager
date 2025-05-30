//go:build e2e

package metrics

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testutils "github.com/kyma-project/telemetry-manager/internal/utils/test"
	"github.com/kyma-project/telemetry-manager/test/testkit/assert"
	kitk8s "github.com/kyma-project/telemetry-manager/test/testkit/k8s"
	kitkyma "github.com/kyma-project/telemetry-manager/test/testkit/kyma"
	"github.com/kyma-project/telemetry-manager/test/testkit/metrics/runtime"
	kitbackend "github.com/kyma-project/telemetry-manager/test/testkit/mocks/backend"
	"github.com/kyma-project/telemetry-manager/test/testkit/mocks/prommetricgen"
	"github.com/kyma-project/telemetry-manager/test/testkit/mocks/telemetrygen"
	"github.com/kyma-project/telemetry-manager/test/testkit/suite"
)

var _ = Describe(suite.ID(), Label(suite.LabelMetrics), Label(suite.LabelSetC), Ordered, func() {
	var (
		mockNs            = suite.ID()
		app1Ns            = "app-1"
		app2Ns            = "app-2"
		backend1Name      = "backend-1"
		backend1ExportURL string
		backend2Name      = "backend-2"
		backend2ExportURL string
	)

	makeResources := func() []client.Object {
		var objs []client.Object
		objs = append(objs, kitk8s.NewNamespace(mockNs).K8sObject(),
			kitk8s.NewNamespace(app1Ns).K8sObject(),
			kitk8s.NewNamespace(app2Ns).K8sObject())

		backend1 := kitbackend.New(mockNs, kitbackend.SignalTypeMetrics, kitbackend.WithName(backend1Name))
		backend1ExportURL = backend1.ExportURL(suite.ProxyClient)
		objs = append(objs, backend1.K8sObjects()...)

		pipelineIncludeApp1Ns := testutils.NewMetricPipelineBuilder().
			WithName("include-"+app1Ns).
			WithPrometheusInput(true, testutils.IncludeNamespaces(app1Ns)).
			WithRuntimeInput(true, testutils.IncludeNamespaces(app1Ns)).
			WithOTLPInput(true, testutils.IncludeNamespaces(app1Ns)).
			WithOTLPOutput(testutils.OTLPEndpoint(backend1.Endpoint())).
			Build()
		objs = append(objs, &pipelineIncludeApp1Ns)

		backend2 := kitbackend.New(mockNs, kitbackend.SignalTypeMetrics, kitbackend.WithName(backend2Name))
		backend2ExportURL = backend2.ExportURL(suite.ProxyClient)
		objs = append(objs, backend2.K8sObjects()...)

		pipelineExcludeApp1Ns := testutils.NewMetricPipelineBuilder().
			WithName("exclude-"+app1Ns).
			WithPrometheusInput(true, testutils.ExcludeNamespaces(app1Ns)).
			WithRuntimeInput(true, testutils.ExcludeNamespaces(app1Ns)).
			WithOTLPInput(true, testutils.ExcludeNamespaces(app1Ns)).
			WithOTLPOutput(testutils.OTLPEndpoint(backend2.Endpoint())).
			Build()
		objs = append(objs, &pipelineExcludeApp1Ns)

		objs = append(objs,
			telemetrygen.NewPod(app1Ns, telemetrygen.SignalTypeMetrics).K8sObject(),
			telemetrygen.NewPod(app2Ns, telemetrygen.SignalTypeMetrics).K8sObject(),
			telemetrygen.NewPod(kitkyma.SystemNamespaceName, telemetrygen.SignalTypeMetrics).K8sObject(),

			prommetricgen.New(app1Ns).Pod().WithPrometheusAnnotations(prommetricgen.SchemeHTTP).K8sObject(),
			prommetricgen.New(app2Ns).Pod().WithPrometheusAnnotations(prommetricgen.SchemeHTTP).K8sObject(),
			prommetricgen.New(kitkyma.SystemNamespaceName).Pod().WithPrometheusAnnotations(prommetricgen.SchemeHTTP).K8sObject(),
		)

		return objs
	}

	Context("When multiple metricpipelines exist", Ordered, func() {
		BeforeAll(func() {
			k8sObjects := makeResources()

			DeferCleanup(func() {
				Expect(kitk8s.DeleteObjects(suite.Ctx, k8sObjects...)).Should(Succeed())
			})

			Expect(kitk8s.CreateObjects(suite.Ctx, k8sObjects...)).Should(Succeed())
		})

		It("Should have a running metric gateway deployment", func() {
			assert.DeploymentReady(suite.Ctx, kitkyma.MetricGatewayName)
		})

		It("Should have a running metric agent daemonset", func() {
			assert.DaemonSetReady(suite.Ctx, kitkyma.MetricAgentName)
		})

		It("Should have a metrics backend running", func() {
			assert.DeploymentReady(suite.Ctx, types.NamespacedName{Name: backend1Name, Namespace: mockNs})
			assert.DeploymentReady(suite.Ctx, types.NamespacedName{Name: backend2Name, Namespace: mockNs})
		})

		// verify metrics from apps1Ns delivered to backend1
		It("Should deliver Runtime metrics from app1Ns to backend1", func() {
			assert.MetricsFromNamespaceDelivered(suite.ProxyClient, backend1ExportURL, app1Ns, runtime.DefaultMetricsNames)
		})

		It("Should deliver Prometheus metrics from app1Ns to backend1", func() {
			assert.MetricsFromNamespaceDelivered(suite.ProxyClient, backend1ExportURL, app1Ns, prommetricgen.CustomMetricNames())
		})

		It("Should deliver OTLP metrics from app1Ns to backend1", func() {
			assert.MetricsFromNamespaceDelivered(suite.ProxyClient, backend1ExportURL, app1Ns, telemetrygen.MetricNames)
		})

		It("Should not deliver metrics from app2Ns to backend1", func() {
			assert.MetricsFromNamespaceNotDelivered(suite.ProxyClient, backend1ExportURL, app2Ns)
		})

		// verify metrics from apps2Ns delivered to backend2
		It("Should deliver Runtime metrics from app2Ns to backend2", func() {
			assert.MetricsFromNamespaceDelivered(suite.ProxyClient, backend2ExportURL, app2Ns, runtime.DefaultMetricsNames)
		})

		It("Should deliver Prometheus metrics from app2Ns to backend2", func() {
			assert.MetricsFromNamespaceDelivered(suite.ProxyClient, backend2ExportURL, app2Ns, prommetricgen.CustomMetricNames())
		})

		It("Should deliver OTLP metrics from app2Ns to backend2", func() {
			assert.MetricsFromNamespaceDelivered(suite.ProxyClient, backend2ExportURL, app2Ns, telemetrygen.MetricNames)
		})

		It("Should not deliver metrics from app1Ns to backend2", func() {
			assert.MetricsFromNamespaceNotDelivered(suite.ProxyClient, backend2ExportURL, app1Ns)
		})
	})
})
