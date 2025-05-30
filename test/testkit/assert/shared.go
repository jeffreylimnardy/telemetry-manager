package assert

import (
	"context"
	"net/http"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"

	kitbackend "github.com/kyma-project/telemetry-manager/test/testkit/mocks/backend"
	"github.com/kyma-project/telemetry-manager/test/testkit/periodic"
	"github.com/kyma-project/telemetry-manager/test/testkit/suite"
)

func BackendDataEventuallyMatches(ctx context.Context, backend *kitbackend.Backend, httpBodyMatcher types.GomegaMatcher, optionalDescription ...any) {
	queryURL := suite.ProxyClient.ProxyURLForService(backend.Namespace(), backend.Name(), kitbackend.QueryPath, kitbackend.QueryPort)
	HTTPResponseEventuallyMatches(ctx, queryURL, httpBodyMatcher, optionalDescription...)
}

func BackendDataConsistentlyMatches(ctx context.Context, backend *kitbackend.Backend, httpBodyMatcher types.GomegaMatcher, optionalDescription ...any) {
	queryURL := suite.ProxyClient.ProxyURLForService(backend.Namespace(), backend.Name(), kitbackend.QueryPath, kitbackend.QueryPort)
	HTTPResponseConsistentlyMatches(ctx, queryURL, httpBodyMatcher, optionalDescription...)
}

func HTTPResponseEventuallyMatches(ctx context.Context, queryURL string, httpBodyMatcher types.GomegaMatcher, optionalDescription ...any) {
	Eventually(func(g Gomega) {
		resp, err := suite.ProxyClient.GetWithContext(ctx, queryURL)
		g.Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		g.Expect(resp).To(HaveHTTPStatus(http.StatusOK))
		g.Expect(resp).To(HaveHTTPBody(httpBodyMatcher), optionalDescription...)
	}, periodic.EventuallyTimeout, periodic.TelemetryInterval).Should(Succeed())
}

func HTTPResponseConsistentlyMatches(ctx context.Context, queryURL string, httpBodyMatcher types.GomegaMatcher, optionalDescription ...any) {
	Consistently(func(g Gomega) {
		resp, err := suite.ProxyClient.GetWithContext(ctx, queryURL)
		g.Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		g.Expect(resp).To(HaveHTTPStatus(http.StatusOK))
		g.Expect(resp).To(HaveHTTPBody(httpBodyMatcher), optionalDescription...)
	}, periodic.ConsistentlyTimeout, periodic.TelemetryInterval).Should(Succeed())
}
