package collector

import (
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetPodsFromResource retrieves the Pods belonging to a given workload (Deployment,
// StatefulSet or DaemonSet) by resolving the workload's label selector and listing
// matching pods in the release namespace.
func (c *Collector) GetPodsFromResource(resourceType string, resourceName string) ([]corev1.Pod, error) {
	var labelSelector string

	apps := c.KubeClient.AppsV1()
	ns := c.ReleaseInfo.Namespace

	switch resourceType {
	case "deployment":
		d, err := apps.Deployments(ns).Get(c.Ctx, resourceName, v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		labelSelector = v1.FormatLabelSelector(d.Spec.Selector)
	case "statefulset":
		s, err := apps.StatefulSets(ns).Get(c.Ctx, resourceName, v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		labelSelector = v1.FormatLabelSelector(s.Spec.Selector)
	case "daemonset":
		ds, err := apps.DaemonSets(ns).Get(c.Ctx, resourceName, v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		labelSelector = v1.FormatLabelSelector(ds.Spec.Selector)
	}

	pods, err := c.KubeClient.CoreV1().Pods(ns).List(c.Ctx, v1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

// filterByAnnotation returns the names of resources whose given annotation equals value.
func filterByAnnotation(resources []v1.ObjectMeta, annotation, value string, debug bool) []string {
	var filtered []string
	for _, r := range resources {
		if debug {
			log.Infof("[filter] name=%s annotation[%s]=%q want=%q", r.Name, annotation, r.Annotations[annotation], value)
		}
		if r.Annotations[annotation] == value {
			filtered = append(filtered, r.Name)
		}
	}
	return filtered
}

// debugDumpMeta logs name/labels/annotations for a set of resources when debug is on.
func debugDumpMeta(kind string, metas []v1.ObjectMeta) {
	for _, m := range metas {
		log.Infof("[%s] name=%s labels=%v annotations=%v", kind, m.Name, m.Labels, m.Annotations)
	}
}
