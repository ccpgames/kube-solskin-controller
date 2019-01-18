package main

import (
	"log"

	config "github.com/micro/go-config"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"regexp"
	"time"
)

func startMetricUpdater(client kubernetes.Interface, cfg config.Config) {
	// Create our informer.
	factory := informers.NewSharedInformerFactory(client, time.Second)
	dplInformer := factory.Apps().V1().Deployments().Informer()
	dmsInformer := factory.Apps().V1().DaemonSets().Informer()
	podInformer := factory.Core().V1().Pods().Informer()
	stopper := make(chan struct{})
	defer close(stopper)

	// Setup our daemonset informer.
	dmsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			onDaemonSetUpdate(obj.(*appsv1.DaemonSet), cfg)
		},
		UpdateFunc: func(old interface{}, obj interface{}) {
			onDaemonSetUpdate(obj.(*appsv1.DaemonSet), cfg)
		},
		DeleteFunc: func(obj interface{}) {
			onDaemonSetDelete(obj.(*appsv1.DaemonSet), cfg)
		},
	})

	// Setup our deployment informer.
	dplInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			onDeploymentUpdate(obj.(*appsv1.Deployment), cfg)
		},
		UpdateFunc: func(old interface{}, obj interface{}) {
			onDeploymentUpdate(obj.(*appsv1.Deployment), cfg)
		},
		DeleteFunc: func(obj interface{}) {
			onDeploymentDelete(obj.(*appsv1.Deployment), cfg)
		},
	})

	// Setup our pod informer.
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			onPodUpdate(obj.(*corev1.Pod), cfg)
		},
		UpdateFunc: func(old interface{}, obj interface{}) {
			onPodUpdate(obj.(*corev1.Pod), cfg)
		},
		DeleteFunc: func(obj interface{}) {
			onPodDelete(obj.(*corev1.Pod), cfg)
		},
	})

	informers := make([]cache.SharedIndexInformer, 3)
	informers[0] = dplInformer
	informers[1] = dmsInformer
	informers[2] = podInformer

	// Start each informer.
	for _, informer := range informers {
		go informer.Run(stopper)
	}

	// Wait until each informer has synced before returning.
	for _, informer := range informers {
		for !informer.HasSynced() {
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func onDaemonSetDelete(daemonset *appsv1.DaemonSet, cfg config.Config) {
	// Get the ignore namespace regexp pattern from the configuration.
	pattern := cfg.Get("ignore_namespace_pattern").String("^kube-")

	// Check metadata to see if we can ignore it.
	m, err := regexp.MatchString(pattern, daemonset.GetNamespace())
	if err != nil {
		log.Fatal(err)
	}

	// If we matched the ignore namespace pattern, simply return.
	if m {
		return
	}

	labels := map[string]string{
		"name":          daemonset.GetName(),
		"namespace":     daemonset.GetNamespace(),
		"resource_type": "daemonset",
	}
	for _, metric := range metrics {
		metric.Delete(labels)
	}
}

// Called whenever a deployment is added or updated in/to the cluster.
func onDaemonSetUpdate(daemonset *appsv1.DaemonSet, cfg config.Config) {
	// Get the ignore namespace regexp pattern from the configuration.
	pattern := cfg.Get("ignore_namespace_pattern").String("^kube-")

	// Check metadata to see if we can ignore it.
	m, err := regexp.MatchString(pattern, daemonset.GetNamespace())
	if err != nil {
		log.Fatal(err)
	}

	// If we matched the ignore namespace pattern, simply return.
	if m {
		return
	}

	// Update observability metrics.
	forDaemonSetObservability(daemonset)
	forDaemonSetLiveness(daemonset)
	forDaemonSetReadiness(daemonset)
	forDaemonSetLimits(daemonset)
}

func forDaemonSetObservability(daemonset *appsv1.DaemonSet) {
	labels := map[string]string{
		"name":          daemonset.GetName(),
		"namespace":     daemonset.GetNamespace(),
		"resource_type": "daemonset",
	}

	// Create or retrieve our metric.
	gauge, err := metrics["solskin_observability_resources"].GetMetricWith(labels)
	if err != nil {
		log.Fatal(err)
	}

	// Set our metric.
	meta := daemonset.Spec.Template.ObjectMeta
	gauge.Set(boolToFloat64(hasAnnotation(meta, "prometheus.io/scrape")))
}

func forDaemonSetLiveness(daemonset *appsv1.DaemonSet) {
	labels := map[string]string{
		"name":          daemonset.GetName(),
		"namespace":     daemonset.GetNamespace(),
		"resource_type": "daemonset",
	}

	// Create or retrieve our metric.
	gauge, err := metrics["solskin_liveness_resources"].GetMetricWith(labels)
	if err != nil {
		log.Fatal(err)
	}

	gauge.Set(float64(countLiveness(daemonset.Spec.Template.Spec)))
}

func forDaemonSetReadiness(daemonset *appsv1.DaemonSet) {
	labels := map[string]string{
		"name":          daemonset.GetName(),
		"namespace":     daemonset.GetNamespace(),
		"resource_type": "daemonset",
	}

	// Create or retrieve our metric.
	gauge, err := metrics["solskin_readiness_resources"].GetMetricWith(labels)
	if err != nil {
		log.Fatal(err)
	}

	gauge.Set(float64(countReadiness(daemonset.Spec.Template.Spec)))
}

func forDaemonSetLimits(daemonset *appsv1.DaemonSet) {
	labels := map[string]string{
		"name":          daemonset.GetName(),
		"namespace":     daemonset.GetNamespace(),
		"resource_type": "daemonset",
	}

	// Create or retrieve our metric.
	gauge, err := metrics["solskin_limits_resources"].GetMetricWith(labels)
	if err != nil {
		log.Fatal(err)
	}

	gauge.Set(float64(countLimits(daemonset.Spec.Template.Spec)))
}

func onDeploymentDelete(deployment *appsv1.Deployment, cfg config.Config) {
	log.Printf("deleted deployment [%s.%s]", deployment.GetName(), deployment.GetNamespace())

	// Get the ignore namespace regexp pattern from the configuration.
	pattern := cfg.Get("ignore_namespace_pattern").String("^kube-")

	// Check metadata to see if we can ignore it.
	m, err := regexp.MatchString(pattern, deployment.GetNamespace())
	if err != nil {
		log.Fatal(err)
	}

	// If we matched the ignore namespace pattern, simply return.
	if m {
		return
	}

	labels := map[string]string{
		"name":          deployment.GetName(),
		"namespace":     deployment.GetNamespace(),
		"resource_type": "deployment",
	}
	for _, metric := range metrics {
		metric.Delete(labels)
	}
}

// Called whenever a deployment is added or updated in/to the cluster.
func onDeploymentUpdate(deployment *appsv1.Deployment, cfg config.Config) {
	log.Printf("update to deployment [%s.%s]", deployment.GetName(), deployment.GetNamespace())

	// Get the ignore namespace regexp pattern from the configuration.
	pattern := cfg.Get("ignore_namespace_pattern").String("^kube-")

	// Check metadata to see if we can ignore it.
	m, err := regexp.MatchString(pattern, deployment.GetNamespace())
	if err != nil {
		log.Fatal(err)
	}

	// If we matched the ignore namespace pattern, simply return.
	if m {
		return
	}

	// Update observability metrics.
	forDeploymentObservability(deployment)
	forDeploymentLiveness(deployment)
	forDeploymentReadiness(deployment)
	forDeploymentLimits(deployment)
}

func forDeploymentObservability(deployment *appsv1.Deployment) {
	labels := map[string]string{
		"name":          deployment.GetName(),
		"namespace":     deployment.GetNamespace(),
		"resource_type": "deployment",
	}

	// Create or retrieve our metric.
	gauge, err := metrics["solskin_observability_resources"].GetMetricWith(labels)
	if err != nil {
		log.Fatal(err)
	}

	// Set our metric.
	meta := deployment.Spec.Template.ObjectMeta
	gauge.Set(boolToFloat64(hasAnnotation(meta, "prometheus.io/scrape")))
}

func forDeploymentLiveness(deployment *appsv1.Deployment) {
	labels := map[string]string{
		"name":          deployment.GetName(),
		"namespace":     deployment.GetNamespace(),
		"resource_type": "deployment",
	}

	// Create or retrieve our metric.
	gauge, err := metrics["solskin_liveness_resources"].GetMetricWith(labels)
	if err != nil {
		log.Fatal(err)
	}

	gauge.Set(float64(countLiveness(deployment.Spec.Template.Spec)))
}

func forDeploymentReadiness(deployment *appsv1.Deployment) {
	labels := map[string]string{
		"name":          deployment.GetName(),
		"namespace":     deployment.GetNamespace(),
		"resource_type": "deployment",
	}

	// Create or retrieve our metric.
	gauge, err := metrics["solskin_readiness_resources"].GetMetricWith(labels)
	if err != nil {
		log.Fatal(err)
	}

	gauge.Set(float64(countReadiness(deployment.Spec.Template.Spec)))
}

func forDeploymentLimits(deployment *appsv1.Deployment) {
	labels := map[string]string{
		"name":          deployment.GetName(),
		"namespace":     deployment.GetNamespace(),
		"resource_type": "deployment",
	}

	// Create or retrieve our metric.
	gauge, err := metrics["solskin_limits_resources"].GetMetricWith(labels)
	if err != nil {
		log.Fatal(err)
	}

	gauge.Set(float64(countLimits(deployment.Spec.Template.Spec)))
}

func onPodDelete(pod *corev1.Pod, cfg config.Config) {
	// Get the ignore namespace regexp pattern from the configuration.
	pattern := cfg.Get("ignore_namespace_pattern").String("^kube-")

	// Check metadata to see if we can ignore it.
	m, err := regexp.MatchString(pattern, pod.GetNamespace())
	if err != nil {
		log.Fatal(err)
	}

	// If we matched the ignore namespace pattern, simply return.
	if m {
		return
	}

	labels := map[string]string{
		"name":          pod.GetName(),
		"namespace":     pod.GetNamespace(),
		"resource_type": "pod",
	}
	for _, metric := range metrics {
		metric.Delete(labels)
	}
}

// Called whenever a pod is added or updated in/to the cluster.
func onPodUpdate(pod *corev1.Pod, cfg config.Config) {
	// Get the ignore namespace regexp pattern from the configuration.
	pattern := cfg.Get("ignore_namespace_pattern").String("^kube-")

	// Check metadata to see if we can ignore it.
	m, err := regexp.MatchString(pattern, pod.GetNamespace())
	if err != nil {
		log.Fatal(err)
	}

	// If we matched the ignore namespace pattern, simply return.
	if m {
		return
	}

	// Update observability metrics.
	forPodObservability(pod)
	forPodLiveness(pod)
	forPodReadiness(pod)
	forPodLimits(pod)
}

func forPodObservability(pod *corev1.Pod) {
	labels := map[string]string{
		"name":          pod.GetName(),
		"namespace":     pod.GetNamespace(),
		"resource_type": "pod",
	}

	// Create or retrieve our metric.
	gauge, err := metrics["solskin_observability_resources"].GetMetricWith(labels)
	if err != nil {
		log.Fatal(err)
	}

	// Set our metric.
	meta := pod.ObjectMeta
	gauge.Set(boolToFloat64(hasAnnotation(meta, "prometheus.io/scrape")))
}

func forPodLiveness(pod *corev1.Pod) {
	labels := map[string]string{
		"name":          pod.GetName(),
		"namespace":     pod.GetNamespace(),
		"resource_type": "pod",
	}

	// Create or retrieve our metric.
	gauge, err := metrics["solskin_liveness_resources"].GetMetricWith(labels)
	if err != nil {
		log.Fatal(err)
	}

	gauge.Set(float64(countLiveness(pod.Spec)))
}

func forPodReadiness(pod *corev1.Pod) {
	labels := map[string]string{
		"name":          pod.GetName(),
		"namespace":     pod.GetNamespace(),
		"resource_type": "pod",
	}

	// Create or retrieve our metric.
	gauge, err := metrics["solskin_readiness_resources"].GetMetricWith(labels)
	if err != nil {
		log.Fatal(err)
	}

	gauge.Set(float64(countReadiness(pod.Spec)))
}

func forPodLimits(pod *corev1.Pod) {
	labels := map[string]string{
		"name":          pod.GetName(),
		"namespace":     pod.GetNamespace(),
		"resource_type": "pod",
	}

	// Create or retrieve our metric.
	gauge, err := metrics["solskin_limits_resources"].GetMetricWith(labels)
	if err != nil {
		log.Fatal(err)
	}

	gauge.Set(float64(countLimits(pod.Spec)))
}

func countLiveness(spec corev1.PodSpec) uint8 {
	count := uint8(0)
	for _, container := range spec.Containers {
		if container.LivenessProbe != nil {
			count += uint8(1)
		}
	}
	return count
}

func countReadiness(spec corev1.PodSpec) uint8 {
	count := uint8(0)
	for _, container := range spec.Containers {
		if container.ReadinessProbe != nil {
			count += uint8(1)
		}
	}
	return count
}

func countLimits(spec corev1.PodSpec) uint8 {
	count := uint8(0)
	for _, container := range spec.Containers {
		count += uint8(len(container.Resources.Limits))
	}
	return count
}

func hasAnnotation(meta metav1.ObjectMeta, annotation string) bool {
	annotations := meta.Annotations
	for key := range annotations {
		if key == annotation {
			return true
		}
	}

	return false
}

func boolToFloat64(value bool) float64 {
	if value {
		return 1.0
	}
	return 0.0
}
