package otlpexporter

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	telemetryv1alpha1 "github.com/kyma-project/telemetry-manager/apis/telemetry/v1alpha1"
	sharedtypesutils "github.com/kyma-project/telemetry-manager/internal/utils/sharedtypes"
	"github.com/kyma-project/telemetry-manager/internal/validators/secretref"
)

const (
	basicAuthHeaderVariablePrefix = "BASIC_AUTH_HEADER"
	otlpEndpointVariablePrefix    = "OTLP_ENDPOINT"
	tlsConfigCertVariablePrefix   = "OTLP_TLS_CERT_PEM"
	tlsConfigKeyVariablePrefix    = "OTLP_TLS_KEY_PEM"
	tlsConfigCaVariablePrefix     = "OTLP_TLS_CA_PEM"
)

var (
	ErrValueOrSecretRefUndefined = errors.New("either value or secret key reference must be defined")
)

func makeEnvVars(ctx context.Context, c client.Reader, output *telemetryv1alpha1.OTLPOutput, pipelineName string) (map[string][]byte, error) {
	var err error

	secretData := make(map[string][]byte)

	err = makeAuthenticationEnvVar(ctx, c, secretData, output, pipelineName)
	if err != nil {
		return nil, err
	}

	err = makeOTLPEndpointEnvVar(ctx, c, secretData, output, pipelineName)
	if err != nil {
		return nil, err
	}

	err = makeHeaderEnvVar(ctx, c, secretData, output, pipelineName)
	if err != nil {
		return nil, err
	}

	err = makeTLSEnvVar(ctx, c, secretData, output, pipelineName)
	if err != nil {
		return nil, err
	}

	return secretData, nil
}

func makeAuthenticationEnvVar(ctx context.Context, c client.Reader, secretData map[string][]byte, output *telemetryv1alpha1.OTLPOutput, pipelineName string) error {
	if output.Authentication != nil && sharedtypesutils.IsValid(&output.Authentication.Basic.User) && sharedtypesutils.IsValid(&output.Authentication.Basic.Password) {
		username, err := resolveValue(ctx, c, output.Authentication.Basic.User)
		if err != nil {
			return err
		}

		password, err := resolveValue(ctx, c, output.Authentication.Basic.Password)
		if err != nil {
			return err
		}

		basicAuthHeader := formatBasicAuthHeader(string(username), string(password))
		basicAuthHeaderVariable := fmt.Sprintf("%s_%s", basicAuthHeaderVariablePrefix, sanitizeEnvVarName(pipelineName))
		secretData[basicAuthHeaderVariable] = []byte(basicAuthHeader)
	}

	return nil
}

func makeOTLPEndpointEnvVar(ctx context.Context, c client.Reader, secretData map[string][]byte, output *telemetryv1alpha1.OTLPOutput, pipelineName string) error {
	otlpEndpointVariable := makeOTLPEndpointVariable(pipelineName)

	endpointURL, err := resolveEndpointURL(ctx, c, output)
	if err != nil {
		return err
	}

	secretData[otlpEndpointVariable] = endpointURL

	return err
}

func makeHeaderEnvVar(ctx context.Context, c client.Reader, secretData map[string][]byte, output *telemetryv1alpha1.OTLPOutput, pipelineName string) error {
	for _, header := range output.Headers {
		key := makeHeaderVariable(header, pipelineName)

		value, err := resolveValue(ctx, c, header.ValueType)
		if err != nil {
			return err
		}

		secretData[key] = prefixHeaderValue(header, value)
	}

	return nil
}

func makeTLSEnvVar(ctx context.Context, c client.Reader, secretData map[string][]byte, output *telemetryv1alpha1.OTLPOutput, pipelineName string) error {
	if output.TLS != nil {
		if sharedtypesutils.IsValid(output.TLS.CA) {
			ca, err := resolveValue(ctx, c, *output.TLS.CA)
			if err != nil {
				return err
			}

			tlsConfigCaVariable := makeTLSCaVariable(pipelineName)
			secretData[tlsConfigCaVariable] = ca
		}

		if sharedtypesutils.IsValid(output.TLS.Cert) && sharedtypesutils.IsValid(output.TLS.Key) {
			cert, err := resolveValue(ctx, c, *output.TLS.Cert)
			if err != nil {
				return err
			}

			key, err := resolveValue(ctx, c, *output.TLS.Key)
			if err != nil {
				return err
			}

			// Make a best effort replacement of linebreaks in cert/key if present.
			sanitizedCert := bytes.ReplaceAll(cert, []byte("\\n"), []byte("\n"))
			sanitizedKey := bytes.ReplaceAll(key, []byte("\\n"), []byte("\n"))

			tlsConfigCertVariable := makeTLSCertVariable(pipelineName)
			secretData[tlsConfigCertVariable] = sanitizedCert

			tlsConfigKeyVariable := makeTLSKeyVariable(pipelineName)
			secretData[tlsConfigKeyVariable] = sanitizedKey
		}
	}

	return nil
}

func prefixHeaderValue(header telemetryv1alpha1.Header, value []byte) []byte {
	if len(strings.TrimSpace(header.Prefix)) > 0 {
		return []byte(fmt.Sprintf("%s %s", strings.TrimSpace(header.Prefix), string(value)))
	}

	return value
}

func resolveEndpointURL(ctx context.Context, c client.Reader, output *telemetryv1alpha1.OTLPOutput) ([]byte, error) {
	endpoint, err := resolveValue(ctx, c, output.Endpoint)
	if err != nil {
		return nil, err
	}

	if len(output.Path) > 0 {
		u, err := url.Parse(string(endpoint))
		if err != nil {
			return nil, err
		}

		u.Path = path.Join(u.Path, output.Path)

		return []byte(u.String()), nil
	}

	return endpoint, nil
}

func formatBasicAuthHeader(username string, password string) string {
	return fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
}

func resolveValue(ctx context.Context, c client.Reader, value telemetryv1alpha1.ValueType) ([]byte, error) {
	if value.Value != "" {
		return []byte(value.Value), nil
	}

	if value.ValueFrom.SecretKeyRef != nil {
		return secretref.GetValue(ctx, c, *value.ValueFrom.SecretKeyRef)
	}

	return nil, ErrValueOrSecretRefUndefined
}

func makeOTLPEndpointVariable(pipelineName string) string {
	return fmt.Sprintf("%s_%s", otlpEndpointVariablePrefix, sanitizeEnvVarName(pipelineName))
}

func makeBasicAuthHeaderVariable(pipelineName string) string {
	return fmt.Sprintf("%s_%s", basicAuthHeaderVariablePrefix, sanitizeEnvVarName(pipelineName))
}

func makeHeaderVariable(header telemetryv1alpha1.Header, pipelineName string) string {
	return fmt.Sprintf("HEADER_%s_%s", sanitizeEnvVarName(pipelineName), sanitizeEnvVarName(header.Name))
}

func makeTLSCertVariable(pipelineName string) string {
	return fmt.Sprintf("%s_%s", tlsConfigCertVariablePrefix, sanitizeEnvVarName(pipelineName))
}

func makeTLSKeyVariable(pipelineName string) string {
	return fmt.Sprintf("%s_%s", tlsConfigKeyVariablePrefix, sanitizeEnvVarName(pipelineName))
}

func makeTLSCaVariable(pipelineName string) string {
	return fmt.Sprintf("%s_%s", tlsConfigCaVariablePrefix, sanitizeEnvVarName(pipelineName))
}

func sanitizeEnvVarName(input string) string {
	result := input
	result = strings.ToUpper(result)
	result = strings.ReplaceAll(result, ".", "_")
	result = strings.ReplaceAll(result, "-", "_")

	return result
}
