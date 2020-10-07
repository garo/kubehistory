package main

import (
	"github.com/golang/glog"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
//	"github.com/hako/durafmt"
//	"encoding/json"
	"k8s.io/client-go/kubernetes"
//	"io/ioutil"
//	"math"
	"k8s.io/client-go/tools/cache"
//	"net/http"
//	"fmt"
//	"os"
//	"strconv"
  "k8s.io/apimachinery/pkg/fields"
)



type Snapshotter struct {
	quit chan int
	client *kubernetes.Clientset
	persistentStorage PersistentStorage
	podStore cache.Store
	nodeStore cache.Store
}

func NewSnapshotter(client *kubernetes.Clientset) (*Snapshotter) {
	s := &Snapshotter{
		quit: make(chan int),
		client: client,
	}

	return s
}

func (s *Snapshotter) InitPods() {
	watchList := cache.NewListWatchFromClient(s.client.Core().RESTClient(), "pods", metav1.NamespaceAll, fields.Everything())
	store, controller := cache.NewInformer(
		watchList,
		&api.Pod{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func (obj interface{}) {
				pod := obj.(*api.Pod)
				s.persistentStorage.StorePod(REASON_SNAPSHOT, pod)
			},
			UpdateFunc: func (oldObj, obj interface{}) {
				pod := obj.(*api.Pod)
				s.persistentStorage.StorePod(REASON_SNAPSHOT, pod)
			},
			DeleteFunc: func (obj interface{}) {
				pod := obj.(*api.Pod)
				s.persistentStorage.StorePod(REASON_DELETED, pod)

			},
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)

	if !cache.WaitForCacheSync(stop, controller.HasSynced) {
		glog.Infof("Timed out waiting for caches to sync")
		return
	}

	s.podStore = store
}

func (s *Snapshotter) InitNodes() {
	watchList := cache.NewListWatchFromClient(s.client.Core().RESTClient(), "nodes", metav1.NamespaceAll, fields.Everything())
	store, controller := cache.NewInformer(
		watchList,
		&api.Node{},
		time.Minute*5,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func (obj interface{}) {
				node := obj.(*api.Node)
				s.persistentStorage.StoreNode(REASON_SNAPSHOT, node)
			},
			DeleteFunc: func (obj interface{}) {
				node := obj.(*api.Node)
				s.persistentStorage.StoreNode(REASON_DELETED, node)
			},
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)

	if !cache.WaitForCacheSync(stop, controller.HasSynced) {
		glog.Infof("Timed out waiting for caches to sync")
		return
	}

	s.nodeStore = store
}

func (s *Snapshotter) SetPersistentStorage(storage PersistentStorage) () {
	s.persistentStorage = storage
}

func (s *Snapshotter) Init() {

	glog.Infof("Snapshotter::Init")

	s.InitPods()
	glog.Infof("Snapshotter::Init - pods completed")
	s.InitNodes()
	glog.Infof("Snapshotter::Init - nodes completed")

	glog.Infof("Snapshotter init phase completed. There are currently %d pods and %d nodes", len(s.podStore.List()),len(s.nodeStore.List()))

	go func(sn *Snapshotter) {
		for {
			select {
			case <- sn.quit:
				return
			default:

				// We don't track updates on the nodes because there are so much of those. Instead
				// We periodically here do a snapshot
				for _, obj := range s.nodeStore.List() {
					node := obj.(*api.Node)
					sn.persistentStorage.StoreNode(REASON_SNAPSHOT, node)
				}

				// Pods don't need to be snapshotted here as we have defined an UpdateFunc to the store.

				sn.persistentStorage.Cleanup()
				time.Sleep(10 * time.Minute)
			}
		}
	}(s)

}