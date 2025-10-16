package k8s

import (
	"log/slog"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Config struct {
	InCluster bool
}

func NewClient(cfg Config) (*kubernetes.Clientset, error) {
	var (
		config *rest.Config
		err    error
	)

	if cfg.InCluster {
		config, err = rest.InClusterConfig()
		if err != nil {
			slog.Error("failed to get in cluster config", slog.String("error", err.Error()))
			return nil, err
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", "./config")
		if err != nil {
			slog.Error("failed to build config", slog.String("error", err.Error()))
			return nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		slog.Error("failed to get k8s client", slog.String("error", err.Error()))
		return nil, err
	}

	return clientset, err

}

// NewRESTConfig returns a Kubernetes REST config that can be used by other
// clientsets (e.g., dynamic client for CRDs like Knative Serving Service).
func NewRESTConfig(cfg Config) (*rest.Config, error) {
	var (
		config *rest.Config
		err    error
	)

	if cfg.InCluster {
		config, err = rest.InClusterConfig()
		if err != nil {
			slog.Error("failed to get in cluster config", slog.String("error", err.Error()))
			return nil, err
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", "./config")
		if err != nil {
			slog.Error("failed to build config", slog.String("error", err.Error()))
			return nil, err
		}
	}

	return config, nil
}

// NewDynamicClient returns a dynamic client built from the provided config.
// Useful for interacting with CRDs such as Knative Serving.
func NewDynamicClient(restConfig *rest.Config) (dynamic.Interface, error) {
	cli, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		slog.Error("failed to get dynamic client", slog.String("error", err.Error()))
		return nil, err
	}
	return cli, nil
}
