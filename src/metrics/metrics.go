package metrics

import (
	"net/http"
	"time"

	config "github.com/micro/go-config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// TODO: metrics to test
//  - liveness (pods, deployments, daemonsets)
//  - readiness (pods, deployments, daemonsets)
//  - observability (pods, deployments, daemonsets)
//  - limits (pods, deployments, daemonsets)

var metrics = map[string]*prometheus.GaugeVec{
	// Solskin metric for the observability of kubernetes resources.
	"solskin_observability_resources": prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "solskin_observable_resources",
		Help: "...",
	}, []string{"name", "namespace", "resource_type"}),

	// Solskin metric for the liveness of kubernetes resources.
	"solskin_liveness_resources": prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "solskin_liveness_resources",
		Help: "...",
	}, []string{"name", "namespace", "resource_type"}),

	// Solskin metric for the readiness of kubernetes resources.
	"solskin_readiness_resources": prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "solskin_readiness_resources",
		Help: "...",
	}, []string{"name", "namespace", "resource_type"}),

	// Solskin metric for the limits of kubernetes resources.
	"solskin_limits_resources": prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "solskin_limits_resources",
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

	kubeconfig := cfg.Get("kubeconfig").String("~/.kube/config")
	kcfg, _ := clientcmd.BuildConfigFromFlags("", kubeconfig)
	client, _ := kubernetes.NewForConfig(kcfg)

	startMetricUpdater(client, cfg)
	time.Sleep(10 * time.Second)
}
