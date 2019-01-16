package main

import (
	"flag"
	"os"

	"github.com/micro/go-config"
	"github.com/micro/go-config/source/env"
	sflag "github.com/micro/go-config/source/flag"

	// "k8s.io/apimachinery/pkg/apis/meta/v1"
	// "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	// "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func getConfiguration() config.Config {
	cfg := config.NewConfig()

	flag.String("exclude", "^kube-", "regex of namespaces to exclude")

	// Load our configuration, with environment stomping over flags.
	flagSource := sflag.NewSource(sflag.IncludeUnset(true))
	envSource := env.NewSource(env.WithStrippedPrefix("SOLSKIN"))
	config.Load(flagSource, envSource)

	return cfg
}

func createKubernetesClientset() *kubernetes.Clientset {
	// Configure our connection to the kube's API.
	kcfg, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(kcfg)
	if err != nil {
		panic(err.Error())
	}

	return clientset
}
