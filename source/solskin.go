package main

import (
	config "github.com/micro/go-config"
	"k8s.io/api/apps/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"log"
	"regexp"
)

func main() {
	cfg := getConfiguration()
	client := createKubernetesClientset()

	// Create our informer.
	factory := informers.NewSharedInformerFactory(client, 0)
	informer := factory.Apps().V1().Deployments().Informer()
	stopper := make(chan struct{})
	defer close(stopper)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			deployment := obj.(*v1.Deployment)
			onDeploymentTrigger(cfg, client, deployment)
		},
		UpdateFunc: func(old interface{}, obj interface{}) {
			deployment := obj.(*v1.Deployment)
			onDeploymentTrigger(cfg, client, deployment)
		},
	})

	// Run the informer.
	informer.Run(stopper)
}

// Determines whether or not a deployment is eligible for management.
func isEligibleForManagement(cfg config.Config, obj *v1.Deployment) bool {
	// Check the namespace of the deployment object.
	namespace := obj.GetNamespace()

	// Determine the namespace exclusion regular expression.
	pattern := cfg.Get("exclude").String("^kube-")

	// Perform a regular expression match against our pattern.
	m, err := regexp.MatchString(pattern, namespace)
	if err != nil {
		panic(err)
	}

	// If we matched the pattern, then this deployment is not eligible for
	// management.
	if m {
		log.Printf("deployment in excluded namespace, skipping {%s}", namespace)
		return false
	}

	// Otherwise, the deployment is eligible for management.
	return true
}

// Determines whether or not to suppress the deployment.
func determineSuppressionDecision(cfg config.Config, obj *v1.Deployment) bool {
	// Simply check for the existence of the prometheus.io/scrape annotation.
	// We're using that as a litmus test to ensure whether or not the owners of
	// the deployment have thought about metrics.
	annotations := obj.Spec.Template.ObjectMeta.Annotations
	for key := range annotations {
		// If the annotations exists, we don't need to suppress the deployment.
		if key == "prometheus.io/scrape" {
			return false
		}
	}

	// Otherwise we need to suppress it.
	return true
}

// Pauses a deployment to suppress it if it doesn't have metrics.
func performSuppression(client kubernetes.Interface, obj *v1.Deployment) {
	// Set the pause attribute to true for the deployment.
	obj.Spec.Paused = true

	// Take the modified deployment and update it in the kube.
	client.Apps().Deployments(obj.GetNamespace()).Update(obj)
}

func onDeploymentTrigger(cfg config.Config, client kubernetes.Interface, obj *v1.Deployment) {
	eligible := isEligibleForManagement(cfg, obj)
	if !eligible {
		return
	}

	shouldSuppress := determineSuppressionDecision(cfg, obj)
	if !shouldSuppress {
		return
	}

	performSuppression(client, obj)
}
