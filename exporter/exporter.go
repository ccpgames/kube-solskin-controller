package exporter

import (
	"fmt"
	"github.com/micro/go-config"
	"log"

	"github.com/prometheus/client_golang/prometheus"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/celestialorb/solskin/common"
)

var categories = []string{
	"observability",
	"liveness",
	"readiness",
	"limits",
}
var promMetrics = make(map[string]*prometheus.GaugeVec, len(categories))

// Service TODO
type Service struct{}

// GetConfigurationSlug TODO
func (s Service) GetConfigurationSlug() string {
	return "exporter"
}

// GenerateEventHandlers TODO
func (s Service) GenerateEventHandlers() []cache.ResourceEventHandlerFuncs {
	return []cache.ResourceEventHandlerFuncs{
		cache.ResourceEventHandlerFuncs{
			AddFunc:    func(obj interface{}) { s.onObjectChange(obj) },
			UpdateFunc: func(_, obj interface{}) { s.onObjectChange(obj) },
			DeleteFunc: func(obj interface{}) { s.onObjectDelete(obj) },
		},
	}
}

// Start will initialize and run the metrics service.
func (s Service) Start(client kubernetes.Interface, cfg config.Config) {
	// Initialize our metrics.
	for _, category := range categories {
		promMetrics[category] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: fmt.Sprintf("solskin_%s_resources", category),
			Help: fmt.Sprintf("proof of %s", category),
		}, []string{"name", "namespace", "resource_type"})
		prometheus.MustRegister(promMetrics[category])
	}
}

// Called when one of the informers detects either a new or updated kubernetes
// resource, with the object as the input parameter.
func (s Service) onObjectChange(obj interface{}) {
	objectMeta, ktype := common.GetObjectMeta(obj)
	labels := map[string]string{
		"name":          objectMeta.GetName(),
		"namespace":     objectMeta.GetNamespace(),
		"resource_type": ktype,
	}

	// Pull out the podspec from the type of object.
	spec := common.GetPodSpec(obj)

	for _, category := range categories {
		// Create or retrieve our metric.
		gauge, err := promMetrics[category].GetMetricWith(labels)
		if err != nil {
			log.Fatal(err)
		}

		// Set our metric.
		value := false
		switch category {
		case "observability":
			value = common.HasObservability(objectMeta)
		case "liveness":
			value = common.HasLiveness(spec)
		case "readiness":
			value = common.HasReadiness(spec)
		case "limits":
			value = common.HasLimits(spec)
		}
		gauge.Set(common.BooleanToFloat64(value))
	}
}

// Called when one of the informers detects a deleted kubernetes resource,
// with the object as the input parameter.
func (s Service) onObjectDelete(obj interface{}) {
	objectMeta, ktype := common.GetObjectMeta(obj)
	labels := map[string]string{
		"name":          objectMeta.GetName(),
		"namespace":     objectMeta.GetNamespace(),
		"resource_type": ktype,
	}

	for _, metric := range promMetrics {
		metric.Delete(labels)
	}
}
