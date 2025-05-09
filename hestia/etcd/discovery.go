package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/daheige/hephfx/hestia"
)

var _ hestia.Discovery = (*etcdDiscovery)(nil)

type etcdDiscovery struct {
	client       *clientv3.Client
	prefix       string
	serviceList  map[string][]*hestia.Service
	disableWatch bool // disable watch
	mu           sync.RWMutex
}

// NewDiscovery create a registry interface instance
func NewDiscovery(endpoints []string, opts ...Option) (hestia.Discovery, error) {
	opt := &Options{
		endpoints:    endpoints,
		dialTimeout:  5 * time.Second,
		prefix:       "hestia/registry-etcd",
		disableWatch: true, // disable watch
	}

	for _, o := range opts {
		o(opt)
	}

	client, err := newEtcdClient(opt)
	if err != nil {
		return nil, err
	}

	e := &etcdDiscovery{
		client:       client,
		prefix:       opt.prefix,
		serviceList:  make(map[string][]*hestia.Service, 20),
		disableWatch: opt.disableWatch,
	}

	return e, nil
}

// GetServices returns a list of instances
// After we obtain the service instance, we can get the currently available service instance
// from the service list according to different strategies.
func (e *etcdDiscovery) GetServices(name string) ([]*hestia.Service, error) {
	var (
		services []*hestia.Service
		exist    bool
	)
	if !e.disableWatch {
		e.mu.RLock()
		services, exist = e.serviceList[name]
		e.mu.RUnlock()
	}

	if !exist {
		var err error
		services, err = e.getServicesByName(name)
		if err != nil {
			return nil, err
		}
		if len(services) == 0 {
			return nil, hestia.ErrServicesNotFound
		}

		if !e.disableWatch {
			e.mu.Lock()
			e.serviceList[name] = services
			e.mu.Unlock()
			go func() {
				e.watch(name)
			}()
		} else {
			e.serviceList[name] = services
		}
	}

	return services, nil
}

// Get returns an available service instance based on the specified service selection strategy.
// the selection strategy is RoundRobinHandler
func (e *etcdDiscovery) Get(name string, strategyHandler ...hestia.StrategyHandler) (*hestia.Service, error) {
	services, err := e.GetServices(name)
	if err != nil {
		return nil, err
	}

	var handler = hestia.RoundRobinHandler
	if len(strategyHandler) > 0 && strategyHandler[0] != nil {
		handler = strategyHandler[0]
	}

	service := handler(services)
	return service, nil
}

// String returns discovery name
func (e *etcdDiscovery) String() string {
	return "etcd"
}

// listen services change
func (e *etcdDiscovery) watch(name string) {
	key := fmt.Sprintf("%s/%s", e.prefix, name)
	respChan := e.client.Watch(context.Background(), key, clientv3.WithPrefix())
	log.Printf("watch etcd prefix:%v\n", key)
	for resp := range respChan {
		for _, event := range resp.Events {
			switch event.Type {
			case clientv3.EventTypePut, clientv3.EventTypeDelete: // 修改或新增
				// 数据有变更，需要重新加载
				e.reload(key, name)
			}
		}
	}
}

func (e *etcdDiscovery) reload(key string, name string) {
	go func() {
		e.mu.Lock()
		defer e.mu.Unlock()

		log.Printf("reload etcd prefix:%s services", key)
		services, err := e.getServicesByName(name)
		if err != nil {
			log.Printf("reload etcd prefix:%s services error:%v", key, err)
			return
		}

		e.serviceList[name] = services
	}()
}

func (e *etcdDiscovery) getServicesByName(name string) ([]*hestia.Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	key := fmt.Sprintf("%s/%s", e.prefix, name)
	resp, err := e.client.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	// 获取所有的服务实例列表
	services := make([]*hestia.Service, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		serviceEntry := &hestia.Service{}
		err = json.Unmarshal(kv.Value, serviceEntry)
		if err != nil {
			log.Printf("unmarshal service failed,error:%v", err)
			continue
		}

		services = append(services, serviceEntry)
	}

	return services, nil
}
