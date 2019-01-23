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
var metrics = make(map[string]*prometheus.GaugeVec, len(categories))

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
		metrics[category] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: fmt.Sprintf("solskin_%s_resources", category),
			Help: fmt.Sprintf("proof of %s", category),
		}, []string{"name", "namespace", "resource_type"})
		prometheus.MustRegister(metrics[category])
	}
}

// Called when one of the informers detects either a new or updated kubernetes
// resource, with the object as the input parameter.
func (s Service) onObjectChange(obj interface{}) {
	log.Println("EXPORTER [onObjectChange]")

	objectMeta, ktype := common.GetObjectMeta(obj)
	labels := map[string]string{
		"name":          objectMeta.GetName(),
		"namespace":     objectMeta.GetNamespace(),
		"resource_type": ktype,
	}

	// Create or retrieve our metric.
	gauge, err := metrics["observability"].GetMetricWith(labels)
	if err != nil {
		log.Fatal(err)
	}

	// Set our metric.
	observable := common.HasObservability(objectMeta)
	gauge.Set(common.BooleanToFloat64(observable))
}

// Called when one of the informers detects a deleted kubernetes resource,
// with the object as the input parameter.
func (s Service) onObjectDelete(obj interface{}) {
	log.Println("EXPORTER [onObjectDelete]")

	objectMeta, ktype := common.GetObjectMeta(obj)
	labels := map[string]string{
		"name":          objectMeta.GetName(),
		"namespace":     objectMeta.GetNamespace(),
		"resource_type": ktype,
	}

	for _, metric := range metrics {
		metric.Delete(labels)
	}
}
