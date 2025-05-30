package fluentbit

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/kyma-project/telemetry-manager/test/testkit/matchers"
)

// FlatLog holds all needed information about a FluentBit log record.
// Gomega doesn't handle deeply nested data structure very well and generates large, unreadable diffs when paired with the deeply nested structure of plogs.
//
// Introducing a go struct with a flat data structure by extracting necessary information from different levels of plogs makes accessing the information easier than using plog.
// Logs directly and improves the readability of the test output logs.
type FlatLog struct {
	Attributes                     map[string]string
	LogBody                        string
	KubernetesAttributes           map[string]string
	KubernetesLabelAttributes      map[string]any
	KubernetesAnnotationAttributes map[string]any
}

func unmarshalLogs(jsonLogs []byte) ([]plog.Logs, error) {
	return matchers.UnmarshalOTLPJSONData(jsonLogs, func(buf []byte) (plog.Logs, error) {
		var unmarshaler plog.JSONUnmarshaler
		return unmarshaler.UnmarshalLogs(buf)
	})
}

// flattenAllLogs flattens an array of pdata.Logs log record to a slice of FlatLogFluentBit.
// It converts the deeply nested pdata.Logs data structure to a flat struct, to make it more readable in the test output logs.
func flattenAllLogs(lds []plog.Logs) []FlatLog {
	var flatLogs []FlatLog

	for _, ld := range lds {
		flatLogs = append(flatLogs, flattenLogs(ld)...)
	}

	return flatLogs
}

// flattenLogs converts a single pdata.Log log record to a slice of FlatLogFluentBit
// It takes relevant information from different levels of pdata and puts it into a FlatLogFluentBit go struct.
func flattenLogs(ld plog.Logs) []FlatLog {
	var flatLogs []FlatLog

	for i := range ld.ResourceLogs().Len() {
		resourceLogs := ld.ResourceLogs().At(i)
		for j := range resourceLogs.ScopeLogs().Len() {
			scopeLogs := resourceLogs.ScopeLogs().At(j)
			for k := range scopeLogs.LogRecords().Len() {
				lr := scopeLogs.LogRecords().At(k)
				k8sAttrs := getKubernetesAttributes(lr)
				flatLogs = append(flatLogs, FlatLog{
					Attributes:                     attributesToMap(lr.Attributes()),
					LogBody:                        lr.Body().AsString(),
					KubernetesAttributes:           attributesToMap(k8sAttrs),
					KubernetesLabelAttributes:      getNestedKubernetesAttributes("labels", k8sAttrs),
					KubernetesAnnotationAttributes: getNestedKubernetesAttributes("annotations", k8sAttrs),
				})
			}
		}
	}

	return flatLogs
}

// attributesToMap converts pdata.AttributeMap to a map using the string representation of the values.
func attributesToMap(attrs pcommon.Map) map[string]string {
	attrMap := make(map[string]string)

	attrs.Range(func(k string, v pcommon.Value) bool {
		// only take if value is not of type map, to reduce nesting and avoid duplication of kubernetes attributes
		if v.Type() != pcommon.ValueTypeMap {
			attrMap[k] = v.AsString()
		}

		return true
	})

	return attrMap
}

// getNestedKubernetesAttributes converts the kubernetes attributes from a LogRecord which are of type
// ValueTypeMap into a map using the string representation of the keys and any representation of the values
func getNestedKubernetesAttributes(key string, attrs pcommon.Map) map[string]any {
	attr, hasAttr := attrs.Get(key)
	if !hasAttr || attr.Type() != pcommon.ValueTypeMap {
		return nil
	}

	return attr.Map().AsRaw()
}

func getKubernetesAttributes(lr plog.LogRecord) pcommon.Map {
	const kubernetesAttrKey = "kubernetes"

	kubernetesAttrs, hasKubernetesAttrs := lr.Attributes().Get(kubernetesAttrKey)
	if !hasKubernetesAttrs || kubernetesAttrs.Type() != pcommon.ValueTypeMap {
		return pcommon.NewMap()
	}

	return kubernetesAttrs.Map()
}
