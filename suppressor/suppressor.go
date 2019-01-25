package suppressor

import (
	"fmt"
	"github.com/celestialorb/solskin/common"
	"github.com/micro/go-config"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kcache "k8s.io/client-go/tools/cache"
	"log"
	"time"
)

var suppressedResourcesMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Help: "Counter of suppressed kubernetes resources.",
		Name: "solskin_suppressed_resources",
	},
	[]string{
		"name",
		"namespace",
		"resource_type",
	},
)

var c = cache.New(5*time.Minute, 5*time.Minute)

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
func (s Service) GenerateEventHandlers() []kcache.ResourceEventHandlerFuncs {
	return []kcache.ResourceEventHandlerFuncs{
		kcache.ResourceEventHandlerFuncs{
			AddFunc:    func(obj interface{}) { s.onObjectChange(obj) },
			UpdateFunc: func(_, obj interface{}) { s.onObjectChange(obj) },
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

	// Get the metadata of the resource.
	m, ktype := common.GetObjectMeta(obj)

	// Grab the unique identifier for the kubernetes resource.
	uid := string(m.GetUID())

	// Check to see if the resource has already been suppressed.
	fqname := fmt.Sprintf("%s.%s", m.GetName(), m.GetNamespace())
	v, found := c.Get(uid)
	if found && v.(bool) {
		return
	}
	c.Set(uid, false, cache.DefaultExpiration)

	// If we don't need to suppress to object, simply return.
	if !s.toSuppress(obj) {
		return
	}

	// Increment our metric counter by one.
	suppressedResourcesMetric.With(map[string]string{
		"name":          m.GetName(),
		"namespace":     m.GetNamespace(),
		"resource_type": ktype,
	}).Add(1.0)

	// If the resource is eligible then we have to suppress it, which will depend
	// on the type of the resource.
	opts := &meta.DeleteOptions{}
	log.Printf("[%s:%s] will be suppressed", ktype, fqname)
	c.Set(uid, true, cache.DefaultExpiration)
	switch ktype {
	case "pod":
		pod := obj.(*core.Pod)

		// To suppress a pod, we simply delete it.
		s.Client.Core().Pods(m.Namespace).Delete(pod.GetName(), opts)
	case "deployment":
		dpl := obj.(*apps.Deployment)

		// To suppress a deployment, we set the replicas to zero.
		replicas := int32(0)
		dpl.Spec.Replicas = &replicas
		s.Client.Apps().Deployments(m.Namespace).Update(dpl)
	case "daemonset":
		ds := obj.(*apps.DaemonSet)

		// To suppress a daemonset, we simply delete it.
		s.Client.Apps().DaemonSets(m.Namespace).Delete(ds.GetName(), opts)
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

	uid := string(om.GetUID())

	values := []bool{}
	name := fmt.Sprintf("%s:%s.%s:%s", ktype, om.GetName(), om.GetNamespace(), uid)
	for k, v := range checks {
		if !v {
			log.Printf("[%s] does not meet %s requirements", name, k)
		}

		values = append(values, v)
	}
	return !common.PassesChecks(values)
}
