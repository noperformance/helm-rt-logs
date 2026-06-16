package kubeclient

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// NewKubeClient builds a Kubernetes clientset from the given kubeconfig file and context.
// An empty context selects the current-context from the kubeconfig.
func NewKubeClient(context, file string) (*kubernetes.Clientset, error) {
	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: file}
	overrides := &clientcmd.ConfigOverrides{CurrentContext: context}

	kconf, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).ClientConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(kconf)
}
