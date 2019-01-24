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

// Called when one of the informers detects either a new or updated kubernetes
// resource, with the object as the input parameter.
func (s Service) onObjectChange(obj interface{}) {
	// Determine if the resource is eligible for suppression, if not skip it.
	if !common.IsEligible(obj, s.Configuration) {
		return
	}

	// If we don't need to suppress to object, simply return.
	if !s.toSuppress(obj) {
		return
	}

	// Get the metadata of the resource.
	m, ktype := common.GetObjectMeta(obj)

	// If the resource is eligible then we have to suppress it, which will depend
	// on the type of the resource.
	switch ktype {
	case "pod":
		pod := obj.(*core.Pod)

		// To suppress a pod, we simply delete it.
		opts := &meta.DeleteOptions{}
		s.Client.Core().Pods(m.Namespace).Delete(pod.GetName(), opts)
		log.Printf("suppressing pod [%s.%s]", m.GetName(), m.GetNamespace())
	case "deployment":
		dpl := obj.(*apps.Deployment)

		// To suppress a deployment, we set the replicas to zero.
		// if *dpl.Spec.Replicas <= 0 {
		// 	return
		// }

		replicas := int32(0)
		dpl.Spec.Replicas = &replicas
		s.Client.Apps().Deployments(m.Namespace).Update(dpl)
		log.Printf("suppressing deployment [%s.%s]", m.GetName(), m.GetNamespace())
	}
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
	return toSuppress(pod, pod.ObjectMeta, pod.Spec)
}

// Helper function to determine if the deployment should be suppressed.
func (s Service) toSuppressDeployment(dpl *apps.Deployment) bool {
	return toSuppress(dpl, dpl.Spec.Template.ObjectMeta, dpl.Spec.Template.Spec)
}

// Helper function to determine if the daemonset should be suppressed.
func (s Service) toSuppressDaemonSet(ds *apps.DaemonSet) bool {
	return toSuppress(ds, ds.Spec.Template.ObjectMeta, ds.Spec.Template.Spec)
}

// Helper function to determine if a resource meets suppression.
func toSuppress(obj interface{}, m meta.ObjectMeta, spec core.PodSpec) bool {
	om, ktype := common.GetObjectMeta(obj)

	checks := map[string]bool{
		"observability": common.HasObservability(m),
		"liveness":      common.HasLiveness(spec),
		"readiness":     common.HasReadiness(spec),
		"limits":        common.HasLimits(spec),
	}

	values := []bool{}
	name := fmt.Sprintf("%s:%s.%s", ktype, om.GetName(), om.GetNamespace())
	for k, v := range checks {
		if !v {
			log.Printf("[%s] does not meet %s requirements", name, k)
		}

		values = append(values, v)
	}
	return !common.PassesChecks(values)
}
