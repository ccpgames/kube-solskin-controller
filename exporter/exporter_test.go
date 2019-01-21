package exporter

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"

	"k8s.io/client-go/kubernetes/fake"
)

// MetricsTest ...
type MetricsTest struct {
	Expected float64
	Name     string
	Labels   map[string]string
}

func TestBasicMetric(t *testing.T) {
	assert.Fail("not yet implemented")
}

func TestObservability(t *testing.T) {
	assert.Fail(t, "not yet implemented")
	// client := setupMetrics()

	// do whatever here with the fake client

	// check metrics here
}

// A helper function to create fake kubernetes client, start the metrics
// service, and return the client.
func setupMetrics() *kubernetes.Clientset {
	client := fake.NewSimpleClientset()
	go Start(client, nil)
	return client
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
