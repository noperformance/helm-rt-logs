package collector

import (
	"context"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type RtLogsOpts struct {
	StopTimeout             int
	StopString              string
	TimeSince               int64
	WaitingFailedPodTimeout int
}

type Collector struct {
	KubeClient  *kubernetes.Clientset
	ReleaseInfo *release.Release
	Opts        *RtLogsOpts
}

func processResources(c *Collector, resourceType string, resourceNames []string, cancel context.CancelFunc, ctx context.Context) error {
	for _, resourceName := range resourceNames {
		pods, err := getPodsFromResource(c, resourceType, resourceName)
		if err != nil {
			return err
		}
		for _, pod := range pods {
			go tailLogs(ctx, c.KubeClient, pod, c.Opts.StopString, cancel, c.Opts.TimeSince, c.Opts.WaitingFailedPodTimeout, resourceType, resourceName)
		}
	}
	return nil
}

func getPodsFromResource(c *Collector, resourceType string, resourceName string) ([]corev1.Pod, error) {
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

func filterByAnnotation(resources []v1.ObjectMeta, annotation, value string) []string {
	var filtered []string
	for _, resource := range resources {
		if resource.Annotations[annotation] == value {
			filtered = append(filtered, resource.Name)
		}
	}
	return filtered
}

func CollectLogs(c Collector) error {

	ctx, cancel := context.WithCancel(context.Background())
	if c.Opts.StopTimeout > 0 {
		time.AfterFunc(time.Duration(c.Opts.StopTimeout)*time.Second, func() {
			cancel()
		})
	}

	deployments, err := c.KubeClient.AppsV1().Deployments(c.ReleaseInfo.Namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		return err
	}

	statefullsets, err := c.KubeClient.AppsV1().StatefulSets(c.ReleaseInfo.Namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		return err
	}

	daemonsets, err := c.KubeClient.AppsV1().DaemonSets(c.ReleaseInfo.Namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		return err
	}

	var deploymentMetas []v1.ObjectMeta
	for _, d := range deployments.Items {
		deploymentMetas = append(deploymentMetas, d.ObjectMeta)
	}

	var stsMetas []v1.ObjectMeta
	for _, d := range statefullsets.Items {
		deploymentMetas = append(stsMetas, d.ObjectMeta)
	}

	var dsMetas []v1.ObjectMeta
	for _, d := range daemonsets.Items {
		deploymentMetas = append(dsMetas, d.ObjectMeta)
	}

	filteredDeployments := filterByAnnotation(deploymentMetas, "meta.helm.sh/release-name", c.ReleaseInfo.Name)
	filteredStatefullsets := filterByAnnotation(stsMetas, "meta.helm.sh/release-name", c.ReleaseInfo.Name)
	filteredDaemonsets := filterByAnnotation(dsMetas, "meta.helm.sh/release-name", c.ReleaseInfo.Name)

	if err := processResources(&c, "deployment", filteredDeployments, cancel, ctx); err != nil {
		return err
	}
	if err := processResources(&c, "statefulset", filteredStatefullsets, cancel, ctx); err != nil {
		return err
	}
	if err := processResources(&c, "daemonset", filteredDaemonsets, cancel, ctx); err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}

func tailLogs(ctx context.Context, clientset *kubernetes.Clientset, pod corev1.Pod, stopOnString string, cancelFunc context.CancelFunc, timeSince int64, waitingfailedpodtimeout int, resType, resName string) {

	var (
		podLogOptions corev1.PodLogOptions
		podFailedMap  map[string]int
	)

	if timeSince > 0 {
		podLogOptions.SinceSeconds = &timeSince
	}

	podLogOptions.Follow = true

	for pod.Status.Phase != "Running" {
		if _, ok := podFailedMap[pod.Name]; ok {
			podFailedMap[pod.Name] += 1
		} else {
			podFailedMap[pod.Name] = 0
		}

		if podFailedMap[pod.Name] >= (waitingfailedpodtimeout / 10) {
			log.Printf("Pod %s failed with 60s timeout \n with message: %s\n with reason: %s\n", pod.Name, pod.Status.Message, pod.Status.Reason)
			return
		}

		time.Sleep(10 * time.Second)
		log.Printf("Pod %s not in Running phase, waiting 10s..", pod.Name)
	}

	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOptions)
	stream, err := req.Stream(ctx)
	if err != nil {
		log.Printf("Error opening stream to pod %s: %s\n", pod.Name, err)
		return
	}
	defer stream.Close()

	buf := make([]byte, 2000)
	for {
		select {
		case <-ctx.Done():
			return
		default:

			//log.Warning(<-describe.ResultChan())

			n, err := stream.Read(buf)
			if err != nil {
				log.Printf("Error reading from stream: %s\n", err)
				return
			}

			if n > 0 {
				line := string(buf[:n])
				log.Printf("[Type: %v][ObjName: %v][PodName: %v][Phase: %v] %v \n --- \n", resType, resName, pod.Name, pod.Status.Phase, line)
				if stopOnString != "" && strings.Contains(line, stopOnString) {
					cancelFunc()
					return
				}
			}
		}
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
