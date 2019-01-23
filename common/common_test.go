package common

import (
	"github.com/stretchr/testify/assert"
	// core "k8s.io/api/core/v1"
	// meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

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
