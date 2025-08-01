package logparser

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	telemetryv1alpha1 "github.com/kyma-project/telemetry-manager/apis/telemetry/v1alpha1"
)

func TestEnsureFinalizer(t *testing.T) {
	t.Run("without DeletionTimestamp", func(t *testing.T) {
		ctx := t.Context()
		scheme := runtime.NewScheme()
		_ = telemetryv1alpha1.AddToScheme(scheme)
		parser := &telemetryv1alpha1.LogParser{ObjectMeta: metav1.ObjectMeta{Name: "parser"}}
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(parser).Build()

		err := ensureFinalizer(ctx, client, parser)
		require.NoError(t, err)

		var updatedParser telemetryv1alpha1.LogParser

		_ = client.Get(ctx, types.NamespacedName{Name: parser.Name}, &updatedParser)

		require.True(t, controllerutil.ContainsFinalizer(&updatedParser, finalizer))
	})

	t.Run("with DeletionTimestamp", func(t *testing.T) {
		ctx := t.Context()
		client := fake.NewClientBuilder().Build()
		timestamp := metav1.Now()
		parser := &telemetryv1alpha1.LogParser{
			ObjectMeta: metav1.ObjectMeta{
				DeletionTimestamp: &timestamp,
				Name:              "parser"}}

		err := ensureFinalizer(ctx, client, parser)
		require.NoError(t, err)
	})
}

func TestCleanupFinalizer(t *testing.T) {
	t.Run("without DeletionTimestamp", func(t *testing.T) {
		ctx := t.Context()
		parser := &telemetryv1alpha1.LogParser{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "parser",
				Finalizers: []string{finalizer},
			},
		}
		client := fake.NewClientBuilder().Build()

		err := cleanupFinalizerIfNeeded(ctx, client, parser)
		require.NoError(t, err)
	})

	t.Run("with DeletionTimestamp", func(t *testing.T) {
		ctx := t.Context()
		scheme := runtime.NewScheme()
		_ = telemetryv1alpha1.AddToScheme(scheme)
		timestamp := metav1.Now()
		parser := &telemetryv1alpha1.LogParser{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "parser",
				Finalizers:        []string{finalizer},
				DeletionTimestamp: &timestamp,
			},
		}
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(parser).Build()

		err := cleanupFinalizerIfNeeded(ctx, client, parser)
		require.NoError(t, err)

		var updatedParser telemetryv1alpha1.LogPipeline

		_ = client.Get(ctx, types.NamespacedName{Name: parser.Name}, &updatedParser)

		require.False(t, controllerutil.ContainsFinalizer(&updatedParser, finalizer))
	})
}
