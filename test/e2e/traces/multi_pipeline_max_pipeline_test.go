//go:build e2e

package traces

import (
	"fmt"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	telemetryv1alpha1 "github.com/kyma-project/telemetry-manager/apis/telemetry/v1alpha1"
	telemetrycontrollers "github.com/kyma-project/telemetry-manager/controllers/telemetry"
	"github.com/kyma-project/telemetry-manager/internal/conditions"
	testutils "github.com/kyma-project/telemetry-manager/internal/utils/test"
	"github.com/kyma-project/telemetry-manager/test/testkit/assert"
	kitk8s "github.com/kyma-project/telemetry-manager/test/testkit/k8s"
	"github.com/kyma-project/telemetry-manager/test/testkit/suite"
)

var _ = Describe(suite.ID(), Label(suite.LabelMaxPipeline), Ordered, func() {

	Context("When reaching the pipeline limit", Ordered, func() {
		const maxNumberOfTracePipelines = telemetrycontrollers.MaxPipelineCount
		var (
			pipelinesNames       = make([]string, 0, maxNumberOfTracePipelines)
			pipelineCreatedFirst *telemetryv1alpha1.TracePipeline
			pipelineCreatedLater *telemetryv1alpha1.TracePipeline
		)

		makeResources := func() []client.Object {
			var objs []client.Object
			for i := range maxNumberOfTracePipelines {
				pipelineName := fmt.Sprintf("%s-limit-%d", suite.ID(), i)
				pipeline := testutils.NewTracePipelineBuilder().WithName(pipelineName).Build()
				pipelinesNames = append(pipelinesNames, pipelineName)

				objs = append(objs, &pipeline)

				if i == 0 {
					pipelineCreatedFirst = &pipeline
				}
			}

			return objs
		}

		BeforeAll(func() {
			k8sObjects := makeResources()
			DeferCleanup(func() {
				k8sObjectsToDelete := slices.DeleteFunc(k8sObjects, func(obj client.Object) bool {
					return obj.GetName() == pipelineCreatedFirst.GetName() // first pipeline is deleted separately in one of the specs
				})
				k8sObjectsToDelete = append(k8sObjectsToDelete, pipelineCreatedLater)
				Expect(kitk8s.DeleteObjects(k8sObjectsToDelete...)).Should(Succeed())
			})
			Expect(kitk8s.CreateObjects(GinkgoT(), k8sObjects...)).Should(Succeed())
		})

		It("Should have only running pipelines", func() {
			for _, pipelineName := range pipelinesNames {
				assert.TracePipelineHealthy(GinkgoT(), pipelineName)
			}
		})

		It("Should set ConfigurationGenerated condition to False and TelemetryFlowHealthy condition to False", func() {
			By("Creating an additional pipeline", func() {
				pipelineName := suite.IDWithSuffix("limit-exceeding")
				pipeline := testutils.NewTracePipelineBuilder().WithName(pipelineName).Build()
				pipelineCreatedLater = &pipeline
				pipelinesNames = append(pipelinesNames, pipelineName)

				Expect(kitk8s.CreateObjects(GinkgoT(), &pipeline)).Should(Succeed())

				assert.TracePipelineHasCondition(GinkgoT(), pipelineName, metav1.Condition{
					Type:   conditions.TypeConfigurationGenerated,
					Status: metav1.ConditionFalse,
					Reason: conditions.ReasonMaxPipelinesExceeded,
				})

				assert.TracePipelineHasCondition(GinkgoT(), pipelineName, metav1.Condition{
					Type:   conditions.TypeFlowHealthy,
					Status: metav1.ConditionFalse,
					Reason: conditions.ReasonSelfMonConfigNotGenerated,
				})
			})
		})

		It("Should have only running pipelines", func() {
			By("Deleting a pipeline", func() {
				Expect(kitk8s.DeleteObjects(pipelineCreatedFirst)).Should(Succeed())

				for _, pipeline := range pipelinesNames[1:] {
					assert.TracePipelineHealthy(GinkgoT(), pipeline)
				}
			})
		})
	})

})
