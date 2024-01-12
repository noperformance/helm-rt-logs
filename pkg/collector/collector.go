package collector

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/release"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type RtLogsOpts struct {
	StopTimeout int
	StopString  string
	TimeSince   int64
	OnlyFailed  bool
	Debug       bool
}

type Collector struct {
	KubeClient     *kubernetes.Clientset
	ReleaseInfo    *release.Release
	Opts           *RtLogsOpts
	Ctx            context.Context
	CancelFunction context.CancelFunc
	PodsFound      bool
}

// ProcessResources handles the log collection from a set of Kubernetes resources (like deployments, statefulsets, or daemonsets) associated with a Helm release.
// It iterates over each resource, retrieves associated pods, and tails their logs concurrently.
// The function also handles pods in pending phase and filters pods based on their status if specified.
func (c *Collector) ProcessResources(resourceType string, resourceNames []string, podsFoundChan chan bool) error {
	var wg sync.WaitGroup
	suitablePodsFound := false

	for _, resourceName := range resourceNames {
		pods, err := c.GetPodsFromResource(resourceType, resourceName)
		if err != nil {
			return err
		}

		if len(pods) == 0 {
			continue
		} else {
			c.PodsFound = true
		}

		for _, pod := range pods {
			for pod.Status.Phase == "Pending" {
				fmt.Printf("[Pod %s] still in pending phase \n", pod.Name)
				time.Sleep(15 * time.Second)
			}

			if c.Opts.OnlyFailed && pod.Status.Phase != "Failed" {
				continue
			}

			suitablePodsFound = true
			wg.Add(1)
			go func(pod corev1.Pod) {
				defer wg.Done()
				c.TailLogs(pod, resourceType, resourceName)
			}(pod)
		}
	}

	if suitablePodsFound {
		select {
		case podsFoundChan <- true:
		default:
		}
	}

	wg.Wait()

	return nil
}

// CollectLogs handles the process of collecting logs from Kubernetes resources associated with a specific Helm release.
// It orchestrates the retrieval of deployments, statefulsets, and daemonsets, and filters them based on annotations.
// The function also manages concurrent processing of these resources to tail logs.
func (c *Collector) CollectLogs() error {

	var wg sync.WaitGroup
	podsFoundChan := make(chan bool, 1)

	if c.Opts.StopTimeout > 0 {
		time.AfterFunc(time.Duration(c.Opts.StopTimeout)*time.Second, func() {
			c.CancelFunction()
		})
	}
	//find all resources
	deployments, err := c.KubeClient.AppsV1().Deployments(c.ReleaseInfo.Namespace).List(c.Ctx, v1.ListOptions{})
	if err != nil {
		return err
	}

	if c.Opts.Debug {
		log.Info("Here deployments list: \n")
		for _, v := range deployments.Items {
			log.Info("\n\n[Name] ", v.Name)
			log.Info("\n[Spec] ", v.Spec)
			log.Info("\n[Labels] ", v.Labels)
			log.Info("\n[Annotations] ", v.Annotations)
			log.Info("\n[ObjMeta] ", v.ObjectMeta)
		}

	}

	statefullsets, err := c.KubeClient.AppsV1().StatefulSets(c.ReleaseInfo.Namespace).List(c.Ctx, v1.ListOptions{})
	if err != nil {
		return err
	}

	if c.Opts.Debug {
		log.Info("Here statefullsets list: \n")
		for _, v := range statefullsets.Items {
			log.Info("\n\n[Name] ", v.Name)
			log.Info("\n[Spec] ", v.Spec)
			log.Info("\n[Labels] ", v.Labels)
			log.Info("\n[Annotations] ", v.Annotations)
			log.Info("\n[ObjMeta] ", v.ObjectMeta)
		}

	}

	daemonsets, err := c.KubeClient.AppsV1().DaemonSets(c.ReleaseInfo.Namespace).List(c.Ctx, v1.ListOptions{})
	if err != nil {
		return err
	}

	if c.Opts.Debug {
		log.Info("Here daemonsets list: \n")
		for _, v := range daemonsets.Items {
			log.Info("\n\n[Name] ", v.Name)
			log.Info("\n[Spec] ", v.Spec)
			log.Info("\n[Labels] ", v.Labels)
			log.Info("\n[Annotations] ", v.Annotations)
			log.Info("\n[ObjMeta] ", v.ObjectMeta)
		}

	}

	var deploymentMetas []v1.ObjectMeta
	for _, d := range deployments.Items {
		deploymentMetas = append(deploymentMetas, d.ObjectMeta)
	}

	var stsMetas []v1.ObjectMeta
	for _, s := range statefullsets.Items {
		stsMetas = append(stsMetas, s.ObjectMeta)
	}

	var dsMetas []v1.ObjectMeta
	for _, d := range daemonsets.Items {
		dsMetas = append(dsMetas, d.ObjectMeta)
	}
	// filter all resources
	filteredDeployments := filterByAnnotation(deploymentMetas, "meta.helm.sh/release-name", c.ReleaseInfo.Name, c.Opts.Debug)
	filteredStatefullsets := filterByAnnotation(stsMetas, "meta.helm.sh/release-name", c.ReleaseInfo.Name, c.Opts.Debug)
	filteredDaemonsets := filterByAnnotation(dsMetas, "meta.helm.sh/release-name", c.ReleaseInfo.Name, c.Opts.Debug)

	if c.Opts.Debug {
		log.Info("Here filtered deployments list: ", filteredDeployments)
		log.Info("Here filtered statefullsets list: ", filteredStatefullsets)
		log.Info("Here filtered daemonsets list: ", filteredDaemonsets)
	}

	// process all resources
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := c.ProcessResources("deployment", filteredDeployments, podsFoundChan); err != nil {
			log.Printf("Error processing deployments: %s", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := c.ProcessResources("statefulset", filteredStatefullsets, podsFoundChan); err != nil {
			log.Printf("Error processing statefulsets: %s", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := c.ProcessResources("daemonset", filteredDaemonsets, podsFoundChan); err != nil {
			log.Printf("Error processing daemonsets: %s", err)
		}
	}()

	wg.Wait()

	select {
	case <-podsFoundChan:
		c.PodsFound = true
	default:
		c.PodsFound = false
	}

	if !c.PodsFound {
		fmt.Print("No pods to tail logs from found (probably all pods in Running phase for -o option, otherwise no pods in Release).")
		c.CancelFunction()
	}
	<-c.Ctx.Done()

	return nil
}
