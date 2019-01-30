package suppressor

import (
	"github.com/ccpgames/kube-solskin-controller/common"
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

// Action type is an enumeration of the action the suppressor should take.
type Action string

const (
	// ActionNone represents taking no action, no logging nor suppression.
	ActionNone Action = "none"

	// ActionLog represents only logging resources that do not meet standards.
	ActionLog Action = "log"

	// ActionSuppress both logs and suppresses subpar resources.
	ActionSuppress Action = "suppress"
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

var c = cache.New(20*time.Second, 30*time.Second)

// Service is the base service for the suppressor service.
type Service struct {
	Configuration config.Config
	Client        kubernetes.Interface
}

// GetSlug returns the slug used for the configuration section.
func (s Service) GetSlug() string {
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
	action := s.Configuration.Get(s.GetSlug(), "action").String(string(ActionLog))

	// If we are configured to take no action, simply return.
	if action == string(ActionNone) {
		return
	}

	// Get the metadata of the resource.
	m, ktype := common.GetObjectMeta(obj)

	// Determine if the resource is eligible for suppression, if not skip it.
	if !common.IsEligible(obj, s.Configuration) {
		log.Printf("[%s] object in namespace [%s], not eligible", common.GetFullLabel(obj), m.GetNamespace())
		return
	}

	// Grab the unique identifier for the kubernetes resource.
	uid := string(m.GetUID())

	// Check to see if the resource has already been suppressed.
	fqname := common.GetFullLabel(obj)
	v, found := c.Get(uid)
	if found && v.(bool) {
		return
	}
	c.Set(uid, false, cache.DefaultExpiration)

	// If we don't need to suppress to object, simply return.
	if !s.toSuppress(obj) {
		log.Printf("[%s] meets standards, will not suppress", fqname)
		return
	}

	// If our configured action is anything other than suppress, exit early.
	if action != string(ActionSuppress) {
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
	c.Set(uid, true, cache.DefaultExpiration)

	// Perform the suppression of the resource only if we're configured to do so.
	log.Printf("[%s] will be suppressed", fqname)
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
	checks := map[string]bool{
		"observability": common.HasObservability(m),
		"liveness":      common.HasLiveness(spec),
		"readiness":     common.HasReadiness(spec),
		"requests":      common.HasRequests(spec),
		"limits":        common.HasLimits(spec),
	}

	values := []bool{}
	for k, v := range checks {
		if !v {
			log.Printf("[%s] does not meet %s requirements", common.GetFullLabel(obj), k)
		}

		values = append(values, v)
	}
	return !common.PassesChecks(values)
}
