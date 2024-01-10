package kubeclient

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewKubeClient(context, file string) (*kubernetes.Clientset, error) {

	kconf, err := clientcmd.BuildConfigFromFlags("", file)
	if err != nil {
		return nil, err
	}

	configLoadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: file}
	configOverrides := &clientcmd.ConfigOverrides{CurrentContext: context}

	kconf, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(configLoadingRules, configOverrides).ClientConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(kconf)
	return clientset, err
}
