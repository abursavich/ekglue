// Command "cds" runs a gRPC server that serves an Envoy CDS and EDS server.
package main

import (
	"context"
	"net/http"

	"github.com/jrockway/ekglue/pkg/glue"
	"github.com/jrockway/ekglue/pkg/k8s"
	"github.com/jrockway/ekglue/pkg/xds"
	"github.com/jrockway/opinionated-server/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

type kflags struct {
	Kubeconfig string `long:"kubeconfig" env:"KUBECONFIG" description:"kubeconfig to use to connect to the cluster, when running outside of the cluster"`
	Master     string `long:"master" env:"KUBE_MASTER" description:"url of the kubernetes master, only necessary when running outside of the cluster and when it's not specified in the provided kubeconfig"`
}

type flags struct {
	Config        string `short:"c" long:"config" env:"EKGLUE_CONFIG_FILE" description:"config file to read"`
	VersionPrefix string `long:"version_prefix" env:"VERSION_PREFIX" description:"a string to prepend to the version number that we use to identify the generated configuration to envoy and in metrics"`
}

func main() {
	server.AppName = "ekglue-cds"

	f := new(flags)
	server.AddFlagGroup("ekglue", f)
	kf := new(kflags)
	server.AddFlagGroup("Kubernetes", kf)
	server.Setup()

	svc := xds.NewServer(f.VersionPrefix)
	server.AddService(func(s *grpc.Server) {
		envoy_api_v2.RegisterClusterDiscoveryServiceServer(s, svc)
	})
	http.Handle("/config_dump", svc)

	var watcher *k8s.ClusterWatcher
	if kf.Kubeconfig != "" || kf.Master != "" {
		var err error
		zap.L().Info("connecting to kubernetes, outside of cluster")
		watcher, err = k8s.ConnectOutOfCluster(kf.Kubeconfig, kf.Master)
		if err != nil {
			zap.L().Fatal("problem connecting to cluster via kubeconfig", zap.String("kubeconfig", kf.Kubeconfig), zap.String("master", kf.Master), zap.Error(err))
		}
	} else {
		var err error
		zap.L().Info("connecting to kubernetes, running in-cluster")
		watcher, err = k8s.ConnectInCluster()
		if err != nil {
			zap.L().Fatal("problem connecting to cluster", zap.Error(err))
		}
	}
	cfg := glue.DefaultConfig()
	if filename := f.Config; filename != "" {
		zap.L().Info("reading config", zap.String("filename", filename))
		var err error
		cfg, err = glue.LoadConfig(filename)
		if err != nil {
			zap.L().Fatal("problem reading config file", zap.String("filename", filename), zap.Error(err))
		}
	}
	go watcher.WatchServices(context.Background(), cfg.ClusterConfig.Store(svc))

	server.ListenAndServe()
}
