package collector

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"helm.sh/helm/v3/pkg/release"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestFilterByAnnotation(t *testing.T) {
	metas := []v1.ObjectMeta{
		{Name: "a", Annotations: map[string]string{"meta.helm.sh/release-name": "rel"}},
		{Name: "b", Annotations: map[string]string{"meta.helm.sh/release-name": "other"}},
		{Name: "c"}, // no annotations
	}
	got := filterByAnnotation(metas, "meta.helm.sh/release-name", "rel", false)
	if len(got) != 1 || got[0] != "a" {
		t.Fatalf("want [a], got %v", got)
	}
}

func newCollector(t *testing.T, objs ...runtime.Object) (*Collector, *bytes.Buffer) {
	t.Helper()
	cs := fake.NewSimpleClientset(objs...)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	buf := &bytes.Buffer{}
	c := &Collector{
		KubeClient:     cs,
		ReleaseInfo:    &release.Release{Name: "rel", Namespace: "ns"},
		Opts:           &RtLogsOpts{},
		Ctx:            ctx,
		CancelFunction: cancel,
		Out:            buf,
	}
	return c, buf
}

func deployment(name string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:        name,
			Namespace:   "ns",
			Annotations: map[string]string{"meta.helm.sh/release-name": "rel"},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &v1.LabelSelector{MatchLabels: map[string]string{"app": name}},
		},
	}
}

func runningPod(name, app string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{Name: name, Namespace: "ns", Labels: map[string]string{"app": app}},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}}},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
}

func TestGetPodsFromResource(t *testing.T) {
	c, _ := newCollector(t, deployment("web"), runningPod("web-1", "web"), runningPod("web-2", "web"))
	pods, err := c.GetPodsFromResource("deployment", "web")
	if err != nil {
		t.Fatal(err)
	}
	if len(pods) != 2 {
		t.Fatalf("want 2 pods, got %d", len(pods))
	}
}

func TestCollectLogsTailsRunningPod(t *testing.T) {
	c, buf := newCollector(t, deployment("web"), runningPod("web-1", "web"))
	if err := c.CollectLogs(); err != nil {
		t.Fatal(err)
	}
	if !c.podsFound.Load() {
		t.Fatal("expected podsFound=true")
	}
	out := buf.String()
	if !strings.Contains(out, "fake logs") {
		t.Fatalf("expected tailed log output, got: %q", out)
	}
	if !strings.Contains(out, "container=main") {
		t.Fatalf("expected container label in output, got: %q", out)
	}
}

func TestCollectLogsNoPods(t *testing.T) {
	c, buf := newCollector(t, deployment("web"))
	if err := c.CollectLogs(); err != nil {
		t.Fatal(err)
	}
	if c.podsFound.Load() {
		t.Fatal("expected podsFound=false")
	}
	if !strings.Contains(buf.String(), "No pods to tail") {
		t.Fatalf("expected no-pods message, got: %q", buf.String())
	}
}

func TestOnlyFailedSkipsRunning(t *testing.T) {
	c, buf := newCollector(t, deployment("web"), runningPod("web-1", "web"))
	c.Opts.OnlyFailed = true
	if err := c.CollectLogs(); err != nil {
		t.Fatal(err)
	}
	if c.podsFound.Load() {
		t.Fatal("running pod must be skipped with OnlyFailed")
	}
	_ = buf
}
