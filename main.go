package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/celestialorb/solskin/exporter"
	"github.com/celestialorb/solskin/notifier"
	"github.com/celestialorb/solskin/suppressor"
	"github.com/micro/go-config"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	cfg := config.NewConfig()

	kubecfg := fmt.Sprintf("%s/.kube/config", os.Getenv("HOME"))

	kubeconfig := cfg.Get("kubeconfig").String(kubecfg)
	kcfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	client, err := kubernetes.NewForConfig(kcfg)
	if err != nil {
		log.Fatal(err)
	}

	stopper := make(chan os.Signal)

	signal.Notify(stopper, syscall.SIGTERM)
	signal.Notify(stopper, syscall.SIGINT)

	go exporter.Start(client, stopper)
	go notifier.Start(stopper)
	go suppressor.Start(stopper)

	<-stopper
}
