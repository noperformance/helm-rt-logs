package main

import (
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"helm-rt-logs/cmd"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

func main() {
	settings := cli.New()

	actionConfig := new(action.Configuration)

	// double check if the env vars are set
	kubeConfigEnv := os.Getenv("KUBECONFIG")

	if kubeConfigEnv == "" {
		home := homedir.HomeDir()
		kubeConfigEnv = filepath.Join(home, ".kube", "config")
		settings.KubeConfig = kubeConfigEnv
	}

	err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), log.Printf)
	if err != nil {
		return
	}
	command := cmd.NewRtLogsCmd(actionConfig, os.Stdout, settings)
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
