package k8s

import (
	"os"
	"strings"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Creates a new k8s client, which will first try building
// from a ServiceAccout if it is inside a cluster, or will
// fallback to the path of a defined kubeconfig file
func NewClient(path string) (*dynamic.DynamicClient, error) {
	config, err := NewConfig(path)
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(config)
}

func NewConfig(path string) (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return envConfig(path)
	}
	return config, nil
}

func envConfig(path string) (*rest.Config, error) {
	if strings.Contains(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = strings.Replace(path, "~", home, -1)
	}
	return clientcmd.BuildConfigFromFlags("", path)
}
