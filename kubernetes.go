package opslevel_common

import (
	"github.com/go-logr/logr"
	"k8s.io/klog/v2"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"

	// This is here because of https://github.com/OpsLevel/kubectl-opslevel/issues/24
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ClientWrapper struct {
	Client  kubernetes.Interface
	Dynamic dynamic.Interface
	Mapper  *restmapper.DeferredDiscoveryRESTMapper
}

func getKubernetesConfig() (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides).ClientConfig()
	if err != nil {
		return nil, err
	}
	return config, nil
}

// CreateKubernetesClient
// This creates a wrapper which gives you an initialized and connected kubernetes client and dynamic client with restmapper
func CreateKubernetesClient() (*ClientWrapper, error) {
	config, err := getKubernetesConfig()
	if err != nil {
		return nil, err
	}

	client1, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	client2, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	dc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	// Suppress k8s client-go logs
	klog.SetLogger(logr.Discard())
	return &ClientWrapper{Client: client1, Dynamic: client2, Mapper: mapper}, nil
}
