package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	config "github.com/micro/go-config"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/expfmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// TestMetricLiveness tests that the metric service correctly reports liveness
// values for kubernetes resources.
func TestMetricLiveness(t *testing.T) {
	t.Error("not yet implemented")
}

// TestMetricObservability tests that the metric service correctly reports
// observability values for kubernetes resources.
func TestMetricObservability(t *testing.T) {
	// Create a fake default configuration.
	cfg := config.NewConfig()

	// Create the fake client.
	client := fake.NewSimpleClientset()

	// Define our kubernetes resources.
	pods := []*corev1.Pod{
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{
			Name:      "without-obs",
			Namespace: "default",
		}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{
			Name:      "with-false-obs",
			Namespace: "default",
			Annotations: map[string]string{
				"prometheus.io/scrape": "false",
			},
		}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{
			Name:      "with-true-obs",
			Namespace: "default",
			Annotations: map[string]string{
				"prometheus.io/scrape": "true",
			},
		}},
	}
	deployments := []*appsv1.Deployment{
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "without-obs",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "without-obs",
						Namespace: "default",
					},
				},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-false-obs",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "with-false-obs",
						Namespace: "default",
						Annotations: map[string]string{
							"prometheus.io/scrape": "false",
						},
					},
				},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-true-obs",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "with-true-obs",
						Namespace: "default",
						Annotations: map[string]string{
							"prometheus.io/scrape": "true",
						},
					},
				},
			},
		},
	}
	daemonsets := []*appsv1.DaemonSet{
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "without-obs",
				Namespace: "default",
			},
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "without-obs",
						Namespace: "default",
					},
				},
			},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-false-obs",
				Namespace: "default",
			},
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "with-false-obs",
						Namespace: "default",
						Annotations: map[string]string{
							"prometheus.io/scrape": "false",
						},
					},
				},
			},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-true-obs",
				Namespace: "default",
			},
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "with-true-obs",
						Namespace: "default",
						Annotations: map[string]string{
							"prometheus.io/scrape": "true",
						},
					},
				},
			},
		},
	}

	// Setup resources in the cluster.
	setupKubernetesTestResources(t, client, pods, deployments, daemonsets)

	// Start the metric updater.
	startMetricUpdater(client, cfg)

	// Check our metrics.
	var subtests = []struct {
		expected float64
		metric   string
		labels   map[string]string
	}{
		{0.0, "solskin_observability_resources", map[string]string{
			"name":          pods[0].GetName(),
			"namespace":     pods[0].GetNamespace(),
			"resource_type": "pod",
		}},
		{1.0, "solskin_observability_resources", map[string]string{
			"name":          pods[1].GetName(),
			"namespace":     pods[1].GetNamespace(),
			"resource_type": "pod",
		}},
		{1.0, "solskin_observability_resources", map[string]string{
			"name":          pods[2].GetName(),
			"namespace":     pods[2].GetNamespace(),
			"resource_type": "pod",
		}},
		{0.0, "solskin_observability_resources", map[string]string{
			"name":          deployments[0].GetName(),
			"namespace":     deployments[0].GetNamespace(),
			"resource_type": "deployment",
		}},
		{1.0, "solskin_observability_resources", map[string]string{
			"name":          deployments[1].GetName(),
			"namespace":     deployments[1].GetNamespace(),
			"resource_type": "deployment",
		}},
		{1.0, "solskin_observability_resources", map[string]string{
			"name":          deployments[2].GetName(),
			"namespace":     deployments[2].GetNamespace(),
			"resource_type": "deployment",
		}},
		{0.0, "solskin_observability_resources", map[string]string{
			"name":          daemonsets[0].GetName(),
			"namespace":     daemonsets[0].GetNamespace(),
			"resource_type": "daemonset",
		}},
		{1.0, "solskin_observability_resources", map[string]string{
			"name":          daemonsets[1].GetName(),
			"namespace":     daemonsets[1].GetNamespace(),
			"resource_type": "daemonset",
		}},
		{1.0, "solskin_observability_resources", map[string]string{
			"name":          daemonsets[2].GetName(),
			"namespace":     daemonsets[2].GetNamespace(),
			"resource_type": "daemonset",
		}},
	}

	for _, subtest := range subtests {
		checkMetrics(t, subtest.metric, subtest.labels, subtest.expected)
	}
}

// TestMetricLimits tests that the metric service correctly reports limits
// values for kubernetes resources.
func TestMetricLimits(t *testing.T) {
	t.Error("not yet implemented")
}

// TestMetricsService perform a simple test of the service.
func TestMetricsService(t *testing.T) {
	// Create a fake default configuration.
	cfg := config.NewConfig()

	// Create the fake client.
	client := fake.NewSimpleClientset()

	// Add a pod.
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
	}

	_, err := client.Core().Pods("default").Create(pod)
	if err != nil {
		t.Error(err)
	}

	// Start the metric updater.
	startMetricUpdater(client, cfg)

	labels := map[string]string{
		"name":          "test",
		"namespace":     "default",
		"resource_type": "pod",
	}
	checkMetrics(t, "solskin_observability_resources", labels, 0.0)
}

// Helper function to establish resources in our fake kubernetes cluster.
func setupKubernetesTestResources(t *testing.T,
	client kubernetes.Interface,
	pods []*corev1.Pod,
	deployments []*appsv1.Deployment,
	daemonsets []*appsv1.DaemonSet,
) {
	// Add our pods to the cluster.
	for _, r := range pods {
		_, err := client.Core().Pods(r.GetNamespace()).Create(r)
		if err != nil {
			t.Error(err)
		}
	}

	// Add our deployments to the cluster.
	for _, r := range deployments {
		_, err := client.Apps().Deployments(r.GetNamespace()).Create(r)
		if err != nil {
			t.Error(err)
		}
	}

	// Add our daemonsets to the cluster.
	for _, r := range daemonsets {
		_, err := client.Apps().DaemonSets(r.GetNamespace()).Create(r)
		if err != nil {
			t.Error(err)
		}
	}
}

// A helper function to start the prometheus service, send a request, and check
// the value of a specific metric.
func checkMetrics(t *testing.T, name string, labels map[string]string, value float64) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/metrics", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := promhttp.Handler()

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var parser expfmt.TextParser
	families, err := parser.TextToMetricFamilies(rr.Body)
	if err != nil {
		t.Error(err)
	}

	for _, family := range families {
		// If it's not the metric family we're looking for, skip.
		if family.GetName() != name {
			continue
		}

		// Check each individual set of label pairings.
		metric := family.GetMetric()
		for _, m := range metric {
			labelPairs := m.GetLabel()
			dest := make(map[string]string, len(labelPairs))
			for _, pair := range labelPairs {
				dest[pair.GetName()] = pair.GetValue()
			}

			eq := reflect.DeepEqual(labels, dest)
			// If the labels aren't equal, continue to next submetric.
			if !eq {
				continue
			}

			// Otherwise we found the exact metric we're looking for.
			// Time to compare the value, assumes metric is a gauge type.
			if m.GetGauge().GetValue() != value {
				t.Errorf("value did not match expected")
			}
			return
		}
		t.Error("found metric family, but no label match")
		return
	}
	t.Errorf("could not find metric family with name [%s]", name)
}

// TODO: metrics to test
//  - liveness (pods and deployments)
//  - readiness (pods and deployments)
//  - observability (pods and deployments)
//  - limits (pods and deployments)
