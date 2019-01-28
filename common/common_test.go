package common

import (
	config "github.com/micro/go-config"
	"github.com/stretchr/testify/assert"
	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

type ObjectMetaTest struct {
	Expected   bool
	ObjectMeta meta.ObjectMeta
}

type SpecTest struct {
	Expected bool
	Spec     core.PodSpec
}

func TestEligibility(t *testing.T) {
	type Test struct {
		Expected      bool
		Resource      interface{}
		Configuration config.Config
	}

	// Define our tests.
	tests := []Test{
		Test{
			Expected: true,
			Resource: core.Pod{
				ObjectMeta: meta.ObjectMeta{
					Namespace: "default",
				},
			},
			Configuration: config.NewConfig(),
		},
		Test{
			Expected: false,
			Resource: core.Pod{
				ObjectMeta: meta.ObjectMeta{
					Namespace: "kube-system",
				},
			},
			Configuration: config.NewConfig(),
		},
	}

	for _, test := range tests {
		actual := IsEligible(test.Resource, test.Configuration)
		assert.Exactly(t, actual, test.Expected)
	}
}

func TestGetPodSpec(t *testing.T) {
	type Test struct {
		Expected *core.PodSpec
		Resource interface{}
	}

	tests := []Test{
		// Pod
		Test{
			Expected: &core.PodSpec{Hostname: "test"},
			Resource: &core.Pod{Spec: core.PodSpec{Hostname: "test"}},
		},

		// Deployment
		Test{
			Expected: &core.PodSpec{Hostname: "test"},
			Resource: &apps.Deployment{
				Spec: apps.DeploymentSpec{
					Template: core.PodTemplateSpec{
						Spec: core.PodSpec{Hostname: "test"},
					},
				},
			},
		},

		// Daemonset
		Test{
			Expected: &core.PodSpec{Hostname: "test"},
			Resource: &apps.DaemonSet{
				Spec: apps.DaemonSetSpec{
					Template: core.PodTemplateSpec{
						Spec: core.PodSpec{Hostname: "test"},
					},
				},
			},
		},

		// Statefulset
		Test{
			Expected: &core.PodSpec{Hostname: "test"},
			Resource: &apps.StatefulSet{
				Spec: apps.StatefulSetSpec{
					Template: core.PodTemplateSpec{
						Spec: core.PodSpec{Hostname: "test"},
					},
				},
			},
		},

		// Job
		Test{
			Expected: &core.PodSpec{Hostname: "test"},
			Resource: &batch.Job{
				Spec: batch.JobSpec{
					Template: core.PodTemplateSpec{
						Spec: core.PodSpec{Hostname: "test"},
					},
				},
			},
		},
	}

	for _, test := range tests {
		r := GetPodSpec(test.Resource)
		assert.Exactly(t, test.Expected, r)
	}
}

func TestPassesChecks(t *testing.T) {
	type Test struct {
		Expected bool
		Checks   []bool
	}

	tests := []Test{
		Test{
			Expected: true,
			Checks:   []bool{},
		},
		Test{
			Expected: false,
			Checks:   []bool{false},
		},
		Test{
			Expected: true,
			Checks:   []bool{true},
		},
		Test{
			Expected: true,
			Checks:   []bool{true, true},
		},
		Test{
			Expected: false,
			Checks:   []bool{true, false},
		},
	}

	for _, test := range tests {
		actual := PassesChecks(test.Checks)
		assert.Exactly(t, test.Expected, actual)
	}
}

func TestHasObservability(t *testing.T) {
	tests := []ObjectMetaTest{
		ObjectMetaTest{
			Expected:   false,
			ObjectMeta: meta.ObjectMeta{},
		},
		ObjectMetaTest{
			Expected: true,
			ObjectMeta: meta.ObjectMeta{
				Annotations: map[string]string{
					"prometheus.io/scrape": "false",
				},
			},
		},
		ObjectMetaTest{
			Expected: true,
			ObjectMeta: meta.ObjectMeta{
				Annotations: map[string]string{
					"prometheus.io/scrape": "true",
				},
			},
		},
	}

	for _, test := range tests {
		actual := HasObservability(test.ObjectMeta)
		assert.Exactly(t, test.Expected, actual)
	}
}

func TestHasLiveness(t *testing.T) {
	tests := []SpecTest{
		// Basic test with exec liveness probe.
		SpecTest{
			Expected: true,
			Spec: core.PodSpec{
				Containers: []core.Container{
					core.Container{
						LivenessProbe: &core.Probe{
							Handler: core.Handler{
								Exec: &core.ExecAction{},
							},
						},
					},
				},
			},
		},

		// Basic test with HTTP liveness probe.
		SpecTest{
			Expected: true,
			Spec: core.PodSpec{
				Containers: []core.Container{
					core.Container{
						LivenessProbe: &core.Probe{
							Handler: core.Handler{
								HTTPGet: &core.HTTPGetAction{},
							},
						},
					},
				},
			},
		},

		// Basic test with TCP liveness probe.
		SpecTest{
			Expected: true,
			Spec: core.PodSpec{
				Containers: []core.Container{
					core.Container{
						LivenessProbe: &core.Probe{
							Handler: core.Handler{
								TCPSocket: &core.TCPSocketAction{},
							},
						},
					},
				},
			},
		},

		// Basic test with no liveness probe.
		SpecTest{
			Expected: false,
			Spec: core.PodSpec{
				Containers: []core.Container{
					core.Container{
						LivenessProbe: &core.Probe{},
					},
				},
			},
		},

		// Test with mix of probes in containers.
		SpecTest{
			Expected: false,
			Spec: core.PodSpec{
				Containers: []core.Container{
					core.Container{
						LivenessProbe: &core.Probe{},
					},
					core.Container{
						LivenessProbe: &core.Probe{
							Handler: core.Handler{
								Exec: &core.ExecAction{},
							},
						},
					},
				},
			},
		},

		// Test with mix of probes in containers.
		SpecTest{
			Expected: true,
			Spec: core.PodSpec{
				Containers: []core.Container{
					core.Container{
						LivenessProbe: &core.Probe{
							Handler: core.Handler{
								Exec: &core.ExecAction{},
							},
						},
					},
					core.Container{
						LivenessProbe: &core.Probe{
							Handler: core.Handler{
								Exec: &core.ExecAction{},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		actual := HasLiveness(test.Spec)
		assert.Exactly(t, test.Expected, actual)
	}
}

func TestHasReadiness(t *testing.T) {
	tests := []SpecTest{
		// Basic test with exec liveness probe.
		SpecTest{
			Expected: true,
			Spec: core.PodSpec{
				Containers: []core.Container{
					core.Container{
						ReadinessProbe: &core.Probe{
							Handler: core.Handler{
								Exec: &core.ExecAction{},
							},
						},
					},
				},
			},
		},

		// Basic test with HTTP liveness probe.
		SpecTest{
			Expected: true,
			Spec: core.PodSpec{
				Containers: []core.Container{
					core.Container{
						ReadinessProbe: &core.Probe{
							Handler: core.Handler{
								HTTPGet: &core.HTTPGetAction{},
							},
						},
					},
				},
			},
		},

		// Basic test with TCP liveness probe.
		SpecTest{
			Expected: true,
			Spec: core.PodSpec{
				Containers: []core.Container{
					core.Container{
						ReadinessProbe: &core.Probe{
							Handler: core.Handler{
								TCPSocket: &core.TCPSocketAction{},
							},
						},
					},
				},
			},
		},

		// Basic test with no liveness probe.
		SpecTest{
			Expected: false,
			Spec: core.PodSpec{
				Containers: []core.Container{
					core.Container{
						ReadinessProbe: &core.Probe{},
					},
				},
			},
		},

		// Test with mix of probes in containers.
		SpecTest{
			Expected: false,
			Spec: core.PodSpec{
				Containers: []core.Container{
					core.Container{
						ReadinessProbe: &core.Probe{},
					},
					core.Container{
						ReadinessProbe: &core.Probe{
							Handler: core.Handler{
								Exec: &core.ExecAction{},
							},
						},
					},
				},
			},
		},

		// Test with mix of probes in containers.
		SpecTest{
			Expected: true,
			Spec: core.PodSpec{
				Containers: []core.Container{
					core.Container{
						ReadinessProbe: &core.Probe{
							Handler: core.Handler{
								Exec: &core.ExecAction{},
							},
						},
					},
					core.Container{
						ReadinessProbe: &core.Probe{
							Handler: core.Handler{
								Exec: &core.ExecAction{},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		actual := HasReadiness(test.Spec)
		assert.Exactly(t, test.Expected, actual)
	}
}

func TestHasLimits(t *testing.T) {
	tests := []SpecTest{
		// Basic test with no resource limits.
		SpecTest{
			Expected: false,
			Spec: core.PodSpec{
				Containers: []core.Container{
					core.Container{
						Resources: core.ResourceRequirements{
							Limits: core.ResourceList{},
						},
					},
				},
			},
		},

		// Basic test with only CPU resource limits.
		SpecTest{
			Expected: false,
			Spec: core.PodSpec{
				Containers: []core.Container{
					core.Container{
						Resources: core.ResourceRequirements{
							Limits: core.ResourceList{
								core.ResourceCPU: *resource.NewScaledQuantity(1, resource.Mega),
							},
						},
					},
				},
			},
		},

		// Basic test with only memory resource limits.
		SpecTest{
			Expected: false,
			Spec: core.PodSpec{
				Containers: []core.Container{
					core.Container{
						Resources: core.ResourceRequirements{
							Limits: core.ResourceList{
								core.ResourceMemory: *resource.NewScaledQuantity(1, resource.Mega),
							},
						},
					},
				},
			},
		},

		// Basic test with both resource limits.
		SpecTest{
			Expected: true,
			Spec: core.PodSpec{
				Containers: []core.Container{
					core.Container{
						Resources: core.ResourceRequirements{
							Limits: core.ResourceList{
								core.ResourceCPU:    *resource.NewScaledQuantity(1, resource.Mega),
								core.ResourceMemory: *resource.NewScaledQuantity(1, resource.Mega),
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		actual := HasLimits(test.Spec)
		assert.Exactly(t, test.Expected, actual)
	}
}

func TestHasRequests(t *testing.T) {
	tests := []SpecTest{
		// Basic test with no resource requests.
		SpecTest{
			Expected: false,
			Spec: core.PodSpec{
				Containers: []core.Container{
					core.Container{
						Resources: core.ResourceRequirements{
							Requests: core.ResourceList{},
						},
					},
				},
			},
		},

		// Basic test with only CPU resource requests.
		SpecTest{
			Expected: false,
			Spec: core.PodSpec{
				Containers: []core.Container{
					core.Container{
						Resources: core.ResourceRequirements{
							Requests: core.ResourceList{
								core.ResourceCPU: *resource.NewScaledQuantity(1, resource.Mega),
							},
						},
					},
				},
			},
		},

		// Basic test with only memory resource requests.
		SpecTest{
			Expected: false,
			Spec: core.PodSpec{
				Containers: []core.Container{
					core.Container{
						Resources: core.ResourceRequirements{
							Requests: core.ResourceList{
								core.ResourceMemory: *resource.NewScaledQuantity(1, resource.Mega),
							},
						},
					},
				},
			},
		},

		// Basic test with both resource requests.
		SpecTest{
			Expected: true,
			Spec: core.PodSpec{
				Containers: []core.Container{
					core.Container{
						Resources: core.ResourceRequirements{
							Requests: core.ResourceList{
								core.ResourceCPU:    *resource.NewScaledQuantity(1, resource.Mega),
								core.ResourceMemory: *resource.NewScaledQuantity(1, resource.Mega),
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		actual := HasRequests(test.Spec)
		assert.Exactly(t, test.Expected, actual)
	}
}
