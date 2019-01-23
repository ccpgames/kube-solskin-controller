package common

import (
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"strings"
)

// PassesChecks TODO
func PassesChecks(checks []bool) bool {
	// Iterate through all results and return false at the first sight of false.
	for _, check := range checks {
		if !check {
			return false
		}
	}

	// If we made it through without seeing false, then we passes all checks.
	return true
}

// HasObservability determines if a kubernetes resource is observable.
func HasObservability(objectMeta meta.ObjectMeta) bool {
	return hasAnnotation(objectMeta, "prometheus.io/scrape")
}

// HasLiveness determines if the spec has proper liveness probes.
func HasLiveness(spec core.PodSpec) bool {
	return false
}

// HasReadiness determines if the spec has proper readiness probes.
func HasReadiness(spec core.PodSpec) bool {
	return false
}

// HasLimits determines if the spec has a proper resource limits.
func HasLimits(spec core.PodSpec) bool {
	return false
}

// GetObjectMeta TODO
func GetObjectMeta(obj interface{}) (meta.ObjectMeta, string) {
	// Use reflection to determine resource type.
	// I don't like this but I can't find a better way of doing it at the moment.
	v := reflect.Indirect(reflect.ValueOf(obj))
	objectMeta := v.FieldByName("ObjectMeta").Interface().(meta.ObjectMeta)
	return objectMeta, strings.ToLower(v.Type().Name())
}

// Helper function to determine is a given annotation exists in the object's
// metadata.
func hasAnnotation(objectMeta meta.ObjectMeta, annotation string) bool {
	annotations := objectMeta.GetAnnotations()
	for key := range annotations {
		if key == annotation {
			return true
		}
	}

	return false
}

// BooleanToFloat64 is a helper function to convert a boolean value into a
// float64.
func BooleanToFloat64(value bool) float64 {
	if value {
		return 1.0
	}
	return 0.0
}
