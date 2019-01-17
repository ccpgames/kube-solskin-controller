package keeper

import (
	"fmt"
	"testing"

	config "github.com/micro/go-config"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake "k8s.io/client-go/kubernetes/fake"
)

func TestManagementEligiblity(t *testing.T) {
	var subtests = []struct {
		namespace string
		name      string
		expected  bool
	}{
		{"default", "demo", true},
		{"default", "other-demo", true},
		{"kube-system", "external-dns", false},
		{"kube-monitor", "prometheus", false},
		{"kube-monitor", "grafana", false},
		{"kube-ingress", "aws-alb-controller", false},
	}

	cfg := config.NewConfig()

	for _, subtest := range subtests {
		testname := fmt.Sprintf("%s.%s", subtest.name, subtest.namespace)
		t.Run(testname, func(t *testing.T) {
			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: subtest.namespace,
					Name:      subtest.name,
				},
			}

			result := isEligibleForManagement(cfg, deployment)
			if result != subtest.expected {
				t.Error("failed eligibility unit test")
			}
		})
	}
}

func TestSuppressionDecision(t *testing.T) {
	var subtests = []struct {
		name        string
		expected    bool
		annotations map[string]string
	}{
		// Deployment with Acknowledged Positive Metrics Scrape
		{"ack-pos", false, map[string]string{"prometheus.io/scrape": "true"}},

		// Deployment with Acknowledged Negative Metrics Scrape
		{"ack-neg", false, map[string]string{"prometheus.io/scrape": "false"}},

		// Deployment with Unacknowledged Metrics Scrape
		{"unack", true, map[string]string{}},
	}

	cfg := config.NewConfig()

	for _, subtest := range subtests {
		t.Run(subtest.name, func(t *testing.T) {
			deployment := &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: subtest.annotations,
						},
					},
				},
			}

			result := determineSuppressionDecision(cfg, deployment)
			if result != subtest.expected {
				t.Error("failed management decision unit test")
			}
		})
	}
}

func TestSuppression(t *testing.T) {
	var subtests = []struct {
		name      string
		namespace string
		paused    bool
	}{
		{"not-paused", "default", false},
		{"already-paused", "default", true},
	}

	client := fake.NewSimpleClientset()

	for _, subtest := range subtests {
		t.Run(subtest.name, func(t *testing.T) {
			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: subtest.namespace,
					Name:      subtest.name,
				},
				Spec: appsv1.DeploymentSpec{
					Paused: subtest.paused,
				},
			}

			dplController := client.Apps().Deployments(subtest.namespace)
			dplController.Create(deployment)

			performSuppression(client, deployment)

			dpl, err := dplController.Get(subtest.name, metav1.GetOptions{})
			if err != nil {
				t.Error(err)
			}

			// Check the paused status.
			if !dpl.Spec.Paused {
				t.Error("failure to paused deployment")
			}
		})
	}
}
