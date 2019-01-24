package main

import (
	"fmt"
	"github.com/kubernetes/client-go/informers"
	"github.com/micro/go-config/source/env"
	"k8s.io/client-go/tools/cache"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/celestialorb/solskin/exporter"
	"github.com/celestialorb/solskin/metrics"
	"github.com/celestialorb/solskin/suppressor"
	config "github.com/micro/go-config"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// SolskinService general service interface.
type SolskinService interface {
	GenerateEventHandlers() []cache.ResourceEventHandlerFuncs
	GetConfigurationSlug() string
	Init()
	Start()
}

func main() {
	// Load up our configuration from the environment.
	cfg := config.NewConfig()
	cfg.Load(env.NewSource(env.WithStrippedPrefix("SOLSKIN")))

	// Try to pull the in-cluster configuration first.
	log.Println("attempting to pull in-cluster kube configuration")
	kubecfg, err := rest.InClusterConfig()
	if err != nil {
		log.Println("service running outside of kube cluster")
		log.Println("attempting to pull kube cluster info from local filesystem")

		// If we're not in a cluster then pull configuration from local filesystem.
		kubeFile := fmt.Sprintf("%s/.kube/config", os.Getenv("HOME"))
		kubeconfig := cfg.Get("cluster", "kubecfg").String(kubeFile)

		kubecfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Println("kube configuration determined")

	client, err := kubernetes.NewForConfig(kubecfg)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("kube configuration is valid")

	stopper := make(chan os.Signal)

	signal.Notify(stopper, syscall.SIGTERM)
	signal.Notify(stopper, syscall.SIGINT)

	// Create our services.
	services := []SolskinService{
		exporter.Service{Client: client, Configuration: cfg},
		suppressor.Service{Client: client, Configuration: cfg},
		metrics.Service{Client: client, Configuration: cfg},
	}

	s, err := StartServices(services, client, cfg)
	if err != nil {
		log.Fatalf("error starting solskin services: %s", err)
	}
	defer close(s)

	// Wait for kill signal.
	<-stopper
}

// StartServices will initialize and kick off all given services with the
// proper set of informers.
func StartServices(
	services []SolskinService,
	client kubernetes.Interface,
	cfg config.Config,
) (chan struct{}, error) {
	// Initialize services here.
	for _, service := range services {
		service.Init()
	}

	// Create our informers
	factory := informers.NewSharedInformerFactory(client, 0)
	informers := []cache.SharedIndexInformer{
		factory.Apps().V1().DaemonSets().Informer(),
		factory.Apps().V1().Deployments().Informer(),
		factory.Apps().V1().ReplicaSets().Informer(),
		factory.Apps().V1().StatefulSets().Informer(),
		factory.Batch().V1().Jobs().Informer(),
		factory.Core().V1().Pods().Informer(),
	}

	handlers := make([]cache.ResourceEventHandlerFuncs, 0)
	for _, service := range services {
		handlers = append(handlers, service.GenerateEventHandlers()...)
	}

	for _, informer := range informers {
		for _, handler := range handlers {
			informer.AddEventHandler(handler)
		}
	}

	// Spool up services here.
	for _, service := range services {
		service.Start()
	}

	// Start our informers.
	s := make(chan struct{})
	for _, informer := range informers {
		go informer.Run(s)
	}

	// Wait until our informer has synced.
	log.Println("waiting for informers to sync")
	for _, informer := range informers {
		for !informer.HasSynced() {
			time.Sleep(10 * time.Millisecond)
		}
	}
	log.Println("informers have synced")

	return s, nil
}
