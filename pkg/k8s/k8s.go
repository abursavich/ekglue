package k8s

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

// ClusterWatcher watches services and endpoints inside of a cluster.
type ClusterWatcher struct {
	coreV1Client rest.Interface
}

// ConnectOutOfCluster connects to the API server from outside of the cluster.
func ConnectOutOfCluster(kubeconfig string, master string) (*ClusterWatcher, error) {
	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("kubernetes: build config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("kubernetes: new client: %w", err)
	}
	return &ClusterWatcher{coreV1Client: clientset.CoreV1().RESTClient()}, nil
}

// ConnectInCluster connects to the API server from a pod inside the cluster.
func ConnectInCluster() (*ClusterWatcher, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("kubernetes: get in-cluster config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("kubernetes: new client: %w", err)
	}
	return &ClusterWatcher{coreV1Client: clientset.CoreV1().RESTClient()}, nil
}

// WatchServices notifes the provided ServiceReceiver of changes to services, in all namespaces.
func (cw *ClusterWatcher) WatchServices(ctx context.Context, s cache.Store) error {
	lw := cache.NewListWatchFromClient(cw.coreV1Client, "services", "", fields.Everything())
	r := cache.NewReflector(lw, &v1.Service{}, s, 0)
	r.Run(ctx.Done())
	return nil
}