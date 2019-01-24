package suppressor

import (
	"fmt"
	"github.com/celestialorb/solskin/common"
	"github.com/micro/go-config"
	"github.com/prometheus/client_golang/prometheus"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"log"
	"strconv"
)

var suppressedResourcesMetric = prometheus.NewCounter(prometheus.CounterOpts{
	Help: "TODO",
	Name: "solskin_suppressed_resources",
})

// Service is the base service for the suppressor service.
type Service struct {
	Configuration config.Config
	Client        kubernetes.Interface
}

// GetConfigurationSlug returns the slug used for the configuration section.
func (s Service) GetConfigurationSlug() string {
	return "suppressor"
}

// GenerateEventHandlers returns all event handlers used by this service.
func (s Service) GenerateEventHandlers() []cache.ResourceEventHandlerFuncs {
	return []cache.ResourceEventHandlerFuncs{
		cache.ResourceEventHandlerFuncs{
			AddFunc:    func(obj interface{}) { s.onObjectChange(obj) },
			UpdateFunc: func(_, obj interface{}) { s.onObjectChange(obj) },
			DeleteFunc: func(obj interface{}) { s.onObjectDelete(obj) },
		},
	}
}

// Init registers prometheus metrics for the suppression service.
func (s Service) Init() {
	// Initialize the suppressor metrics.
	prometheus.MustRegister(suppressedResourcesMetric)
}

// Start will start any other components the service needs.
func (s Service) Start() {
	// do nothing
}

func (s Service) onSuppress(obj interface{}) {
	// Get the metadata of the resource.
	m, ktype := common.GetObjectMeta(obj)

	// If the resource is eligible then we have to suppress it, which will depend
	// on the type of the resource.
	switch ktype {
	case "pod":
		pod := obj.(*core.Pod)
		s.Client.Core().Pods(m.Namespace).Delete(pod.GetName(), &meta.DeleteOptions{})
		log.Printf("suppressing pod [%s.%s]", m.GetName(), m.GetNamespace())
	case "deployment":
		dpl := obj.(*apps.Deployment)

		if *dpl.Spec.Replicas <= 0 {
			return
		}

		// Store the desired replicas in an annotation.
		replicasStr := fmt.Sprintf("%d", *dpl.Spec.Replicas)
		dpl.ObjectMeta = common.SetServiceAnnotation(m, "replicas", replicasStr)

		replicas := int32(0)
		dpl.Spec.Replicas = &replicas
		s.Client.Apps().Deployments(m.Namespace).Update(dpl)
		log.Printf("suppressing deployment [%s.%s]", m.GetName(), m.GetNamespace())
	}
}

func (s Service) onUnsuppress(obj interface{}) {
	// Get the metadata of the resource.
	m, ktype := common.GetObjectMeta(obj)

	// If the resource is eligible then we have to suppress it, which will depend
	// on the type of the resource.
	switch ktype {
	case "deployment":
		dpl := obj.(*apps.Deployment)

		// Retrieve the desired replicas.
		replicasStr := common.GetServiceAnnotation(m, "replicas")
		replicasInt, err := strconv.ParseInt(replicasStr, 10, 32)
		if err != nil {
			log.Println("could not parse replica count")
			return
		}
		replicas := int32(replicasInt)

		// Check the replicas against the deployment spec, if they differ AND the
		// deployment spec is positive then the deployment spec wins.
		if (*dpl.Spec.Replicas > 0) && (replicas != *dpl.Spec.Replicas) {
			replicas = *dpl.Spec.Replicas
		}

		dpl.Spec.Replicas = &replicas
		s.Client.Apps().Deployments(m.Namespace).Update(dpl)
		log.Printf("suppressing deployment [%s.%s]", m.GetName(), m.GetNamespace())
	}
}

// Called when one of the informers detects either a new or updated kubernetes
// resource, with the object as the input parameter.
func (s Service) onObjectChange(obj interface{}) {
	// Determine if the resource is eligible for suppression, if not skip it.
	if !common.IsEligible(obj, s.Configuration) {
		return
	}

	// Suppress or unsuppress the resource.
	if s.toSuppress(obj) {
		s.onSuppress(obj)
	} else {
		s.onUnsuppress(obj)
	}

	// // Determine if the resource should be suppressed.
	// if !s.toSuppress(obj) {
	// 	return
	// }

	// // Get the metadata of the resource.
	// m, ktype := common.GetObjectMeta(obj)

	// // If the resource is eligible then we have to suppress it, which will depend
	// // on the type of the resource.
	// switch ktype {
	// case "pod":
	// 	pod := obj.(*core.Pod)
	// 	s.Client.Core().Pods(m.Namespace).Delete(pod.GetName(), &meta.DeleteOptions{})
	// 	log.Printf("suppressing pod [%s.%s]", m.GetName(), m.GetNamespace())
	// case "deployment":
	// 	dpl := obj.(*apps.Deployment)

	// 	// Store the desired replicas in an annotation.
	// 	dpl.ObjectMeta.Annotations["celestialorb/solskin.replicas"] = fmt.Sprintf("%d", dpl.Spec.Replicas)

	// 	replicas := int32(0)

	// 	dpl.Spec.Paused = true
	// 	dpl.Spec.Replicas = &replicas
	// 	s.Client.Apps().Deployments(m.Namespace).Update(dpl)
	// 	log.Printf("suppressing deployment [%s.%s]", m.GetName(), m.GetNamespace())
	// }
}

// Called when one of the informers detects either a new or updated kubernetes
// resource, with the object as the input parameter.
func (s Service) onObjectDelete(obj interface{}) {
	// Determine if the resource is eligible for suppression, if not skip it.
	if !common.IsEligible(obj, s.Configuration) {
		return
	}

	// Determine if the resource should be unsuppressed.
	if s.toSuppress(obj) {
		return
	}

	// Get the metadata of the resource.
	m, ktype := common.GetObjectMeta(obj)

	// If the resource is eligible then we have to suppress it, which will depend
	// on the type of the resource.
	switch ktype {
	case "deployment":
		dpl := obj.(*apps.Deployment)
		dpl.Spec.Paused = false
		s.Client.Apps().Deployments(m.Namespace).Update(dpl)
	}
}

// Helper function to determine if the resource should be suppressed.
func (s Service) toSuppress(obj interface{}) bool {
	// Determine the suppression decision.
	suppression := false
	_, ktype := common.GetObjectMeta(obj)
	switch ktype {
	case "pod":
		suppression = s.toSuppressPod(obj.(*core.Pod))
	case "deployment":
		suppression = s.toSuppressDeployment(obj.(*apps.Deployment))
	case "daemonset":
		suppression = s.toSuppressDaemonSet(obj.(*apps.DaemonSet))
	}
	return suppression
}

// Helper function to determine if the pod should be suppressed.
func (s Service) toSuppressPod(pod *core.Pod) bool {
	return toSuppress(pod.ObjectMeta, pod.Spec)
}

// Helper function to determine if the deployment should be suppressed.
func (s Service) toSuppressDeployment(dpl *apps.Deployment) bool {
	return toSuppress(dpl.Spec.Template.ObjectMeta, dpl.Spec.Template.Spec)
}

// Helper function to determine if the daemonset should be suppressed.
func (s Service) toSuppressDaemonSet(ds *apps.DaemonSet) bool {
	return toSuppress(ds.Spec.Template.ObjectMeta, ds.Spec.Template.Spec)
}

// Helper function to determine if a resource meets suppression.
func toSuppress(m meta.ObjectMeta, spec core.PodSpec) bool {
	checks := []bool{
		common.HasObservability(m),
		common.HasLiveness(spec),
		common.HasReadiness(spec),
		common.HasLimits(spec),
	}

	return !common.PassesChecks(checks)
}
