package k8s

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/klog/v2"
	"strings"
	"time"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"

	// This is here because of https://github.com/OpsLevel/kubectl-opslevel/issues/24
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	namespacesWereCached bool
	namespacesCache      []string
)

type NamespaceSelector struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

type Selector struct {
	ApiVersion string   `json:"apiVersion"`
	Kind       string   `json:"kind"`
	Namespaces []string `json:"namespaces,omitempty"`
	Labels     []string `json:"labels,omitempty"`
	Excludes   []string `json:"excludes,omitempty"`
}

func (selector *Selector) GetListOptions() metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: selector.LabelSelector(),
	}
}

func (selector *Selector) LabelSelector() string {
	var labels []string
	for key, value := range selector.Labels {
		labels = append(labels, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(labels, ",")
}

type Client struct {
	Client  kubernetes.Interface
	Dynamic dynamic.Interface
	Mapper  *restmapper.DeferredDiscoveryRESTMapper
}

// NewK8SClient
// This creates a wrapper which gives you an initialized and connected kubernetes client
// It then has a number of helper functions
func NewK8SClient() (*Client, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides).ClientConfig()
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
	return &Client{Client: client1, Dynamic: client2, Mapper: mapper}, nil
}

func (c *Client) GetMapping(selector Selector) (*meta.RESTMapping, error) {
	gv, gvErr := schema.ParseGroupVersion(selector.ApiVersion)
	if gvErr != nil {
		return nil, gvErr
	}
	gvk := gv.WithKind(selector.Kind)

	mapping, mappingErr := c.Mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if mappingErr != nil {
		return nil, mappingErr
	}

	return mapping, nil
}

func (c *Client) GetGVR(selector Selector) (*schema.GroupVersionResource, error) {
	mapping, err := c.GetMapping(selector)
	if err != nil {
		return nil, err
	}
	return &mapping.Resource, nil
}

func (c *Client) GetInformerFactory(resync time.Duration) dynamicinformer.DynamicSharedInformerFactory {
	return dynamicinformer.NewDynamicSharedInformerFactory(c.Dynamic, resync)
}

func (c *Client) GetNamespaces(selector Selector) ([]string, error) {
	if len(selector.Namespaces) > 0 {
		return selector.Namespaces, nil
	} else {
		if namespacesWereCached {
			return namespacesCache, nil
		}
		allNamespaces, err := c.GetAllNamespaces()
		if err != nil {
			return nil, err
		}
		namespacesWereCached = true
		namespacesCache = allNamespaces
		return namespacesCache, nil
	}
}

func (c *Client) GetAllNamespaces() ([]string, error) {
	var output []string
	resources, queryErr := c.Client.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if queryErr != nil {
		return output, queryErr
	}
	for _, resource := range resources.Items {
		output = append(output, resource.Name)
	}
	return output, nil
}

func (c *Client) Query(selector Selector) (output []unstructured.Unstructured, err error) {
	aggregator := func(resource unstructured.Unstructured) {
		output = append(output, resource)
	}
	namespaces, err := c.GetNamespaces(selector)
	if err != nil {
		return
	}
	mapping, err := c.GetMapping(selector)
	if err != nil {
		err = fmt.Errorf("%s \n\t please ensure you are using a valid `ApiVersion` and `Kind` found in `kubectl api-resources --verbs=\"get,list\"`", err)
		return
	}
	options := selector.GetListOptions()
	dr := c.Dynamic.Resource(mapping.Resource)
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		for _, namespace := range namespaces {
			if err = List(dr.Namespace(namespace), options, aggregator); err != nil {
				return
			}
		}
	} else {
		if err = List(dr, options, aggregator); err != nil {
			return
		}
	}
	return
}

func List(client dynamic.ResourceInterface, options metav1.ListOptions, aggregator func(resource unstructured.Unstructured)) (err error) {
	resources, err := client.List(context.Background(), options)
	if err != nil {
		return
	}
	for _, resource := range resources.Items {
		aggregator(resource)
	}
	return
}
