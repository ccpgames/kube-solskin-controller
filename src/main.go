package main

import (
	"flag"
	"os"

	"github.com/micro/go-config"
	"github.com/micro/go-config/source/env"
	sflag "github.com/micro/go-config/source/flag"

	// "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	// "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	// appsv1 "k8s.io/api/apps/v1"
)

func main() {
	startMetrics()
}

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

// func startKeeper() {
// 	cfg := getConfiguration()
// 	client := createKubernetesClientset()

// 	// Create our informer.
// 	factory := informers.NewSharedInformerFactory(client, 0)
// 	informer := factory.Apps().V1().Deployments().Informer()
// 	stopper := make(chan struct{})
// 	defer close(stopper)

// 	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
// 		AddFunc: func(obj interface{}) {
// 			deployment := obj.(*appsv1.Deployment)
// 			keeper.OnDeploymentTrigger(cfg, client, deployment)
// 		},
// 		UpdateFunc: func(old interface{}, obj interface{}) {
// 			deployment := obj.(*appsv1.Deployment)
// 			keeper.OnDeploymentTrigger(cfg, client, deployment)
// 		},
// 	})

// 	// Run the informer.
// 	informer.Run(stopper)
// }
