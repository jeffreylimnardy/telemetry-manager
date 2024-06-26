package otelcollector

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/telemetry-manager/internal/k8sutils"
	"github.com/kyma-project/telemetry-manager/internal/otelcollector/ports"
	commonresources "github.com/kyma-project/telemetry-manager/internal/resources/common"
)

// applyCommonResources applies resources to gateway and agent deployment node
func applyCommonResources(ctx context.Context, c client.Client, name types.NamespacedName, clusterRole *rbacv1.ClusterRole, allowedPorts []int32) error {
	// Create RBAC resources in the following order: service account, cluster role, cluster role binding.
	if err := k8sutils.CreateOrUpdateServiceAccount(ctx, c, makeServiceAccount(name)); err != nil {
		return fmt.Errorf("failed to create service account: %w", err)
	}

	if err := k8sutils.CreateOrUpdateClusterRole(ctx, c, clusterRole); err != nil {
		return fmt.Errorf("failed to create cluster role: %w", err)
	}

	if err := k8sutils.CreateOrUpdateClusterRoleBinding(ctx, c, makeClusterRoleBinding(name)); err != nil {
		return fmt.Errorf("failed to create cluster role binding: %w", err)
	}

	if err := k8sutils.CreateOrUpdateService(ctx, c, makeMetricsService(name)); err != nil {
		return fmt.Errorf("failed to create metrics service: %w", err)
	}

	if err := k8sutils.CreateOrUpdateNetworkPolicy(ctx, c, commonresources.MakeNetworkPolicy(name, allowedPorts, defaultLabels(name.Name))); err != nil {
		return fmt.Errorf("failed to create network policy: %w", err)
	}

	return nil
}

func deleteCommonResources(ctx context.Context, c client.Client, name types.NamespacedName) error {
	objectMeta := metav1.ObjectMeta{
		Name:      name.Name,
		Namespace: name.Namespace,
	}

	// Attempt to clean up as many resources as possible and avoid early return when one of the deletions fails
	var allErrors error = nil
	clusterRoleBinding := rbacv1.ClusterRoleBinding{ObjectMeta: objectMeta}
	if err := k8sutils.DeleteObject(ctx, c, &clusterRoleBinding); err != nil {
		allErrors = errors.Join(allErrors, fmt.Errorf("failed to delete cluster role binding: %w", err))
	}

	clusterRole := rbacv1.ClusterRole{ObjectMeta: objectMeta}
	if err := k8sutils.DeleteObject(ctx, c, &clusterRole); err != nil {
		allErrors = errors.Join(allErrors, fmt.Errorf("failed to delete cluster role: %w", err))
	}

	serviceAccount := corev1.ServiceAccount{ObjectMeta: objectMeta}
	if err := k8sutils.DeleteObject(ctx, c, &serviceAccount); err != nil {
		allErrors = errors.Join(allErrors, fmt.Errorf("failed to delete service account: %w", err))
	}

	metricsService := corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: name.Name + "-metrics", Namespace: name.Namespace}}
	if err := k8sutils.DeleteObject(ctx, c, &metricsService); err != nil {
		allErrors = errors.Join(allErrors, fmt.Errorf("failed to delete metrics service: %w", err))
	}

	networkPolicy := networkingv1.NetworkPolicy{ObjectMeta: objectMeta}
	if err := k8sutils.DeleteObject(ctx, c, &networkPolicy); err != nil {
		allErrors = errors.Join(allErrors, fmt.Errorf("failed to delete network policy: %w", err))
	}

	return allErrors
}

func defaultLabels(baseName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name": baseName,
	}
}

func makeServiceAccount(name types.NamespacedName) *corev1.ServiceAccount {
	serviceAccount := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
			Labels:    defaultLabels(name.Name),
		},
	}
	return &serviceAccount
}

func makeClusterRoleBinding(name types.NamespacedName) *rbacv1.ClusterRoleBinding {
	clusterRoleBinding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
			Labels:    defaultLabels(name.Name),
		},
		Subjects: []rbacv1.Subject{{Name: name.Name, Namespace: name.Namespace, Kind: rbacv1.ServiceAccountKind}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     name.Name,
		},
	}
	return &clusterRoleBinding
}

func makeConfigMap(name types.NamespacedName, collectorConfigYAML string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
			Labels:    defaultLabels(name.Name),
		},
		Data: map[string]string{
			configMapKey: collectorConfigYAML,
		},
	}
}

func makeSecret(name types.NamespacedName, secretData map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
			Labels:    defaultLabels(name.Name),
		},
		Data: secretData,
	}
}

func makeMetricsService(name types.NamespacedName) *corev1.Service {
	labels := defaultLabels(name.Name)
	selectorLabels := make(map[string]string)
	maps.Copy(selectorLabels, labels)
	labels["telemetry.kyma-project.io/self-monitor"] = "enabled"

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name + "-metrics",
			Namespace: name.Namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   strconv.Itoa(ports.Metrics),
				"prometheus.io/scheme": "http",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http-metrics",
					Protocol:   corev1.ProtocolTCP,
					Port:       ports.Metrics,
					TargetPort: intstr.FromInt32(ports.Metrics),
				},
			},
			Selector: selectorLabels,
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
}
