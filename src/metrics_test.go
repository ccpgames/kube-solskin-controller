package main

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"net/http/httptest"

	"testing"
)

func TestMain(t *testing.T) {
	main()
}

func TestMetricLiveness(t *testing.T) {
	t.Error("not yet implemented")
}

func TestMetricObservability(t *testing.T) {
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
}

func TestMetricLimits(t *testing.T) {
	t.Error("not yet implemented")
}

// TODO: metrics to test
//  - liveness (pods and deployments)
//  - readiness (pods and deployments)
//  - observability (pods and deployments)
//  - limits (pods and deployments)
