package collector

import (
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"context"
)

func (c *Collector) GetPodsFromResource(resourceType string, resourceName string) ([]corev1.Pod, error) {
	var labelSelector string

	switch resourceType {
	case "deployment":
		deployment, err := c.KubeClient.AppsV1().Deployments(c.ReleaseInfo.Namespace).Get(context.Background(), resourceName, v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		labelSelector = v1.FormatLabelSelector(deployment.Spec.Selector)
	case "statefulset":
		sts, err := c.KubeClient.AppsV1().StatefulSets(c.ReleaseInfo.Namespace).Get(context.Background(), resourceName, v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		labelSelector = v1.FormatLabelSelector(sts.Spec.Selector)
	case "daemonset":
		ds, err := c.KubeClient.AppsV1().DaemonSets(c.ReleaseInfo.Namespace).Get(context.Background(), resourceName, v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		labelSelector = v1.FormatLabelSelector(ds.Spec.Selector)
	}

	pods, err := c.KubeClient.CoreV1().Pods(c.ReleaseInfo.Namespace).List(context.Background(), v1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

func filterByAnnotation(resources []v1.ObjectMeta, annotation, value string, debug bool) []string {
	var filtered []string
	for _, resource := range resources {
		if debug {
			log.Info("\n\n [Filtreing]")
			log.Info("\n [Filtreing][annotation] ", resource.Annotations[annotation])
			log.Info("\n [Filtreing][value] ", value)
		}
		if resource.Annotations[annotation] == value {
			filtered = append(filtered, resource.Name)
		}
	}
	return filtered
}
