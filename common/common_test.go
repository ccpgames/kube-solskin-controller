package common

import (
	"github.com/stretchr/testify/assert"
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
