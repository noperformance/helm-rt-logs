package kubeclient

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewKubeClient(context, file string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags(context, file)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	return clientset, err
}
