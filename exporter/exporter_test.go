package exporter

import (
	// "log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/expfmt"
	// "github.com/stretchr/testify/assert"

	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// MetricsTest ...
type MetricsTest struct {
	Expected float64
	Name     string
	Labels   map[string]string
}

func TestPodObservability(t *testing.T) {
	client := fake.NewSimpleClientset()

	// do whatever here with the fake client
	pods := []*core.Pod{
		&core.Pod{ObjectMeta: meta.ObjectMeta{
			Name:      "without-obs",
			Namespace: "default",
		}},
		&core.Pod{ObjectMeta: meta.ObjectMeta{
			Name:      "with-false-obs",
			Namespace: "default",
			Annotations: map[string]string{
				"prometheus.io/scrape": "false",
			},
		}},
		&core.Pod{ObjectMeta: meta.ObjectMeta{
			Name:      "with-true-obs",
			Namespace: "default",
			Annotations: map[string]string{
				"prometheus.io/scrape": "true",
			},
		}},
	}
	for _, object := range pods {
		client.Core().Pods(object.Namespace).Create(object)
	}

	// Start the exporter service.
	startExporter(client)

	// Define our expected metrics.
	tests := []MetricsTest{
		MetricsTest{
			Expected: 0,
			Name:     "solskin_observability_resources",
			Labels: map[string]string{
				"name":          "without-obs",
				"namespace":     "default",
				"resource_type": "pod",
			},
		},
		MetricsTest{
			Expected: 0,
			Name:     "solskin_observability_resources",
			Labels: map[string]string{
				"name":          "with-false-obs",
				"namespace":     "default",
				"resource_type": "pod",
			},
		},
		MetricsTest{
			Expected: 0,
			Name:     "solskin_observability_resources",
			Labels: map[string]string{
				"name":          "with-true-obs",
				"namespace":     "default",
				"resource_type": "pod",
			},
		},
	}

	// Check our expected metrics against the exporter.
	checkMetrics(t, tests)
}

// A helper function to start the prometheus service, send a request, and check
// the value of a specific metric.
func checkMetrics(t *testing.T, tests []MetricsTest) {
	// Wait for just a little bit to allow the informer to do their job.
	time.Sleep(100 * time.Millisecond)

	// Create a request to pass to our handler. We don't have any query
	// parameters for now, so we'll pass 'nil' as the third parameter.
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

	for _, test := range tests {
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
}
