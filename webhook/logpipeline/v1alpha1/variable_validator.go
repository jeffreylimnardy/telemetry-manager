package v1alpha1

import (
	"fmt"

	telemetryv1alpha1 "github.com/kyma-project/telemetry-manager/apis/telemetry/v1alpha1"
)

func validateVariables(logPipeline *telemetryv1alpha1.LogPipeline, logPipelines *telemetryv1alpha1.LogPipelineList) error {
	if len(logPipeline.Spec.Variables) == 0 {
		return nil
	}

	for _, l := range logPipelines.Items {
		if l.Name != logPipeline.Name {
			for _, variable := range l.Spec.Variables {
				err := findConflictingVariables(logPipeline, variable, l.Name)
				if err != nil {
					return err
				}
			}
		}
	}

	for _, variable := range logPipeline.Spec.Variables {
		if validateMandatoryFieldsAreEmpty(variable) {
			return fmt.Errorf("mandatory field variable name or secretKeyRef name or secretKeyRef namespace or secretKeyRef key cannot be empty")
		}
	}

	return nil
}

func validateMandatoryFieldsAreEmpty(vr telemetryv1alpha1.LogPipelineVariableRef) bool {
	secretKey := vr.ValueFrom.SecretKeyRef
	return len(vr.Name) == 0 || len(secretKey.Key) == 0 || len(secretKey.Namespace) == 0 || len(secretKey.Name) == 0
}

func findConflictingVariables(logPipeLine *telemetryv1alpha1.LogPipeline, vr telemetryv1alpha1.LogPipelineVariableRef, existingPipelineName string) error {
	for _, v := range logPipeLine.Spec.Variables {
		if v.Name == vr.Name {
			return fmt.Errorf("variable name must be globally unique: variable '%s' is used in pipeline '%s'", v.Name, existingPipelineName)
		}
	}

	return nil
}
