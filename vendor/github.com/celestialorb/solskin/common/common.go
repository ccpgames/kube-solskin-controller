package common

import (
	"fmt"
	config "github.com/micro/go-config"
	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"reflect"
	"regexp"
	"strings"
)

// GetServiceAnnotation retrieves the annotations made by this service.
func GetServiceAnnotation(m meta.ObjectMeta, a string) string {
	label := fmt.Sprintf("celestialorb/solskin.%s", a)
	return m.Annotations[label]
}

// SetServiceAnnotation sets the annotations made by this service.
func SetServiceAnnotation(m meta.ObjectMeta, a string, b string) meta.ObjectMeta {
	label := fmt.Sprintf("celestialorb/solskin.%s", a)
	if m.Annotations == nil {
		m.Annotations = map[string]string{
			label: b,
		}
	}

	m.Annotations[label] = b
	return m
}

// GetPodSpec will extract the pod specification from any type of kubernetes
// resource and return it.
func GetPodSpec(obj interface{}) *core.PodSpec {
	_, ktype := GetObjectMeta(obj)
	switch ktype {
	case "pod":
		return &obj.(*core.Pod).Spec
	case "deployment":
		return &obj.(*apps.Deployment).Spec.Template.Spec
	case "daemonset":
		return &obj.(*apps.DaemonSet).Spec.Template.Spec
	case "statefulset":
		return &obj.(*apps.StatefulSet).Spec.Template.Spec
	case "job":
		return &obj.(*batch.Job).Spec.Template.Spec
	}

	return &core.PodSpec{}
}

// IsEligible determines whether or not the object is eligible for monitoring
// and suppression based on the given configuration.
func IsEligible(obj interface{}, cfg config.Config) bool {
	// Grab the object's metadata.
	m, _ := GetObjectMeta(obj)

	// Extract the pattern from the service configuration.
	p := cfg.Get("eligibility", "exclude_namespace").String("^kube-")

	// Run the regexp against the namespace of the resource.
	match, err := regexp.MatchString(p, m.Namespace)
	if err != nil {
		log.Fatal(err)
	}

	// If we have a match, then the resource isn't eligible.
	return !match
}

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
	if len(spec.Containers) <= 0 {
		return false
	}

	for _, container := range spec.Containers {
		probe := container.LivenessProbe
		if probe == nil {
			return false
		}

		h := probe.Handler
		if !hasDefinedHandler(h) {
			return false
		}
	}
	return true
}

// HasReadiness determines if the spec has proper readiness probes.
func HasReadiness(spec core.PodSpec) bool {
	if len(spec.Containers) <= 0 {
		return false
	}

	for _, container := range spec.Containers {
		probe := container.ReadinessProbe
		if probe == nil {
			return false
		}

		h := probe.Handler
		if !hasDefinedHandler(h) {
			return false
		}
	}
	return true
}

func hasDefinedHandler(h core.Handler) bool {
	if h.Exec != nil {
		return true
	}
	if h.HTTPGet != nil {
		return true
	}
	if h.TCPSocket != nil {
		return true
	}
	return false
}

// HasLimits determines if the spec has a proper resource limits.
func HasLimits(spec core.PodSpec) bool {
	if len(spec.Containers) <= 0 {
		return false
	}

	for _, container := range spec.Containers {
		r := container.Resources.Limits
		if !hasAllLimits(r) {
			return false
		}
	}
	return true
}

func hasAllLimits(r core.ResourceList) bool {
	keys := []core.ResourceName{
		core.ResourceCPU,
		core.ResourceMemory,
	}
	for _, k := range keys {
		_, ok := r[k]
		if !ok {
			return false
		}
	}
	return true
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
