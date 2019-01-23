package suppressor

import (
	config "github.com/micro/go-config"
	"github.com/stretchr/testify/assert"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

type ResourceTest struct {
	Expected bool
	Resource interface{}
}

func TestEligibility(t *testing.T) {
	// Define our tests.
	tests := []ResourceTest{
		ResourceTest{
			Expected: true,
			Resource: core.Pod{
				ObjectMeta: meta.ObjectMeta{
					Namespace: "default",
				},
			},
		},
		ResourceTest{
			Expected: false,
			Resource: core.Pod{
				ObjectMeta: meta.ObjectMeta{
					Namespace: "kube-system",
				},
			},
		},
	}

	s := Service{
		Configuration: config.NewConfig(),
	}
	for _, test := range tests {
		actual := s.isEligible(test.Resource)
		assert.Exactly(t, actual, test.Expected)
	}
}

func TestSuppressionDecision(t *testing.T) {
	tests := []ResourceTest{
		ResourceTest{
			Expected: true,
			Resource: &core.Pod{
				ObjectMeta: meta.ObjectMeta{},
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
