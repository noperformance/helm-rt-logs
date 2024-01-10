package collector

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/release"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type RtLogsOpts struct {
	StopTimeout             int
	StopString              string
	TimeSince               int64
	WaitingFailedPodTimeout int
	OnlyFailed              bool
	Debug                   bool
}

type Collector struct {
	KubeClient     *kubernetes.Clientset
	ReleaseInfo    *release.Release
	Opts           *RtLogsOpts
	Ctx            context.Context
	CancelFunction context.CancelFunc
	PodsFound      bool
}

func (c *Collector) ProcessResources(resourceType string, resourceNames []string) error {
	suitablePodsFound := false

	for _, resourceName := range resourceNames {
		pods, err := c.GetPodsFromResource(resourceType, resourceName)
		if err != nil {
			return err
		}

		if len(pods) == 0 {
			return nil
		} else {
			c.PodsFound = true
		}

		for _, pod := range pods {

			for pod.Status.Phase == "Pending" {
				fmt.Printf("[Pod %s] still in pending phase \n", pod.Name)
				time.Sleep(15 * time.Second)
			}

			if c.Opts.OnlyFailed && pod.Status.Phase == "Running" {
				return nil
			}

			suitablePodsFound = true
			c.TailLogs(pod, resourceType, resourceName)
		}
	}

	if !suitablePodsFound {
		c.PodsFound = false
	}

	return nil
}

func (c *Collector) CollectLogs() error {

	if c.Opts.StopTimeout > 0 {
		time.AfterFunc(time.Duration(c.Opts.StopTimeout)*time.Second, func() {
			c.CancelFunction()
		})
	}

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

	filteredDeployments := filterByAnnotation(deploymentMetas, "meta.helm.sh/release-name", c.ReleaseInfo.Name, c.Opts.Debug)
	filteredStatefullsets := filterByAnnotation(stsMetas, "meta.helm.sh/release-name", c.ReleaseInfo.Name, c.Opts.Debug)
	filteredDaemonsets := filterByAnnotation(dsMetas, "meta.helm.sh/release-name", c.ReleaseInfo.Name, c.Opts.Debug)

	if c.Opts.Debug {
		log.Info("Here filtered deployments list: ", filteredDeployments)
		log.Info("Here filtered statefullsets list: ", filteredStatefullsets)
		log.Info("Here filtered daemonsets list: ", filteredDaemonsets)
	}

	if err := c.ProcessResources("deployment", filteredDeployments); err != nil {
		return err
	}
	if err := c.ProcessResources("statefulset", filteredStatefullsets); err != nil {
		return err
	}
	if err := c.ProcessResources("daemonset", filteredDaemonsets); err != nil {
		return err
	}

	if !c.PodsFound {
		fmt.Print("no pods found")
		c.CancelFunction()
	}
	<-c.Ctx.Done()

	return nil
}
