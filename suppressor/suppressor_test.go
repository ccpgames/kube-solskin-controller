package suppressor

import (
	config "github.com/micro/go-config"
	"github.com/stretchr/testify/assert"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

type ResourceTest struct {
	Expected bool
	Resource interface{}
}

func TestSuppressionDecision(t *testing.T) {
	tests := []ResourceTest{
		// Pod without any standards.
		ResourceTest{
			Expected: true,
			Resource: &core.Pod{},
		},

		// Deployment without any standards.
		ResourceTest{
			Expected: true,
			Resource: &apps.Deployment{},
		},

		// DaemonSet without any standards.
		ResourceTest{
			Expected: true,
			Resource: &apps.DaemonSet{},
		},

		// Pod with all proper standards.
		ResourceTest{
			Expected: false,
			Resource: &core.Pod{
				ObjectMeta: meta.ObjectMeta{
					Annotations: map[string]string{
						"prometheus.io/scrape": "false",
					},
				},
				Spec: core.PodSpec{
					Containers: []core.Container{
						core.Container{
							LivenessProbe: &core.Probe{
								Handler: core.Handler{
									Exec: &core.ExecAction{},
								},
							},
							ReadinessProbe: &core.Probe{
								Handler: core.Handler{
									Exec: &core.ExecAction{},
								},
							},
							Resources: core.ResourceRequirements{
								Requests: core.ResourceList{
									core.ResourceCPU:    *resource.NewScaledQuantity(1, resource.Mega),
									core.ResourceMemory: *resource.NewScaledQuantity(1, resource.Mega),
								},
								Limits: core.ResourceList{
									core.ResourceCPU:    *resource.NewScaledQuantity(1, resource.Mega),
									core.ResourceMemory: *resource.NewScaledQuantity(1, resource.Mega),
								},
							},
						},
					},
				},
			},
		},

		// Deployment with all proper standards.
		ResourceTest{
			Expected: false,
			Resource: &apps.Deployment{
				Spec: apps.DeploymentSpec{
					Template: core.PodTemplateSpec{
						ObjectMeta: meta.ObjectMeta{
							Annotations: map[string]string{
								"prometheus.io/scrape": "false",
							},
						},
						Spec: core.PodSpec{
							Containers: []core.Container{
								core.Container{
									LivenessProbe: &core.Probe{
										Handler: core.Handler{
											Exec: &core.ExecAction{},
										},
									},
									ReadinessProbe: &core.Probe{
										Handler: core.Handler{
											Exec: &core.ExecAction{},
										},
									},
									Resources: core.ResourceRequirements{
										Requests: core.ResourceList{
											core.ResourceCPU:    *resource.NewScaledQuantity(1, resource.Mega),
											core.ResourceMemory: *resource.NewScaledQuantity(1, resource.Mega),
										},
										Limits: core.ResourceList{
											core.ResourceCPU:    *resource.NewScaledQuantity(1, resource.Mega),
											core.ResourceMemory: *resource.NewScaledQuantity(1, resource.Mega),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	s := Service{
		Configuration: config.NewConfig(),
	}
	for _, test := range tests {
		actual := s.toSuppress(test.Resource)
		assert.Exactly(t, test.Expected, actual)
	}
}
