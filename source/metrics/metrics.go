package metrics

import (
	"fmt"
	"net/http"
)

func getMetrics(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Welcome to my website!")
}

func metrics() {
	http.HandleFunc("/metrics", getMetrics)
	http.ListenAndServe(":80", nil)
}
