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

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	settings := cli.New()

	actionConfig := new(action.Configuration)

	// Resolve the kubeconfig path: honor KUBECONFIG, otherwise fall back to ~/.kube/config.
	// Always set it on settings so the path is respected regardless of HELM_KUBECONFIG.
	kubeConfigEnv := os.Getenv("KUBECONFIG")
	if kubeConfigEnv == "" {
		kubeConfigEnv = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}
	settings.KubeConfig = kubeConfigEnv

	err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), log.Printf)
	if err != nil {
		log.Fatalf("failed to init helm action config: %s", err)
	}
	command := cmd.NewRtLogsCmd(actionConfig, os.Stdout, settings)
	command.Version = version
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
