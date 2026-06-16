package collector

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// pendingPollInterval is how often a Pending pod is re-fetched while waiting for it to schedule.
const pendingPollInterval = 5 * time.Second

type RtLogsOpts struct {
	StopTimeout int
	StopString  string
	TimeSince   int64
	Tail        int64 // last N lines per container; <0 means all
	Timestamps  bool  // prefix each line with an RFC3339 timestamp
	OnlyFailed  bool
	Container   string // when set, tail only this container
	Debug       bool
}

type Collector struct {
	KubeClient     kubernetes.Interface
	ReleaseInfo    *release.Release
	Opts           *RtLogsOpts
	Ctx            context.Context
	CancelFunction context.CancelFunc
	Out            io.Writer

	outMu     sync.Mutex  // serializes concurrent writes to Out
	podsFound atomic.Bool // set when at least one pod was selected for tailing
}

// printf writes formatted output to the collector's writer under a lock so that
// lines from concurrent tailers do not interleave.
func (c *Collector) printf(format string, a ...interface{}) {
	c.outMu.Lock()
	defer c.outMu.Unlock()
	fmt.Fprintf(c.Out, format, a...)
}

// waitForPodScheduled re-fetches a Pending pod until it leaves the Pending phase,
// the context is cancelled, or the pod disappears. The original List returns a stale
// snapshot, so the phase must be polled from the API rather than read from the copy.
func (c *Collector) waitForPodScheduled(pod corev1.Pod) (corev1.Pod, error) {
	for pod.Status.Phase == corev1.PodPending {
		c.printf("[Pod %s] still pending\n", pod.Name)
		select {
		case <-c.Ctx.Done():
			return pod, c.Ctx.Err()
		case <-time.After(pendingPollInterval):
		}
		fresh, err := c.KubeClient.CoreV1().Pods(pod.Namespace).Get(c.Ctx, pod.Name, v1.GetOptions{})
		if err != nil {
			return pod, err
		}
		pod = *fresh
	}
	return pod, nil
}

// ProcessResources tails logs for every pod of the given workloads concurrently.
// Pending pods are waited on, non-running pods are skipped when OnlyFailed is set,
// and one tailer is started per container.
func (c *Collector) ProcessResources(resourceType string, resourceNames []string) error {
	var wg sync.WaitGroup

	for _, resourceName := range resourceNames {
		pods, err := c.GetPodsFromResource(resourceType, resourceName)
		if err != nil {
			return err
		}

		for _, pod := range pods {
			pod, err := c.waitForPodScheduled(pod)
			if err != nil {
				return err
			}

			if c.Opts.OnlyFailed && pod.Status.Phase == corev1.PodRunning {
				continue
			}

			for _, container := range pod.Spec.Containers {
				if c.Opts.Container != "" && container.Name != c.Opts.Container {
					continue
				}
				c.podsFound.Store(true)
				wg.Add(1)
				go func(pod corev1.Pod, container string) {
					defer wg.Done()
					c.TailLogs(pod, container, resourceType, resourceName)
				}(pod, container.Name)
			}
		}
	}

	wg.Wait()
	return nil
}

// CollectLogs discovers the Deployments, StatefulSets and DaemonSets belonging to the
// release, then tails the logs of their pods until every stream ends or the run is
// stopped (StopTimeout, StopString or an interrupt signal).
func (c *Collector) CollectLogs() error {
	defer c.CancelFunction()

	if c.Opts.StopTimeout > 0 {
		time.AfterFunc(time.Duration(c.Opts.StopTimeout)*time.Second, c.CancelFunction)
	}

	ns := c.ReleaseInfo.Namespace
	apps := c.KubeClient.AppsV1()

	deployments, err := apps.Deployments(ns).List(c.Ctx, v1.ListOptions{})
	if err != nil {
		return err
	}
	statefulsets, err := apps.StatefulSets(ns).List(c.Ctx, v1.ListOptions{})
	if err != nil {
		return err
	}
	daemonsets, err := apps.DaemonSets(ns).List(c.Ctx, v1.ListOptions{})
	if err != nil {
		return err
	}

	var depMetas, stsMetas, dsMetas []v1.ObjectMeta
	for _, d := range deployments.Items {
		depMetas = append(depMetas, d.ObjectMeta)
	}
	for _, s := range statefulsets.Items {
		stsMetas = append(stsMetas, s.ObjectMeta)
	}
	for _, d := range daemonsets.Items {
		dsMetas = append(dsMetas, d.ObjectMeta)
	}

	if c.Opts.Debug {
		debugDumpMeta("deployment", depMetas)
		debugDumpMeta("statefulset", stsMetas)
		debugDumpMeta("daemonset", dsMetas)
	}

	const releaseAnnotation = "meta.helm.sh/release-name"
	filtered := map[string][]string{
		"deployment":  filterByAnnotation(depMetas, releaseAnnotation, c.ReleaseInfo.Name, c.Opts.Debug),
		"statefulset": filterByAnnotation(stsMetas, releaseAnnotation, c.ReleaseInfo.Name, c.Opts.Debug),
		"daemonset":   filterByAnnotation(dsMetas, releaseAnnotation, c.ReleaseInfo.Name, c.Opts.Debug),
	}

	if c.Opts.Debug {
		log.Infof("[filtered] %v", filtered)
	}

	var wg sync.WaitGroup
	for resourceType, names := range filtered {
		wg.Add(1)
		go func(resourceType string, names []string) {
			defer wg.Done()
			if err := c.ProcessResources(resourceType, names); err != nil {
				log.Printf("error processing %ss: %s", resourceType, err)
			}
		}(resourceType, names)
	}
	wg.Wait()

	if !c.podsFound.Load() {
		c.printf("No pods to tail logs from found (with -o all pods may be Running, otherwise the release has no pods).\n")
	}

	return nil
}
