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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type MetricsTest struct {
	Expected float64
	Name     string
	Labels   map[string]string
}

// TestMetricLiveness tests that the metric service correctly reports liveness
// values for kubernetes resources.
func TestMetricLiveness(t *testing.T) {
	// Create a fake default configuration.
	cfg := config.NewConfig()

	// Create the fake client.
	client := fake.NewSimpleClientset()

	// Define our kubernetes resources.
	pods := []*corev1.Pod{
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "without-liveness",
				Namespace: "default",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-exec-liveness",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					corev1.Container{
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								Exec: &corev1.ExecAction{
									Command: []string{"uname"},
								},
							},
						},
					},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-http-liveness",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					corev1.Container{
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Port: intstr.FromInt(80),
								},
							},
						},
					},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-tcp-liveness",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					corev1.Container{
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								TCPSocket: &corev1.TCPSocketAction{
									Port: intstr.FromInt(80),
								},
							},
						},
					},
				},
			},
		},
	}
	deployments := []*appsv1.Deployment{
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "without-liveness",
				Namespace: "default",
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-exec-liveness",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: pods[1].Spec,
				},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-http-liveness",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: pods[2].Spec,
				},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-tcp-liveness",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: pods[3].Spec,
				},
			},
		},
	}
	daemonsets := []*appsv1.DaemonSet{
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "without-liveness",
				Namespace: "default",
			},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-exec-liveness",
				Namespace: "default",
			},
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: pods[1].Spec,
				},
			},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-http-liveness",
				Namespace: "default",
			},
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: pods[2].Spec,
				},
			},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-tcp-liveness",
				Namespace: "default",
			},
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: pods[3].Spec,
				},
			},
		},
	}

	// Setup resources in the cluster.
	setupKubernetesTestResources(t, client, pods, deployments, daemonsets)

	// Start the metric updater.
	startMetricUpdater(client, cfg)

	subtests := make([]MetricsTest, 0)
	for idx, pod := range pods {
		// Determine expected value (first one is false / 0.0).
		e := 0.0
		if idx > 0 {
			e = 1.0
		}

		// Create the test entry.
		test := MetricsTest{
			Expected: e,
			Name:     "solskin_liveness_resources",
			Labels: map[string]string{
				"name":          pod.GetName(),
				"namespace":     pod.GetNamespace(),
				"resource_type": "pod",
			},
		}
		subtests = append(subtests, test)
	}

	for idx, deployment := range deployments {
		// Determine expected value (first one is false / 0.0).
		e := 0.0
		if idx > 0 {
			e = 1.0
		}

		// Create the test entry.
		test := MetricsTest{
			Expected: e,
			Name:     "solskin_liveness_resources",
			Labels: map[string]string{
				"name":          deployment.GetName(),
				"namespace":     deployment.GetNamespace(),
				"resource_type": "deployment",
			},
		}
		subtests = append(subtests, test)
	}

	for idx, daemonset := range daemonsets {
		// Determine expected value (first one is false / 0.0).
		e := 0.0
		if idx > 0 {
			e = 1.0
		}

		// Create the test entry.
		test := MetricsTest{
			Expected: e,
			Name:     "solskin_liveness_resources",
			Labels: map[string]string{
				"name":          daemonset.GetName(),
				"namespace":     daemonset.GetNamespace(),
				"resource_type": "daemonset",
			},
		}
		subtests = append(subtests, test)
	}

	for _, subtest := range subtests {
		checkMetrics(t, subtest)
	}
}

// TestMetricReadiness tests that the metric service correctly reports readiness
// values for kubernetes resources.
func TestMetricReadiness(t *testing.T) {
	// Create a fake default configuration.
	cfg := config.NewConfig()

	// Create the fake client.
	client := fake.NewSimpleClientset()

	// Define our kubernetes resources.
	pods := []*corev1.Pod{
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "without-readiness",
				Namespace: "default",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-exec-readiness",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					corev1.Container{
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								Exec: &corev1.ExecAction{
									Command: []string{"uname"},
								},
							},
						},
					},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-http-readiness",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					corev1.Container{
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Port: intstr.FromInt(80),
								},
							},
						},
					},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-tcp-readiness",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					corev1.Container{
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								TCPSocket: &corev1.TCPSocketAction{
									Port: intstr.FromInt(80),
								},
							},
						},
					},
				},
			},
		},
	}
	deployments := []*appsv1.Deployment{
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "without-readiness",
				Namespace: "default",
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-exec-readiness",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: pods[1].Spec,
				},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-http-readiness",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: pods[2].Spec,
				},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-tcp-readiness",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: pods[3].Spec,
				},
			},
		},
	}
	daemonsets := []*appsv1.DaemonSet{
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "without-readiness",
				Namespace: "default",
			},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-exec-readiness",
				Namespace: "default",
			},
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: pods[1].Spec,
				},
			},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-http-readiness",
				Namespace: "default",
			},
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: pods[2].Spec,
				},
			},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-tcp-readiness",
				Namespace: "default",
			},
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: pods[3].Spec,
				},
			},
		},
	}

	// Setup resources in the cluster.
	setupKubernetesTestResources(t, client, pods, deployments, daemonsets)

	// Start the metric updater.
	startMetricUpdater(client, cfg)

	subtests := make([]MetricsTest, 0)
	for idx, pod := range pods {
		// Determine expected value (first one is false / 0.0).
		e := 0.0
		if idx > 0 {
			e = 1.0
		}

		// Create the test entry.
		test := MetricsTest{
			Expected: e,
			Name:     "solskin_readiness_resources",
			Labels: map[string]string{
				"name":          pod.GetName(),
				"namespace":     pod.GetNamespace(),
				"resource_type": "pod",
			},
		}
		subtests = append(subtests, test)
	}

	for idx, deployment := range deployments {
		// Determine expected value (first one is false / 0.0).
		e := 0.0
		if idx > 0 {
			e = 1.0
		}

		// Create the test entry.
		test := MetricsTest{
			Expected: e,
			Name:     "solskin_readiness_resources",
			Labels: map[string]string{
				"name":          deployment.GetName(),
				"namespace":     deployment.GetNamespace(),
				"resource_type": "deployment",
			},
		}
		subtests = append(subtests, test)
	}

	for idx, daemonset := range daemonsets {
		// Determine expected value (first one is false / 0.0).
		e := 0.0
		if idx > 0 {
			e = 1.0
		}

		// Create the test entry.
		test := MetricsTest{
			Expected: e,
			Name:     "solskin_readiness_resources",
			Labels: map[string]string{
				"name":          daemonset.GetName(),
				"namespace":     daemonset.GetNamespace(),
				"resource_type": "daemonset",
			},
		}
		subtests = append(subtests, test)
	}

	for _, subtest := range subtests {
		checkMetrics(t, subtest)
	}
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
					Spec: pods[0].Spec,
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
					Spec: pods[1].Spec,
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
					Spec: pods[2].Spec,
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
					Spec: pods[0].Spec,
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
					Spec: pods[1].Spec,
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
					Spec: pods[2].Spec,
				},
			},
		},
	}

	// Setup resources in the cluster.
	setupKubernetesTestResources(t, client, pods, deployments, daemonsets)

	// Start the metric updater.
	startMetricUpdater(client, cfg)

	subtests := make([]MetricsTest, 0)
	for idx, pod := range pods {
		// Determine expected value (first one is false / 0.0).
		e := 0.0
		if idx > 0 {
			e = 1.0
		}

		// Create the test entry.
		test := MetricsTest{
			Expected: e,
			Name:     "solskin_observability_resources",
			Labels: map[string]string{
				"name":          pod.GetName(),
				"namespace":     pod.GetNamespace(),
				"resource_type": "pod",
			},
		}
		subtests = append(subtests, test)
	}
	for idx, deployment := range deployments {
		// Determine expected value (first one is false / 0.0).
		e := 0.0
		if idx > 0 {
			e = 1.0
		}

		// Create the test entry.
		test := MetricsTest{
			Expected: e,
			Name:     "solskin_observability_resources",
			Labels: map[string]string{
				"name":          deployment.GetName(),
				"namespace":     deployment.GetNamespace(),
				"resource_type": "deployment",
			},
		}
		subtests = append(subtests, test)
	}
	for idx, daemonset := range daemonsets {
		// Determine expected value (first one is false / 0.0).
		e := 0.0
		if idx > 0 {
			e = 1.0
		}

		// Create the test entry.
		test := MetricsTest{
			Expected: e,
			Name:     "solskin_observability_resources",
			Labels: map[string]string{
				"name":          daemonset.GetName(),
				"namespace":     daemonset.GetNamespace(),
				"resource_type": "daemonset",
			},
		}
		subtests = append(subtests, test)
	}

	for _, subtest := range subtests {
		checkMetrics(t, subtest)
	}
}

// TestMetricLimits tests that the metric service correctly reports limits
// values for kubernetes resources.
func TestMetricLimits(t *testing.T) {
	// Create a fake default configuration.
	cfg := config.NewConfig()

	// Create the fake client.
	client := fake.NewSimpleClientset()

	// Define our kubernetes resources.
	pods := []*corev1.Pod{
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "without-limits",
				Namespace: "default",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-some-limits",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					corev1.Container{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:              *resource.NewScaledQuantity(1, resource.Mega),
								corev1.ResourceEphemeralStorage: *resource.NewScaledQuantity(1, resource.Mega),
							},
						},
					},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-all-limits",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					corev1.Container{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:              *resource.NewScaledQuantity(1, resource.Mega),
								corev1.ResourceMemory:           *resource.NewScaledQuantity(1, resource.Mega),
								corev1.ResourceStorage:          *resource.NewScaledQuantity(1, resource.Mega),
								corev1.ResourceEphemeralStorage: *resource.NewScaledQuantity(1, resource.Mega),
							},
						},
					},
				},
			},
		},
	}
	deployments := []*appsv1.Deployment{
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "without-limits",
				Namespace: "default",
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-some-limits",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: pods[1].Spec,
				},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-all-limits",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: pods[2].Spec,
				},
			},
		},
	}
	daemonsets := []*appsv1.DaemonSet{
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "without-limits",
				Namespace: "default",
			},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-some-limits",
				Namespace: "default",
			},
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: pods[1].Spec,
				},
			},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-all-limits",
				Namespace: "default",
			},
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: pods[2].Spec,
				},
			},
		},
	}

	// Setup resources in the cluster.
	setupKubernetesTestResources(t, client, pods, deployments, daemonsets)

	// Start the metric updater.
	startMetricUpdater(client, cfg)

	subtests := make([]MetricsTest, 0)
	for idx, pod := range pods {
		// Determine expected value (first one is false / 0.0).
		e := 2.0 * float64(idx)

		// Create the test entry.
		test := MetricsTest{
			Expected: e,
			Name:     "solskin_limits_resources",
			Labels: map[string]string{
				"name":          pod.GetName(),
				"namespace":     pod.GetNamespace(),
				"resource_type": "pod",
			},
		}
		subtests = append(subtests, test)
	}

	for idx, deployment := range deployments {
		// Determine expected value (first one is false / 0.0).
		e := 2.0 * float64(idx)

		// Create the test entry.
		test := MetricsTest{
			Expected: e,
			Name:     "solskin_limits_resources",
			Labels: map[string]string{
				"name":          deployment.GetName(),
				"namespace":     deployment.GetNamespace(),
				"resource_type": "deployment",
			},
		}
		subtests = append(subtests, test)
	}

	for idx, daemonset := range daemonsets {
		// Determine expected value (first one is false / 0.0).
		e := 2.0 * float64(idx)

		// Create the test entry.
		test := MetricsTest{
			Expected: e,
			Name:     "solskin_limits_resources",
			Labels: map[string]string{
				"name":          daemonset.GetName(),
				"namespace":     daemonset.GetNamespace(),
				"resource_type": "daemonset",
			},
		}
		subtests = append(subtests, test)
	}

	for _, subtest := range subtests {
		checkMetrics(t, subtest)
	}
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

	test := MetricsTest{
		Expected: 0.0,
		Name:     "solskin_observability_resources",
		Labels: map[string]string{
			"name":          "test",
			"namespace":     "default",
			"resource_type": "pod",
		},
	}
	checkMetrics(t, test)
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
func checkMetrics(t *testing.T, test MetricsTest) {
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
		if family.GetName() != test.Name {
			continue
		}

		// Check each individual set of label pairings.
		metric := family.GetMetric()
		for _, m := range metric {
			labelPairs := m.GetLabel()
			labels := make(map[string]string, len(labelPairs))
			for _, pair := range labelPairs {
				labels[pair.GetName()] = pair.GetValue()
			}

			eq := reflect.DeepEqual(test.Labels, labels)
			// If the labels aren't equal, continue to next submetric.
			if !eq {
				continue
			}

			// Otherwise we found the exact metric we're looking for.
			// Time to compare the value, assumes metric is a gauge type.
			v := m.GetGauge().GetValue()
			if m.GetGauge().GetValue() != test.Expected {
				t.Errorf("value did not match, expected [%f], actual [%f]", test.Expected, v)
			}
			return
		}
		t.Error("found metric family, but no label match")
		return
	}
	t.Errorf("could not find metric family with name [%s]", test.Name)
}

// TODO: metrics to test
//  - liveness (pods and deployments)
//  - readiness (pods and deployments)
//  - observability (pods and deployments)
//  - limits (pods and deployments)
