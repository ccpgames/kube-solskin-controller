package metrics

import (
	"fmt"
	"github.com/micro/go-config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"log"
	"net/http"
)

// Service TODO
type Service struct{}

// GetConfigurationSlug TODO
func (s Service) GetConfigurationSlug() string {
	return "metrics"
}

// GenerateEventHandlers TODO
func (s Service) GenerateEventHandlers() []cache.ResourceEventHandlerFuncs {
	return []cache.ResourceEventHandlerFuncs{}
}

// Start will initialize and run the metrics service.
func (s Service) Start(client kubernetes.Interface, cfg config.Config) {
	// Get port from configuration.
	portCfg := fmt.Sprintf("%s.port", s.GetConfigurationSlug())
	port := cfg.Get(portCfg).Int(8080)

	// Get endpoint from configuration.
	endpoint := fmt.Sprintf("%s.endpoint", s.GetConfigurationSlug())
	endpoint = cfg.Get(endpoint).String("metrics")

	// TODO: handle errors
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
	}
	http.Handle(fmt.Sprintf("/%s", endpoint), promhttp.Handler())
	log.Println("starting metric exporter server")
	go server.ListenAndServe()

	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()

	// // Stop the http server. TODO: handle errors
	// server.Shutdown(ctx)
}
