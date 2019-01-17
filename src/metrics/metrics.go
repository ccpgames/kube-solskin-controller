package metrics

import (
	"net/http"
	"os"
	"time"

	config "github.com/micro/go-config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// TODO: metrics to test
//  - liveness (pods and deployments)
//  - readiness (pods and deployments)
//  - observability (pods and deployments)
//  - limits (pods and deployments)

var metrics = map[string]*prometheus.GaugeVec{
	// Solskin metric for the observability of pods and deployments.
	"solskin_observability_resources": prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "solskin_observable_resources",
		Help: "...",
	}, []string{"name", "namespace", "resource_type"}),

	// Solskin metric for the liveness of pods and deployments.
	"solskin_liveness_resources": prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "solskin_liveness_resources",
		Help: "...",
	}, []string{"name", "namespace", "resource_type"}),

	// Solskin metric for the readiness of pods and deployments.
	"solskin_readiness_resources": prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "solskin_readiness_resources",
		Help: "...",
	}, []string{"name", "namespace", "resource_type"}),
}

// HealthHandler simply writes out an empty 200 status response.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Status", "200")
}

func init() {
	// Register all defined metrics.
	for _, metric := range metrics {
		prometheus.MustRegister(metric)
	}
}

func main() {
	cfg := config.NewConfig()

	http.HandleFunc("/health", HealthHandler)
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8080", nil)

	kcfg, _ := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	client, _ := kubernetes.NewForConfig(kcfg)

	startMetricUpdater(client, cfg)
	time.Sleep(10 * time.Second)
}
