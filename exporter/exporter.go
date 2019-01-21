package exporter

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	// apps "k8s.io/api/apps/v1"
	// core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var categories = []string{
	"observability",
	"liveness",
	"readiness",
	"limits",
}
var metrics = make(map[string]*prometheus.GaugeVec, len(categories))

// Start will initialize and run the metrics service.
func Start(client kubernetes.Interface, stopper <-chan os.Signal) {
	startExporter(client)

	// TODO: handle errors
	server := &http.Server{
		Addr: ":8080",
	}
	http.Handle("/metrics", promhttp.Handler())
	log.Println("starting metric exporter server")
	go server.ListenAndServe()

	<-stopper
	log.Println("received stopper signal")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Stop the http server. TODO: handle errors
	server.Shutdown(ctx)
}

func startExporter(client kubernetes.Interface) {
	// Initialize our metrics.
	for _, category := range categories {
		metrics[category] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: fmt.Sprintf("solskin_%s_resources", category),
			Help: fmt.Sprintf("proof of %s", category),
		}, []string{"name", "namespace", "resource_type"})
		prometheus.MustRegister(metrics[category])
	}

	factory := informers.NewSharedInformerFactory(client, 0)
	informers := []cache.SharedIndexInformer{
		factory.Apps().V1().DaemonSets().Informer(),
		factory.Apps().V1().Deployments().Informer(),
		factory.Apps().V1().ReplicaSets().Informer(),
		factory.Apps().V1().StatefulSets().Informer(),
		factory.Batch().V1().Jobs().Informer(),
		factory.Core().V1().Pods().Informer(),
	}

	s := make(chan struct{})
	for _, informer := range informers {
		informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    func(obj interface{}) { onObjectChange(obj) },
			UpdateFunc: func(_, obj interface{}) { onObjectChange(obj) },
			DeleteFunc: func(obj interface{}) { onObjectDelete(obj) },
		})

		go informer.Run(s)
	}

	// Wait until our informer has synced.
	log.Println("waiting for informer to sync")
	for _, informer := range informers {
		for !informer.HasSynced() {
			time.Sleep(10 * time.Millisecond)
		}
	}
	log.Println("informers have synced")
}

func getObjectMeta(obj interface{}) (meta.ObjectMeta, string) {
	// Use reflection to determine resource type.
	// I don't like this but I can't find a better way of doing it at the moment.
	v := reflect.Indirect(reflect.ValueOf(obj))
	objectMeta := v.FieldByName("ObjectMeta").Interface().(meta.ObjectMeta)
	return objectMeta, strings.ToLower(v.Type().Name())
}

func onObjectChange(obj interface{}) {
	objectMeta, ktype := getObjectMeta(obj)
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
	observable := b2f64(hasAnnotation(objectMeta, "prometheus.io/scrape"))
	gauge.Set(observable)
}

func onObjectDelete(obj interface{}) {
	objectMeta, ktype := getObjectMeta(obj)
	labels := map[string]string{
		"name":          objectMeta.GetName(),
		"namespace":     objectMeta.GetNamespace(),
		"resource_type": ktype,
	}

	for _, metric := range metrics {
		metric.Delete(labels)
	}
}

func hasAnnotation(objectMeta meta.ObjectMeta, annotation string) bool {
	annotations := objectMeta.GetAnnotations()
	for key := range annotations {
		if key == annotation {
			return true
		}
	}

	return false
}

func b2f64(value bool) float64 {
	if value {
		return 1.0
	}
	return 0.0
}
