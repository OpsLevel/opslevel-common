package k8s

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type ControllerHandler func([]interface{})

type Controller struct {
	id       string
	factory  dynamicinformer.DynamicSharedInformerFactory
	queue    *workqueue.Type
	informer cache.SharedIndexInformer
	maxBatch int
	Channel  chan struct{}
	OnAdd    ControllerHandler
	OnUpdate ControllerHandler
	OnDelete ControllerHandler
}

type ControllerEventType string

const (
	ControllerEventTypeCreate ControllerEventType = "create"
	ControllerEventTypeUpdate ControllerEventType = "update"
	ControllerEventTypeDelete ControllerEventType = "delete"
)

type KubernetesEvent struct {
	Key  string
	Type ControllerEventType
}

func nullKubernetesControllerHandler(items []interface{}) {}

func (c *Controller) getLength() int {
	current := c.queue.Len()
	if current < c.maxBatch {
		return current
	} else {
		return c.maxBatch
	}
}

func (c *Controller) processNextItem() bool {
	var matchType ControllerEventType
	items := make([]interface{}, 0)
	length := c.getLength()
	for i := 0; i < length; i++ {
		item, quit := c.queue.Get()
		if quit {
			return false
		}
		event := item.(KubernetesEvent)
		if i == 0 {
			// First item determines what event type we process
			matchType = event.Type
		} else {
			if event.Type != matchType {
				c.queue.Add(item)
				c.queue.Done(item)
				continue
			}
		}
		obj, exists, getErr := c.informer.GetIndexer().GetByKey(event.Key)
		if getErr != nil {
			log.Warn().Msgf("error fetching object with key %s from informer cache: %v", event.Key, getErr)
			c.queue.Done(item)
			return true
		}
		if !exists {
			if matchType != ControllerEventTypeDelete {
				log.Warn().Msgf("object with key %s doesn't exist in informer cache", event.Key)
			}
			c.queue.Done(item)
			return true
		}
		c.queue.Done(item)
		items = append(items, obj)
	}
	switch matchType {
	case ControllerEventTypeCreate:
		c.OnAdd(items)
	case ControllerEventTypeUpdate:
		c.OnUpdate(items)
	case ControllerEventTypeDelete:
		c.OnDelete(items)
	}

	return true
}
func (c *Controller) mainloop() {
	for c.processNextItem() {
	}
}

func (c *Controller) Start(workers int) {
	defer runtime.HandleCrash()
	defer c.queue.ShutDown()
	c.factory.Start(c.Channel) // Starts all informers

	for _, ready := range c.factory.WaitForCacheSync(c.Channel) {
		if !ready {
			runtime.HandleError(fmt.Errorf("[%s] Timed out waiting for caches to sync", c.id))
			return
		}
		log.Info().Msgf("[%s] Informer is ready and synced", c.id)
	}
	if workers < 1 {
		workers = 1
	}
	for i := 0; i < workers; i++ {
		log.Info().Msgf("[%s] Creating worker #%d", c.id, i+1)
		go wait.Until(c.mainloop, time.Second, c.Channel)
	}

	<-c.Channel
}

func NewController(gvr schema.GroupVersionResource, resyncInterval time.Duration, maxBatch int) *Controller {
	k8sClient, err := NewK8SClient()
	if err != nil {
		panic(err)
	}
	queue := workqueue.New()
	factory := k8sClient.GetInformerFactory(resyncInterval)
	informer := factory.ForResource(gvr).Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			var err error
			var item KubernetesEvent
			item.Key, err = cache.MetaNamespaceKeyFunc(obj)
			item.Type = ControllerEventTypeCreate
			if err == nil {
				log.Debug().Msgf("Queuing event: %+v", item)
				queue.Add(item)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			var err error
			var item KubernetesEvent
			item.Key, err = cache.MetaNamespaceKeyFunc(old)
			item.Type = ControllerEventTypeUpdate
			if err == nil {
				log.Debug().Msgf("Queuing event: %+v", item)
				queue.Add(item)
			}
		},
		DeleteFunc: func(obj interface{}) {
			var err error
			var item KubernetesEvent
			item.Key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			item.Type = ControllerEventTypeDelete
			if err == nil {
				log.Debug().Msgf("Queuing event: %+v", item)
				queue.Add(item)
			}
		},
	})
	return &Controller{
		id:       fmt.Sprintf("%s/%s/%s", gvr.Group, gvr.Version, gvr.Resource),
		queue:    queue,
		factory:  factory,
		informer: informer,
		maxBatch: maxBatch,
		Channel:  make(chan struct{}),
		OnAdd:    nullKubernetesControllerHandler,
		OnUpdate: nullKubernetesControllerHandler,
		OnDelete: nullKubernetesControllerHandler,
	}
}
