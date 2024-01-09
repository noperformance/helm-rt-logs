package collector

import (
	"context"
	"fmt"
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

func CollectLogs(c Collector) error {

	ctx, cancel := context.WithCancel(context.Background())
	if c.Opts.StopTimeout > 0 {
		time.AfterFunc(time.Duration(c.Opts.StopTimeout)*time.Second, func() {
			cancel()
		})
	}

	pods, err := c.KubeClient.CoreV1().Pods(c.ReleaseInfo.Namespace).List(ctx, v1.ListOptions{
		LabelSelector: fmt.Sprintf("meta.helm.sh/release-name=%s", c.ReleaseInfo.Name),
	})
	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		go tailLogs(ctx, c.KubeClient, pod, c.Opts.StopString, cancel, c.Opts.TimeSince, c.Opts.WaitingFailedPodTimeout)
	}

	<-ctx.Done()
	cancel()
	return nil
}

func tailLogs(ctx context.Context, clientset *kubernetes.Clientset, pod corev1.Pod, stopOnString string, cancelFunc context.CancelFunc, timeSince int64, waitingfailedpodtimeout int) {

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
				log.Printf("[Name: %v][Phase: %v] %v \n --- \n", pod.Name, pod.Status.Phase, line)
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
