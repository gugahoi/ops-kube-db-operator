package controller

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/client-go/util/workqueue"

	"k8s.io/apimachinery/pkg/util/runtime"
	informercorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	listercorev1 "k8s.io/client-go/listers/core/v1"
	apicorev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/MYOB-Technology/dataform/pkg/db"
	"github.com/MYOB-Technology/dataform/pkg/service"
)

// RDSController is a controller for RDS DBs.
type RDSController struct {
	// cmGetter is a configMap getter
	cmGetter corev1.ConfigMapsGetter
	// cmLister is a secondary cache of configMaps used for lookups
	cmLister listercorev1.ConfigMapLister
	// cmSynces is a flag to indicate if the cache is synced
	cmSynced cache.InformerSynced
	// queue is where incoming work is placed - it handles de-dup and rate limiting
	queue workqueue.RateLimitingInterface
	// rds is how we interact with AWS RDS Service
	rds *db.Manager
}

// New instantiates an rds controller
func New(
	queue workqueue.RateLimitingInterface,
	client *kubernetes.Clientset,
	cmInformer informercorev1.ConfigMapInformer,
) *RDSController {

	c := &RDSController{
		cmGetter: client.CoreV1(),
		cmLister: cmInformer.Lister(),
		cmSynced: cmInformer.Informer().HasSynced,
		queue:    queue,
		rds:      db.NewManager(service.New("")),
	}

	cmInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: c.enqueue,
			UpdateFunc: func(old, new interface{}) {
				if !reflect.DeepEqual(old, new) {
					c.enqueue(new)
				}
			},
			DeleteFunc: c.enqueue,
		},
	)

	return c
}

// Run starts the controller
func (c *RDSController) Run(threadiness int, stopChan <-chan struct{}) {
	// do not allow panics to crash the controller
	defer runtime.HandleCrash()

	// shutdown the queue when done
	defer c.queue.ShutDown()

	log.Print("Starting RDS Controller")

	log.Print("waiting for cache to sync")
	if !cache.WaitForCacheSync(stopChan, c.cmSynced) {
		log.Print("timeout waiting for sync")
		return
	}
	log.Print("caches synced successfully")

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopChan)
	}

	// block until we are told to exit
	<-stopChan
}

func (c *RDSController) runWorker() {
	// process the next item in queue until it is empty
	for c.processNextWorkItem() {
	}
}

func (c *RDSController) processNextWorkItem() bool {
	// get next item from work queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}

	// indicate to queue when work is finished on a specific item
	defer c.queue.Done(key)

	err := c.processConfigMap(key.(string))
	if err == nil {
		// processed succesfully, lets forget item in queue and return success
		c.queue.Forget(key)
		return true
	}

	// There was an error processing the item, log and requeue
	runtime.HandleError(fmt.Errorf("%v", err))

	// Add item back in with a rate limited backoff
	c.queue.AddRateLimited(key)

	return true
}

func (c *RDSController) enqueue(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(fmt.Errorf("error obtaining key for enqueued object: %v", err))
	}
	c.queue.Add(key)
}

func (c *RDSController) processConfigMap(key string) error {
	// get resource name and namespace out of key
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("error splitting namespace/key from obj %s: %v", key, err)
	}
	instanceName := fmt.Sprintf("%s-%s", ns, name)
	cm, err := c.cmLister.ConfigMaps(ns).Get(name)
	if err != nil {
		// if we get here it is likely the ConfigMap has been deleted
		log.Printf("failed to retrieve up to date cm %s: %v - it has likely been deleted", key, err)
		// TODO: Delete RDS? Or do something smart about it
		_, err := c.rds.Delete(instanceName)
		if err != nil {
			return fmt.Errorf("error deleting RDS Instance %s: %v", key, err)
		}
		return nil
	}

	// if our annotation is not present, let's bail
	if cm.Annotations["gustavo.com.au/rds"] != "true" {
		return nil
	}
	newCmInf, _ := scheme.Scheme.DeepCopy(cm)
	newCm := newCmInf.(*apicorev1.ConfigMap)

	// we have cm that needs to be processed
	log.Printf("Processing: %s/%s", ns, name)

	if newCm.Data == nil || newCm.Data["ARN"] == "" {
		// if there is no ARN set in the CM, then we need to create a new RDS Instance
		// TODO: check if one already exists, possibly have a mechanism to import?
		log.Printf("Creating RDS Instance: %s", key)
		db, err := c.rds.Create(instanceName)
		if err != nil {
			return fmt.Errorf("Failed to create RDS Instance for %s: %v", key, err)
		}

		// store the ARN in the configmap
		if newCm.Data == nil {
			data := make(map[string]string)
			data["ARN"] = *db.ARN
			newCm.Data = data
		} else {
			newCm.Data["ARN"] = *db.ARN
		}

		log.Printf("Updating %s with ARN %s", key, *db.ARN)
		_, err = c.cmGetter.ConfigMaps(ns).Update(newCm)
		if err != nil {
			return fmt.Errorf("failed to update cm %s: %v", key, err)
		}
	} else {
		// if there is an ARN set, we might need to update
		log.Printf("Updating RDS Instance: %s (doing nothing for now)", key)
	}

	log.Printf("Finished updating %s", key)
	return nil
}
