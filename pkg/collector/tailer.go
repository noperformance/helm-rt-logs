package collector

import (
	"bufio"
	"errors"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// TailLogs streams the logs of a single container and writes them line by line until
// the stream ends (container terminated), the context is cancelled, or StopString is seen.
func (c *Collector) TailLogs(pod corev1.Pod, container, resType, resName string) {
	podLogOptions := corev1.PodLogOptions{
		Follow:     true,
		Container:  container,
		Timestamps: c.Opts.Timestamps,
	}
	if c.Opts.TimeSince > 0 {
		podLogOptions.SinceSeconds = &c.Opts.TimeSince
	}
	if c.Opts.Tail >= 0 {
		podLogOptions.TailLines = &c.Opts.Tail
	}

	if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodUnknown {
		c.printf("[Pod %s] phase=%s message=%q reason=%q\n", pod.Name, pod.Status.Phase, pod.Status.Message, pod.Status.Reason)
	}

	req := c.KubeClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOptions)
	stream, err := req.Stream(c.Ctx)
	if err != nil {
		c.printf("[Pod %s/%s] error opening stream: %s\n", pod.Name, container, err)
		return
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		select {
		case <-c.Ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		c.printf("[%s/%s][pod=%s][container=%s][phase=%s] %s\n",
			resType, resName, pod.Name, container, pod.Status.Phase, line)

		if c.Opts.StopString != "" && strings.Contains(line, c.Opts.StopString) {
			c.CancelFunction()
			return
		}
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, c.Ctx.Err()) {
		c.printf("[Pod %s/%s] stream error: %s\n", pod.Name, container, err)
	}
}
