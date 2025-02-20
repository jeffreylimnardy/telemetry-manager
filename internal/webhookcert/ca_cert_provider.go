package webhookcert

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	k8sutils "github.com/kyma-project/telemetry-manager/internal/utils/k8s"
)

const (
	caCertFile = "ca.crt"
	caKeyFile  = "ca.key"
)

type caCertGenerator interface {
	generateCert() (certPEM, keyPEM []byte, err error)
}

type caCertProviderImpl struct {
	client           client.Client
	expiryChecker    certExpiryChecker
	keyLengthChecker keyLengthChecker
	generator        caCertGenerator
}

func newCACertProvider(client client.Client) *caCertProviderImpl {
	clock := realClock{}

	const duration30d = 30 * 24 * time.Hour

	return &caCertProviderImpl{
		client:        client,
		expiryChecker: &certExpiryCheckerImpl{clock: realClock{}, softExpiryOffset: duration30d},
		generator: &caCertGeneratorImpl{
			clock: clock,
		},
		keyLengthChecker: &keyLengthCheckerImpl{},
	}
}

func (p *caCertProviderImpl) provideCert(ctx context.Context, caSecretName types.NamespacedName) ([]byte, []byte, error) {
	var caSecret corev1.Secret

	var shouldCreateNew bool

	if err := p.client.Get(ctx, caSecretName, &caSecret); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, nil, fmt.Errorf("failed to find ca cert caSecretName: %w", err)
		}

		shouldCreateNew = true
	} else {
		shouldCreateNew = !p.checkCASecret(ctx, &caSecret)
	}

	if shouldCreateNew {
		logf.FromContext(ctx).Info("Generating new CA cert/key",
			"secretName", caSecretName.Name,
			"secretNamespace", caSecretName.Namespace)

		caCertPEM, caKeyPEM, err := p.generator.generateCert()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generateCert ca cert: %w", err)
		}

		newSecret := makeCASecret(caCertPEM, caKeyPEM, caSecretName)
		if err = k8sutils.CreateOrUpdateSecret(ctx, p.client, &newSecret); err != nil {
			return nil, nil, fmt.Errorf("failed to create ca cert caSecretName: %w", err)
		}

		return caCertPEM, caKeyPEM, nil
	}

	return caSecret.Data[caCertFile], caSecret.Data[caKeyFile], nil
}

func (p *caCertProviderImpl) checkCASecret(ctx context.Context, caSecret *corev1.Secret) bool {
	caCertPEM, caKeyPEM, err := p.fetchCACertAndKey(caSecret)
	if err != nil {
		logf.FromContext(ctx).Error(err, "Invalid ca secret. Creating a new one",
			"secretName", caSecret.Name,
			"secretNamespace", caSecret.Namespace)

		return false
	}

	certNotExpired, checkCertErr := p.expiryChecker.checkExpiry(ctx, caCertPEM)

	keyLengthValid, checkKeyErr := p.keyLengthChecker.checkKeyLength(ctx, caKeyPEM)

	return checkCertErr == nil && certNotExpired && checkKeyErr == nil && keyLengthValid
}

func (p *caCertProviderImpl) fetchCACertAndKey(caSecret *corev1.Secret) ([]byte, []byte, error) {
	var caCertPEM, caKeyPEM []byte
	if val, found := caSecret.Data[caCertFile]; found {
		caCertPEM = val
	} else {
		return nil, nil, fmt.Errorf("ca cert not found: %v", caCertFile)
	}

	if val, found := caSecret.Data[caKeyFile]; found {
		caKeyPEM = val
	} else {
		return nil, nil, fmt.Errorf("ca key not found: %v", caKeyFile)
	}

	return caCertPEM, caKeyPEM, nil
}

func makeCASecret(certificate []byte, key []byte, name types.NamespacedName) corev1.Secret {
	return corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
		},
		Data: map[string][]byte{
			caCertFile: certificate,
			caKeyFile:  key,
		},
	}
}
