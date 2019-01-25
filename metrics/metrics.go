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

// Service is the base service for the metrics service.
type Service struct {
	Client        kubernetes.Interface
	Configuration config.Config
}

// GetSlug returns the slug used for the configuration section.
func (s Service) GetSlug() string {
	return "metrics"
}

// GenerateEventHandlers returns all event handlers used by this service.
func (s Service) GenerateEventHandlers() []cache.ResourceEventHandlerFuncs {
	return []cache.ResourceEventHandlerFuncs{}
}

// Init doesn't need to do anything for this service.
func (s Service) Init() {
	// do nothing
}

// Start will run the metrics http service.
func (s Service) Start() {
	// Retrieve the configuration slug for this service.
	cslug := s.GetSlug()

	// Get port from configuration.
	port := s.Configuration.Get(cslug, "port").Int(8080)

	// Get endpoint from configuration.
	endpoint := s.Configuration.Get(cslug, "endpoint").String("metrics")

	// Create our server.
	log.Printf("attempting to start server on :%d/%s", port, endpoint)
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
	}
	http.Handle(fmt.Sprintf("/%s", endpoint), promhttp.Handler())

	// Start the metrics server.
	log.Println("starting metric exporter server")
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %s", err)
		}
	}()
}
