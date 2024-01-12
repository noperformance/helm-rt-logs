package collector

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"strings"
)

// TailLogs initiates a log stream from a specified Kubernetes pod and continuously outputs the logs to the console.
// It supports options such as following the log stream and filtering logs based on a time range.
// The function also handles special cases where the pod has failed or is in an unknown phase.
func (c *Collector) TailLogs(pod corev1.Pod, resType, resName string) {
	var (
		podLogOptions corev1.PodLogOptions
	)

	if c.Opts.TimeSince > 0 {
		podLogOptions.SinceSeconds = &c.Opts.TimeSince
	}

	podLogOptions.Follow = true

	if pod.Status.Phase == "Failed" || pod.Status.Phase == "Unknown" {
		fmt.Printf("Pod %s failed with 60s timeout \n with message: %s\n with reason: %s\n", pod.Name, pod.Status.Message, pod.Status.Reason)
	}

	req := c.KubeClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOptions)
	stream, err := req.Stream(c.Ctx)
	if err != nil {
		fmt.Printf("Error opening stream to pod %s: %s\n", pod.Name, err)
		return
	}
	defer stream.Close()

	buf := make([]byte, 2000)
	for {
		select {
		case <-c.Ctx.Done():
			return
		default:
			n, err := stream.Read(buf)
			if err != nil {
				fmt.Printf("Error reading from stream: %s\n", err)
				return
			}

			if n > 0 {
				line := string(buf[:n])
				fmt.Printf("[ObjType: %v][ObjName: %v][PodName: %v][PodPhase: %v] %v \n --- \n", resType, resName, pod.Name, pod.Status.Phase, line)
				if c.Opts.StopString != "" && strings.Contains(line, c.Opts.StopString) {
					c.CancelFunction()
					return
				}
			}
		}
	}
}
