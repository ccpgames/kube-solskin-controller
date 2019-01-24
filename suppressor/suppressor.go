package suppressor

import (
	"github.com/celestialorb/solskin/common"
	"github.com/micro/go-config"
	"github.com/prometheus/client_golang/prometheus"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var suppressedResourcesMetric = prometheus.NewCounter(prometheus.CounterOpts{
	Help: "TODO",
	Name: "solskin_suppressed_resources",
})

// Service TODO
type Service struct {
	Configuration config.Config
	Client        kubernetes.Interface
}

// GetConfigurationSlug TODO
func (s Service) GetConfigurationSlug() string {
	return "suppressor"
}

// GenerateEventHandlers TODO
func (s Service) GenerateEventHandlers() []cache.ResourceEventHandlerFuncs {
	return []cache.ResourceEventHandlerFuncs{
		cache.ResourceEventHandlerFuncs{
			AddFunc:    func(obj interface{}) { s.onObjectChange(obj) },
			UpdateFunc: func(_, obj interface{}) { s.onObjectChange(obj) },
			DeleteFunc: func(obj interface{}) { s.onObjectDelete(obj) },
		},
	}
}

// Init TODO
func (s Service) Init() {
	// Initialize the suppressor metrics.
	prometheus.MustRegister(suppressedResourcesMetric)
}

// Start TODO
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

	// Determine if the resource should be suppressed.
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
		s.Client.Core().Pods(m.Namespace).Delete(pod.GetName(), &meta.DeleteOptions{})
	case "deployment":
		dpl := obj.(*apps.Deployment)
		dpl.Spec.Paused = true
		s.Client.Apps().Deployments(m.Namespace).Update(dpl)
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
